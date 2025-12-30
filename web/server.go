package web

import (
	"embed"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"almono/api"
)

//go:embed templates/*.tmpl templates/immutable.css
var templatesFS embed.FS

type Server struct {
	templates *template.Template
	svc       *api.Service
	css       template.CSS
	pageSize  int
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

type OutputRow struct {
	LineNum    int
	Content    string
	ShowSpacer bool
}

type ResponseView struct {
	CSS         template.CSS
	RequestID   int64
	Prompt      string
	Status      string
	Lines       []OutputRow
	PageNumbers []PageNumber
	Page        int
	Pages       int
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
	return &Server{
		templates: tmpl,
		svc:       svc,
		pageSize:  10,
		css:       template.CSS(rawCSS),
	}, nil
}

func (s *Server) HandleRequests(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/requests/" || r.URL.Path == "/requests" || r.URL.Path == "/" {
		s.HandleList(w, r)
		return
	}
	if strings.HasPrefix(r.URL.Path, "/requests/") {
		s.HandleResponse(w, r)
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

func (s *Server) HandleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
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

func (s *Server) HandleResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/requests/")
	idStr = strings.TrimSuffix(idStr, "/")
	if idStr == "" || strings.Contains(idStr, "/") {
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
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	page := parseInt(r.URL.Query().Get("page"), 1)
	linesPerPage := 50
	offset := (page - 1) * linesPerPage

	lines, total, err := s.svc.GetOutputLines(r.Context(), id, linesPerPage, offset)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	pages := (total + linesPerPage - 1) / linesPerPage
	if pages < 1 {
		pages = 1
	}

	// reverse lines to show oldest first (chronological order)
	outputRows := make([]OutputRow, 0, len(lines))
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		outputRows = append(outputRows, OutputRow{
			LineNum:    line.LineNum,
			Content:    line.Content,
			ShowSpacer: i > 0,
		})
	}

	pageNumbers := make([]PageNumber, 0, 5)
	for i := 1; i <= 5; i++ {
		pageNumbers = append(pageNumbers, PageNumber{Value: i, HasSpacer: i < 5})
	}

	data := ResponseView{
		CSS:         s.css,
		RequestID:   req.ID,
		Prompt:      req.Prompt,
		Status:      req.Status,
		Lines:       outputRows,
		PageNumbers: pageNumbers,
		Page:        page,
		Pages:       pages,
	}
	if err := s.templates.ExecuteTemplate(w, "response", data); err != nil {
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
