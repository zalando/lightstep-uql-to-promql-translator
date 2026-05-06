package server

import (
	"embed"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/zalando/lightstep-uql-to-promql-translator/pkg/model"
)

const (
	MaxRequestBodySize = 1 << 20 // 1 MB
)

//go:embed static/*
var staticFiles embed.FS

type TranslateRequest struct {
	Query string `json:"query"`
}

type ErrorResponse struct {
	Status       string `json:"status"`
	SourceIndex  int    `json:"source_index"`
	SourceLength int    `json:"source_length"`
}

type TranslateResponse struct {
	PromQL string         `json:"promql"`
	Error  *ErrorResponse `json:"error"`
}

type ServerTranslateFunc func(string) (string, *model.Error)

type Server struct {
	addr          string
	translateFunc ServerTranslateFunc
}

func New(addr string, translateFunc ServerTranslateFunc) *Server {
	return &Server{
		addr:          addr,
		translateFunc: translateFunc,
	}
}

func jsonMarshalString(data string) string {
	result, _ := json.Marshal(data)
	return string(result)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	data, err := staticFiles.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "Failed to load page", http.StatusInternalServerError)
		log.Printf("Failed to read index.html: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}

func (s *Server) handleTranslate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, MaxRequestBodySize)

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req TranslateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	log.Printf("got query: %s", jsonMarshalString(req.Query))
	promqlQuery, translationErr := s.translateFunc(req.Query)

	response := TranslateResponse{
		PromQL: promqlQuery,
		Error:  nil,
	}

	if translationErr != nil {
		response.Error = &ErrorResponse{
			Status:       translationErr.Status,
			SourceIndex:  translationErr.SourceIndex,
			SourceLength: translationErr.SourceLength,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[1:]
	data, err := staticFiles.ReadFile(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	contentType := "text/plain"
	if len(path) > 4 && path[len(path)-4:] == ".css" {
		contentType = "text/css"
	}

	w.Header().Set("Content-Type", contentType)
	w.Write(data)
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.handleIndex)
	http.HandleFunc("/static/", s.handleStatic)
	http.HandleFunc("/api/translate", s.handleTranslate)

	server := &http.Server{
		Addr:              s.addr,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20, // 1 MB
	}

	log.Printf("Starting server on %s", s.addr)
	return server.ListenAndServe()
}
