package web

import (
	"bytes"
	"embed"
	"html/template"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"

	"almono/api"

	"github.com/fogleman/gg"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
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
	CSS          template.CSS
	RequestID    int64
	Prompt       string
	Status       string
	Lines        []OutputRow
	FinalMessage string
	PageNumbers  []PageNumber
	Page         int
	Pages        int
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
		// check if it's an image request
		if strings.HasSuffix(r.URL.Path, "/image") || strings.HasSuffix(r.URL.Path, "/image/") {
			s.HandleImage(w, r)
			return
		}
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
		log.Printf("CreateRequest failed: %v", err)
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

	lines, _, err := s.svc.GetOutputLines(r.Context(), id, 1000, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// get latest status (reasoning only) for display while processing
	// lines are ordered DESC (newest first)
	var latestStatus string
	for _, line := range lines {
		if line.LineType == "reasoning" {
			latestStatus = line.Content
			break
		}
	}
	var statusRows []OutputRow
	if latestStatus != "" {
		statusRows = append(statusRows, OutputRow{Content: latestStatus})
	}

	data := ResponseView{
		CSS:       s.css,
		RequestID: req.ID,
		Prompt:    req.Prompt,
		Status:    req.Status,
		Lines:     statusRows,
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

// ----------------------------------
// Terminal-style image generation
// ----------------------------------

const (
	termFontPath  = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"
	termFontBold  = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf"
	termFontSize  = 14.0
	termPadding   = 20.0
	termBgColor   = "#1e1e1e"
	termFgColor   = "#d4d4d4"
	termCodeColor = "#ce9178"
	termWidth     = 550
	termLineH     = 20.0
)

func (s *Server) HandleImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// extract ID from /requests/{id}/image
	path := strings.TrimPrefix(r.URL.Path, "/requests/")
	path = strings.TrimSuffix(path, "/image/")
	path = strings.TrimSuffix(path, "/image")
	id, err := strconv.ParseInt(path, 10, 64)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	_, ok, err := s.svc.GetRequest(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// get all output lines
	lines, _, err := s.svc.GetOutputLines(r.Context(), id, 1000, 0)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// collect all message and error lines for image
	// lines are ordered DESC (newest first), so reverse to get chronological order
	var messages []string
	for i := len(lines) - 1; i >= 0; i-- {
		if lines[i].LineType == "message" || lines[i].LineType == "error" {
			messages = append(messages, lines[i].Content)
		}
	}

	// generate terminal image with all messages aggregated
	img, err := renderTerminalImage(messages)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	png.Encode(w, img.Image())
}

// ----------------------------------
// Styled text segment for markdown
// ----------------------------------

type styledSegment struct {
	text string
	bold bool
	code bool
}

func renderTerminalImage(lines []string) (*gg.Context, error) {
	content := strings.Join(lines, "\n")

	// parse markdown and extract styled segments
	segments := parseMarkdown(content)

	// wrap into lines with style info
	wrapped := wrapStyledLines(segments, 50)

	// calculate image height
	height := termPadding*2 + float64(len(wrapped))*termLineH
	if height < 100 {
		height = 100
	}

	dc := gg.NewContext(termWidth, int(height))

	// draw background
	dc.SetHexColor(termBgColor)
	dc.Clear()

	// draw styled text
	y := termPadding + termFontSize
	for _, line := range wrapped {
		x := termPadding
		for _, seg := range line {
			// load appropriate font
			fontPath := termFontPath
			if seg.bold {
				fontPath = termFontBold
			}
			dc.LoadFontFace(fontPath, termFontSize)

			// set color
			if seg.code {
				dc.SetHexColor(termCodeColor)
			} else {
				dc.SetHexColor(termFgColor)
			}

			dc.DrawString(seg.text, x, y)
			w, _ := dc.MeasureString(seg.text)
			x += w
		}
		y += termLineH
	}

	return dc, nil
}

func parseMarkdown(content string) []styledSegment {
	var segments []styledSegment
	source := []byte(content)
	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	// walk AST and extract styled segments
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch node := n.(type) {
		case *ast.Text:
			txt := string(node.Segment.Value(source))
			bold := isInStrong(n)
			code := isInCode(n)
			if txt != "" {
				segments = append(segments, styledSegment{text: txt, bold: bold, code: code})
			}
			if node.SoftLineBreak() || node.HardLineBreak() {
				segments = append(segments, styledSegment{text: "\n"})
			}
		case *ast.CodeSpan:
			var buf bytes.Buffer
			for c := node.FirstChild(); c != nil; c = c.NextSibling() {
				if t, ok := c.(*ast.Text); ok {
					buf.Write(t.Segment.Value(source))
				}
			}
			segments = append(segments, styledSegment{text: buf.String(), code: true})
			return ast.WalkSkipChildren, nil
		case *ast.Paragraph:
			if n.PreviousSibling() != nil {
				segments = append(segments, styledSegment{text: "\n"})
			}
		}
		return ast.WalkContinue, nil
	})

	return segments
}

func isInStrong(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.Emphasis); ok {
			if p.(*ast.Emphasis).Level == 2 {
				return true
			}
		}
	}
	return false
}

func isInCode(n ast.Node) bool {
	for p := n.Parent(); p != nil; p = p.Parent() {
		if _, ok := p.(*ast.CodeSpan); ok {
			return true
		}
	}
	return false
}

func wrapStyledLines(segments []styledSegment, maxChars int) [][]styledSegment {
	var result [][]styledSegment
	var currentLine []styledSegment
	lineLen := 0

	for _, seg := range segments {
		if seg.text == "\n" {
			result = append(result, currentLine)
			currentLine = nil
			lineLen = 0
			continue
		}

		words := strings.Fields(seg.text)
		for i, word := range words {
			wordLen := len(word)
			needSpace := i > 0 || (lineLen > 0 && len(currentLine) > 0)
			spaceLen := 0
			if needSpace {
				spaceLen = 1
			}

			if lineLen+spaceLen+wordLen > maxChars && lineLen > 0 {
				result = append(result, currentLine)
				currentLine = nil
				lineLen = 0
				needSpace = false
			}

			if needSpace {
				currentLine = append(currentLine, styledSegment{text: " ", bold: seg.bold, code: seg.code})
				lineLen++
			}
			currentLine = append(currentLine, styledSegment{text: word, bold: seg.bold, code: seg.code})
			lineLen += wordLen
		}
	}

	if len(currentLine) > 0 {
		result = append(result, currentLine)
	}

	return result
}
