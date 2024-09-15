package adapter_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/pkg/adapter"
)

type TestService struct{}

func (s *TestService) ValidMethod(arg1 string, arg2 int) (string, error) {
	return "success", nil
}

func (s *TestService) MethodWithError(arg1 string) (string, error) {
	return "", fmt.Errorf("method error")
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
})
