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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/app"
	st "github.com/hpc-gridware/go-clusterscheduler/cmd/sharetree/pkg/sharetree"
)

// Handler encapsulates HTTP handlers for the sharetree API
type Handler struct {
	App *app.App
}

// NewHandler creates a new API handler with the given app
func NewHandler(app *app.App) *Handler {
	return &Handler{App: app}
}

// GetSharetreeHandler returns the current sharetree.
func (h *Handler) GetSharetreeHandler(w http.ResponseWriter, r *http.Request) {
	// Calculate percentages before returning
	sharetree := h.App.CalculatePercentages(h.App.CurrentSharetree)

	response := ApiResponse{
		Message:   "Sharetree retrieved",
		Sharetree: sharetree,
		TempFile:  h.App.GetCurrentTempFileName(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// UpdateSharetreeHandler handles updating the sharetree.
func (h *Handler) UpdateSharetreeHandler(w http.ResponseWriter, r *http.Request) {
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate sharetree before updating
	if err := st.ValidateSharetree(req.Sharetree); err != nil {
		http.Error(w, "Invalid sharetree structure: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Additional validation for root node
	if err := h.App.ValidateRootNode(req.Sharetree); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Update global sharetree variable
	h.App.CurrentSharetree = req.Sharetree

	// Always save to temp file
	err := h.App.SaveSharetreeAsSGE(h.App.CurrentSharetree, h.App.TempFilePath)
	if err != nil {
		http.Error(w, "Error saving to temporary file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate percentages before returning
	sharetree := h.App.CalculatePercentages(h.App.CurrentSharetree)

	response := ApiResponse{
		Message:   "Sharetree updated and saved to temporary file",
		Sharetree: sharetree,
		TempFile:  h.App.GetCurrentTempFileName(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// LoadFromContentHandler loads a sharetree from provided content.
func (h *Handler) LoadFromContentHandler(w http.ResponseWriter, r *http.Request) {
	var req LoadFromContentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Load from SGE content
	loadedSharetree, err := h.App.LoadSharetreeFromSGE(req.Content)
	if err != nil {
		http.Error(w, "Error loading sharetree: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate the loaded sharetree
	if err := st.ValidateSharetree(loadedSharetree); err != nil {
		http.Error(w, "Invalid sharetree structure: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Update global sharetree
	h.App.CurrentSharetree = loadedSharetree

	// Always save to temp file
	err = h.App.SaveSharetreeAsSGE(h.App.CurrentSharetree, h.App.TempFilePath)
	if err != nil {
		http.Error(w, "Error saving to temporary file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate percentages before returning
	sharetree := h.App.CalculatePercentages(h.App.CurrentSharetree)

	response := ApiResponse{
		Message:   "Sharetree loaded from content and saved to temporary file",
		Sharetree: sharetree,
		TempFile:  h.App.GetCurrentTempFileName(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// SaveSharetreeHandler saves the current sharetree to a specified file.
func (h *Handler) SaveSharetreeHandler(w http.ResponseWriter, r *http.Request) {
	var req SaveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request format: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Filename == "" {
		http.Error(w, "Filename is required", http.StatusBadRequest)
		return
	}

	// Ensure filename has .sge extension
	if !strings.HasSuffix(req.Filename, ".sge") {
		req.Filename += ".sge"
	}

	// Save current sharetree to the specified file
	err := h.App.SaveSharetreeAsSGE(h.App.CurrentSharetree, req.Filename)
	if err != nil {
		http.Error(w, "Error saving sharetree: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate percentages before returning
	sharetree := h.App.CalculatePercentages(h.App.CurrentSharetree)

	response := ApiResponse{
		Message:   "Sharetree saved to " + req.Filename,
		Sharetree: sharetree,
		TempFile:  h.App.GetCurrentTempFileName(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// DownloadSharetreeHandler allows downloading the current sharetree.
func (h *Handler) DownloadSharetreeHandler(w http.ResponseWriter, r *http.Request) {
	// Generate SGE content from current sharetree
	content, err := st.ConvertToSgeFormat(h.App.CurrentSharetree)
	if err != nil {
		http.Error(w, "Error generating SGE content: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a filename based on the timestamp
	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("sharetree_%s.sge", timestamp)

	// Set headers for file download
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))

	// Write content to response
	if _, err := w.Write([]byte(content)); err != nil {
		log.Printf("Error writing download response: %v", err)
	}
}

// RefreshTempFileHandler creates a new empty temp file
func (h *Handler) RefreshTempFileHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.App.InitializeTempFile(); err != nil {
		http.Error(w, "Failed to refresh temporary file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Calculate percentages before returning
	sharetree := h.App.CalculatePercentages(h.App.CurrentSharetree)

	response := ApiResponse{
		Message:   "Created new temporary sharetree file",
		Sharetree: sharetree,
		TempFile:  h.App.GetCurrentTempFileName(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RegisterHandlers sets up all the API routes
func (h *Handler) RegisterHandlers() {
	http.HandleFunc("/api/getsharetree", h.GetSharetreeHandler)
	http.HandleFunc("/api/update", h.UpdateSharetreeHandler)
	http.HandleFunc("/api/loadfromcontent", h.LoadFromContentHandler)
	http.HandleFunc("/api/download", h.DownloadSharetreeHandler)
	http.HandleFunc("/api/refreshtemp", h.RefreshTempFileHandler)
}
