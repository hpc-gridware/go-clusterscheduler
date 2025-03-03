/*___INFO__MARK_BEGIN__*/
/*************************************************************************
*  Copyright 2025 HPC-Gridware GmbH
*
*  Licensed under the Apache License, Version 2.0 (the "License");
*  you may not use this file except in compliance with the License.
*  You may obtain a copy of the License at
*
*      http://www.apache.org/licenses/LICENSE-2.0
*
*  Unless required by applicable law or agreed to in writing, software
*  distributed under the License is distributed on an "AS IS" BASIS,
*  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
*  See the License for the specific language governing permissions and
*  limitations under the License.
*
************************************************************************/
/*___INFO__MARK_END__*/

package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/api"
	"github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/app"
	st "github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/sharetree"
)

func TestAPI(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Suite")
}

var _ = Describe("API Handlers", func() {
	var (
		handler *api.Handler
		testApp *app.App
	)

	BeforeEach(func() {
		var err error
		testApp, err = app.NewApp()
		Expect(err).NotTo(HaveOccurred())
		handler = api.NewHandler(testApp)
	})

	Describe("GetSharetreeHandler", func() {
		It("should return the current sharetree", func() {
			req := httptest.NewRequest("GET", "/api/getsharetree", nil)
			w := httptest.NewRecorder()

			handler.GetSharetreeHandler(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response api.ApiResponse
			err := json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Message).To(Equal("Sharetree retrieved"))
			Expect(response.Sharetree).NotTo(BeEmpty())
			Expect(response.TempFile).NotTo(BeEmpty())
		})
	})

	Describe("UpdateSharetreeHandler", func() {
		It("should update the sharetree with valid data", func() {
			validTree := []st.SharetreeNode{
				{ID: 0, Name: "Root", Type: 0, Shares: 1, ChildNodes: "1"},
				{ID: 1, Name: "TestNode", Type: 0, Shares: 100, ChildNodes: "NONE"},
			}

			reqBody := api.UpdateRequest{Sharetree: validTree}
			jsonData, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest("POST", "/api/update", bytes.NewBuffer(jsonData))
			w := httptest.NewRecorder()

			handler.UpdateSharetreeHandler(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response api.ApiResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Message).To(ContainSubstring("updated"))
			Expect(response.Sharetree).To(HaveLen(2))
		})

		It("should reject invalid sharetree data", func() {
			// Missing required Root node
			invalidTree := []st.SharetreeNode{
				{ID: 1, Name: "TestNode", Type: 0, Shares: 100, ChildNodes: "NONE"},
			}

			reqBody := api.UpdateRequest{Sharetree: invalidTree}
			jsonData, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest("POST", "/api/update", bytes.NewBuffer(jsonData))
			w := httptest.NewRecorder()

			handler.UpdateSharetreeHandler(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("LoadFromContentHandler", func() {
		It("should load a valid sharetree from content", func() {
			validContent := `id=0
name=Root
type=0
shares=1
childnodes=1
id=1
name=TestNode
type=0
shares=100
childnodes=NONE`

			reqBody := api.LoadFromContentRequest{Content: validContent}
			jsonData, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest("POST", "/api/loadfromcontent", bytes.NewBuffer(jsonData))
			w := httptest.NewRecorder()

			handler.LoadFromContentHandler(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))

			var response api.ApiResponse
			err = json.Unmarshal(w.Body.Bytes(), &response)
			Expect(err).NotTo(HaveOccurred())

			Expect(response.Message).To(ContainSubstring("loaded"))
			Expect(response.Sharetree).To(HaveLen(2))
		})

		It("should reject invalid content", func() {
			invalidContent := `invalid content`

			reqBody := api.LoadFromContentRequest{Content: invalidContent}
			jsonData, err := json.Marshal(reqBody)
			Expect(err).NotTo(HaveOccurred())

			req := httptest.NewRequest("POST", "/api/loadfromcontent", bytes.NewBuffer(jsonData))
			w := httptest.NewRecorder()

			handler.LoadFromContentHandler(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
		})
	})

	Describe("DownloadSharetreeHandler", func() {
		It("should provide the current sharetree for download", func() {
			req := httptest.NewRequest("GET", "/api/download", nil)
			w := httptest.NewRecorder()

			handler.DownloadSharetreeHandler(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Disposition")).To(ContainSubstring("attachment"))
			Expect(w.Header().Get("Content-Type")).To(Equal("text/plain"))
			Expect(w.Body.String()).To(ContainSubstring("id=0"))
			Expect(w.Body.String()).To(ContainSubstring("name=Root"))
		})
	})
})
