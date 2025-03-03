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

package api

import (
	st "github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/sharetree"
)

// ApiResponse represents a response from the API.
type ApiResponse struct {
	Message   string             `json:"message"`
	Sharetree []st.SharetreeNode `json:"sharetree"`
	TempFile  string             `json:"tempFile"`
}

// UpdateRequest represents a request to update the sharetree.
type UpdateRequest struct {
	Sharetree []st.SharetreeNode `json:"sharetree"`
}

// LoadFromFileRequest represents a request to load a sharetree from a file.
type LoadFromFileRequest struct {
	Filename string `json:"filename"`
}

// LoadFromContentRequest represents a request to load a sharetree from content.
type LoadFromContentRequest struct {
	Content string `json:"content"`
}

// SaveRequest represents a request to save the current sharetree to a file.
type SaveRequest struct {
	Filename string `json:"filename"`
}
