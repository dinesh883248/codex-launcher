package api

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type requestHandler struct {
	svc *Service
}

type createRequestPayload struct {
	Prompt string `json:"prompt"`
}

type listResponse struct {
	Requests []Request `json:"requests"`
	Page     int       `json:"page"`
	Pages    int       `json:"pages"`
	Total    int       `json:"total"`
}

func NewRequestHandler(svc *Service) http.Handler {
	return &requestHandler{svc: svc}
}

func (h *requestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.handleList(w, r)
	case http.MethodPost:
		h.handleCreate(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (h *requestHandler) handleList(w http.ResponseWriter, r *http.Request) {
	page := parseInt(r.URL.Query().Get("page"), 1)
	limit := parseInt(r.URL.Query().Get("limit"), 10)
	result, err := h.svc.ListRequests(r.Context(), page, limit)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(listResponse{
		Requests: result.Requests,
		Page:     result.Page,
		Pages:    result.Pages,
		Total:    result.Total,
	})
}

func (h *requestHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var payload createRequestPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	req, err := h.svc.CreateRequest(r.Context(), payload.Prompt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(req)
}

func parseInt(val string, fallback int) int {
	if val == "" {
		return fallback
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		return fallback
	}
	return num
}
