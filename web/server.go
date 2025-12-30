package web

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"embed"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"strconv"
	"strings"
	"time"

	"almono/api"
)

//go:embed templates/*.tmpl templates/immutable.css static/*
var templatesFS embed.FS

type Server struct {
	templates      *template.Template
	svc            *api.Service
	css            template.CSS
	pageSize       int
	staticFS       http.FileSystem
	cssVer         string
	jsVer          string
	screenshotPath string
	castPath       string
}

type RequestRow struct {
	ID         int64
	Display    string
	URL        string
	ShowSpacer bool
}

type PageNumber struct {
	Value     int
	HasSpacer bool
}

type ListView struct {
	CSS         template.CSS
	Requests    []RequestRow
	PageNumbers []PageNumber
	Page        int
	Pages       int
}

type CreateView struct {
	CSS template.CSS
}

type LivestreamView struct {
	CSS        template.CSS
	CSSVersion string
	JSVersion  string
	CastURL    string
	Message    string
	Streaming  bool
}

func NewServer(svc *api.Service) (*Server, error) {
	tmpl, err := template.ParseFS(templatesFS, "templates/*.tmpl")
	if err != nil {
		return nil, err
	}
	rawCSS, err := templatesFS.ReadFile("templates/immutable.css")
	if err != nil {
		return nil, err
	}
	staticRoot, err := fs.Sub(templatesFS, "static")
	if err != nil {
		return nil, err
	}
	cssVer, err := assetHash("static/asciinema-player.css")
	if err != nil {
		return nil, err
	}
	jsVer, err := assetHash("static/asciinema-player.min.js")
	if err != nil {
		return nil, err
	}
	return &Server{
		templates: tmpl,
		svc:       svc,
		pageSize:  10,
		css:       template.CSS(rawCSS),
		staticFS:  http.FS(staticRoot),
		cssVer:    cssVer,
		jsVer:     jsVer,
	}, nil
}

func (s *Server) StaticHandler() http.Handler {
	return http.FileServer(s.staticFS)
}

func (s *Server) SetScreenshotPath(path string) {
	s.screenshotPath = path
}

func (s *Server) SetCastPath(path string) {
	s.castPath = path
}

func (s *Server) HandleRequests(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/requests/" || r.URL.Path == "/requests" || r.URL.Path == "/" {
		s.HandleList(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/requests/") {
		s.HandleRequestCast(w, r)
		return
	}
	w.WriteHeader(http.StatusNotFound)
}

func (s *Server) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	page := parseInt(r.URL.Query().Get("page"), 1)
	result, err := s.svc.ListRequests(r.Context(), page, s.pageSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	rows := make([]RequestRow, 0, len(result.Requests))
	for i, req := range result.Requests {
		url := "/requests/" + strconv.FormatInt(req.ID, 10) + "/"
		rows = append(rows, RequestRow{
			ID:         req.ID,
			Display:    req.Prompt,
			URL:        url,
			ShowSpacer: i < len(result.Requests)-1,
		})
	}
	pageNumbers := make([]PageNumber, 0, 5)
	for i := 1; i <= 5; i++ {
		pageNumbers = append(pageNumbers, PageNumber{Value: i, HasSpacer: i < 5})
	}
	data := ListView{
		CSS:         s.css,
		Requests:    rows,
		PageNumbers: pageNumbers,
		Page:        result.Page,
		Pages:       result.Pages,
	}
	if err := s.templates.ExecuteTemplate(w, "request_list", data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleLivestream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	view := LivestreamView{
		CSS:        s.css,
		CSSVersion: s.cssVer,
		JSVersion:  s.jsVer,
		CastURL:    "/stream",
		Streaming:  true,
	}
	if err := s.templates.ExecuteTemplate(w, "livestream", view); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.renderCreate(w)
	case http.MethodPost:
		s.handleCreatePost(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) HandleLivestreamScreenshot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.screenshotPath == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Content-Type", "image/png")
	http.ServeFile(w, r, s.screenshotPath)
}

func (s *Server) HandleRequestCast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/requests/")
	idStr = strings.TrimSuffix(idStr, "/")
	if idStr == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if strings.Contains(idStr, "/") {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	req, ok, err := s.svc.GetRequest(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok || req.CastPath == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	view := LivestreamView{
		CSS:        s.css,
		CSSVersion: s.cssVer,
		JSVersion:  s.jsVer,
		CastURL:    "/casts/" + req.CastPath,
		Streaming:  false,
	}
	if err := s.templates.ExecuteTemplate(w, "livestream", view); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (s *Server) HandleStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if s.castPath == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	file, err := http.Dir("/").Open(s.castPath)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer file.Close()

	flusher, ok := w.(http.Flusher)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store, max-age=0")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	reader := bufio.NewReader(file)
	firstLine := true
	for {
		select {
		case <-r.Context().Done():
			return
		default:
		}

		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				time.Sleep(250 * time.Millisecond)
				continue
			}
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		if firstLine {
			firstLine = false
			var header struct {
				Version int `json:"version"`
				Term    struct {
					Cols int `json:"cols"`
					Rows int `json:"rows"`
				} `json:"term"`
			}
			if json.Unmarshal([]byte(line), &header) == nil && header.Version == 3 && header.Term.Cols > 0 && header.Term.Rows > 0 {
				payload, err := json.Marshal(struct {
					Cols int `json:"cols"`
					Rows int `json:"rows"`
				}{Cols: header.Term.Cols, Rows: header.Term.Rows})
				if err == nil {
					_, _ = fmt.Fprintf(w, "data: %s\n\n", payload)
					flusher.Flush()
					continue
				}
			}
		}
		_, _ = fmt.Fprintf(w, "data: %s\n\n", line)
		flusher.Flush()
	}
}

func (s *Server) handleCreatePost(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	prompt := r.FormValue("request")
	_, err := s.svc.CreateRequest(r.Context(), prompt)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/requests/", http.StatusSeeOther)
}

func (s *Server) renderCreate(w http.ResponseWriter) {
	if err := s.templates.ExecuteTemplate(w, "request_create", CreateView{CSS: s.css}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
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

func assetHash(path string) (string, error) {
	data, err := templatesFS.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum[:8]), nil
}
