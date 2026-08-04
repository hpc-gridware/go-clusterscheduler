package adapter_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/adapter"
	qconfcore "github.com/hpc-gridware/go-clusterscheduler/pkg/qconf/core"
)

type TestService struct{}

func (s *TestService) ValidMethod(arg1 string, arg2 int) (string, error) {
	return "success", nil
}

func (s *TestService) MethodWithError(arg1 string) (string, error) {
	return "", fmt.Errorf("method error")
}

// RunCommand is the raw argv passthrough that must never be reachable over the
// REST boundary. It mirrors the qconf wrapper's exported RunCommand.
func (s *TestService) RunCommand(args ...string) (string, error) {
	return "ran", nil
}

func postMethod(url, method string, args []interface{}) *http.Response {
	body, _ := json.Marshal(map[string]interface{}{"method": method, "args": args})
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	return resp
}

var _ = Describe("Adapter", func() {
	var (
		a      http.Handler
		server *httptest.Server
	)

	BeforeEach(func() {
		a = adapter.NewAdapter(&TestService{})
		server = httptest.NewServer(a)
	})

	AfterEach(func() {
		server.Close()
	})

	Context("ServeHTTP", func() {

		It("should handle valid requests", func() {
			reqBody := map[string]interface{}{
				"method": "ValidMethod",
				"args":   []interface{}{"test", 123},
			}
			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", server.URL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			var respBody string
			json.NewDecoder(resp.Body).Decode(&respBody)
			Expect(respBody).To(Equal("success"))
		})

		It("should handle invalid method names", func() {

			reqBody := map[string]interface{}{
				"method": "InvalidMethod",
				"args":   []interface{}{"test", 123},
			}
			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", server.URL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

		})

		It("should handle invalid arguments", func() {
			reqBody := map[string]interface{}{
				"method": "ValidMethod",
				"args":   []interface{}{"test"},
			}
			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", server.URL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("should handle method errors", func() {
			reqBody := map[string]interface{}{
				"method": "MethodWithError",
				"args":   []interface{}{"test"},
			}
			body, _ := json.Marshal(reqBody)
			req, _ := http.NewRequest("POST", server.URL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		})

		It("should handle invalid JSON payloads", func() {
			body := []byte(`{"method": "ValidMethod", "args": ["test", 123]`)
			req, _ := http.NewRequest("POST", server.URL, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()
		})

	})

	Context("method gating", func() {
		It("denies the raw RunCommand passthrough even by default", func() {
			resp := postMethod(server.URL, "RunCommand", []interface{}{"-mattr"})
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("omits denied methods from the /methods enumeration", func() {
			resp, err := http.Get(server.URL + "/methods")
			Expect(err).ToNot(HaveOccurred())
			defer resp.Body.Close()
			var body struct {
				Methods []struct {
					Name string `json:"name"`
				} `json:"methods"`
			}
			Expect(json.NewDecoder(resp.Body).Decode(&body)).ToNot(HaveOccurred())
			for _, m := range body.Methods {
				Expect(m.Name).ToNot(Equal("RunCommand"))
			}
		})
	})

	Context("explicit deny-list", func() {
		var denyServer *httptest.Server

		BeforeEach(func() {
			h := adapter.NewAdapterWithDeniedMethods(&TestService{}, []string{"MethodWithError"})
			denyServer = httptest.NewServer(h)
		})
		AfterEach(func() { denyServer.Close() })

		It("blocks an additionally-denied method", func() {
			resp := postMethod(denyServer.URL, "MethodWithError", []interface{}{"x"})
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})

		It("still allows other methods", func() {
			resp := postMethod(denyServer.URL, "ValidMethod", []interface{}{"test", 123})
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("exports a non-empty destructive-methods list", func() {
			Expect(adapter.DestructiveQConfMethods).To(ContainElement("ShutdownMasterDaemon"))
		})

		// Regression guard: every non-CRUD control method on the QConf
		// interface (Shutdown*/Kill*/Clean*/Clear*) must be in the deny list.
		// This would have caught the ClearShareTreeUsage omission and fails
		// the build if a future destructive method is added without denying it.
		It("denies every destructive QConf control method", func() {
			denied := map[string]bool{}
			for _, m := range adapter.DestructiveQConfMethods {
				denied[m] = true
			}
			iface := reflect.TypeOf((*qconfcore.QConf)(nil)).Elem()
			for i := 0; i < iface.NumMethod(); i++ {
				name := iface.Method(i).Name
				for _, prefix := range []string{"Shutdown", "Kill", "Clean", "Clear"} {
					if strings.HasPrefix(name, prefix) {
						Expect(denied[name]).To(BeTrue(),
							"destructive QConf method %q must be in adapter.DestructiveQConfMethods", name)
					}
				}
			}
		})
	})

	Context("explicit allow-list", func() {
		var allowServer *httptest.Server

		BeforeEach(func() {
			h := adapter.NewAdapterWithAllowedMethods(&TestService{}, []string{"ValidMethod"})
			allowServer = httptest.NewServer(h)
		})
		AfterEach(func() { allowServer.Close() })

		It("allows a method on the list", func() {
			resp := postMethod(allowServer.URL, "ValidMethod", []interface{}{"test", 123})
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})

		It("rejects a method not on the list", func() {
			resp := postMethod(allowServer.URL, "MethodWithError", []interface{}{"x"})
			defer resp.Body.Close()
			Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
		})
	})
})
