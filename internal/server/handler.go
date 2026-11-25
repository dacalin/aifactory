package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"strings"

	projectdomain "github.com/dacalin/aifactory/internal/app/project"
	"github.com/dacalin/aifactory/internal/model"
	"github.com/dacalin/aifactory/internal/server/dialog"
)

type Handler struct {
	projects Projects
	tasks    Tasks
	picker   dialog.FolderPicker
	assets   fs.FS
}

func NewHandler(projects Projects, tasks Tasks, picker dialog.FolderPicker, assets fs.FS) http.Handler {
	handler := &Handler{projects: projects, tasks: tasks, picker: picker, assets: assets}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", handler.health)
	mux.HandleFunc("GET /api/projects", handler.listProjects)
	mux.HandleFunc("POST /api/projects", handler.saveProject)
	mux.HandleFunc("GET /api/projects/{id}", handler.getProject)
	mux.HandleFunc("PATCH /api/projects/{id}", handler.updateProject)
	mux.HandleFunc("GET /api/projects/{id}/branches", handler.listProjectBranches)
	mux.HandleFunc("GET /api/projects/{id}/tasks", handler.listProjectTasks)
	mux.HandleFunc("POST /api/projects/{id}/tasks", handler.createProjectTask)
	mux.HandleFunc("POST /api/folder-selection", handler.selectFolder)
	mux.Handle("/", handler.frontend())
	return protectLocalBrowser(mux)
}

func (handler *Handler) health(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

func (handler *Handler) listProjects(response http.ResponseWriter, request *http.Request) {
	projects, err := handler.projects.List(request.Context())
	if err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}
	writeJSON(response, http.StatusOK, projects)
}

func (handler *Handler) getProject(response http.ResponseWriter, request *http.Request) {
	project, err := handler.projects.Get(request.Context(), request.PathValue("id"))
	if errors.Is(err, projectdomain.ErrProjectNotFound) {
		writeError(response, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}
	writeJSON(response, http.StatusOK, project)
}

func (handler *Handler) updateProject(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, 8*1024)
	var input struct {
		BaseBranch string `json:"base_branch"`
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil || strings.TrimSpace(input.BaseBranch) == "" {
		writeError(response, http.StatusBadRequest, errors.New("request must contain a base_branch"))
		return
	}
	project, err := handler.projects.SetBaseBranch(request.Context(), request.PathValue("id"), strings.TrimSpace(input.BaseBranch))
	if errors.Is(err, projectdomain.ErrProjectNotFound) {
		writeError(response, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(response, http.StatusOK, project)
}

func (handler *Handler) listProjectBranches(response http.ResponseWriter, request *http.Request) {
	branches, err := handler.projects.ListBranches(request.Context(), request.PathValue("id"))
	if errors.Is(err, projectdomain.ErrProjectNotFound) {
		writeError(response, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string][]string{"branches": branches})
}

func (handler *Handler) saveProject(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, 64*1024)
	var input projectRequest
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(response, http.StatusBadRequest, errors.New("request must contain valid project JSON"))
		return
	}
	var project model.Project
	var created bool
	var err error
	switch input.Operation {
	case "create":
		if strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.ParentPath) == "" {
			writeError(response, http.StatusBadRequest, errors.New("project name and parent folder are required"))
			return
		}
		name := strings.TrimSpace(input.Name)
		project, created, err = handler.projects.Create(request.Context(), name, filepath.Join(input.ParentPath, name))
	case "open":
		if strings.TrimSpace(input.Path) == "" {
			writeError(response, http.StatusBadRequest, errors.New("select a project folder"))
			return
		}
		project, created, err = handler.projects.Open(request.Context(), input.Path)
	default:
		writeError(response, http.StatusBadRequest, errors.New("operation must be create or open"))
		return
	}
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, err)
		return
	}
	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	writeJSON(response, status, project)
}

func (handler *Handler) selectFolder(response http.ResponseWriter, request *http.Request) {
	if handler.picker == nil {
		writeError(response, http.StatusNotImplemented, errors.New("native folder selection is unavailable"))
		return
	}
	var input struct {
		Operation string `json:"operation"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(response, request.Body, 1024)).Decode(&input); err != nil {
		writeError(response, http.StatusBadRequest, errors.New("request must specify a folder selection operation"))
		return
	}
	title := "Selecciona una carpeta de proyecto"
	if input.Operation == "create" {
		title = "Selecciona dónde crear el proyecto"
	} else if input.Operation != "open" {
		writeError(response, http.StatusBadRequest, errors.New("operation must be create or open"))
		return
	}
	selectedPath, err := handler.picker.SelectFolder(request.Context(), title)
	if errors.Is(err, dialog.ErrCancelled) {
		writeJSON(response, http.StatusOK, map[string]any{"cancelled": true})
		return
	}
	if err != nil {
		writeError(response, http.StatusServiceUnavailable, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]string{"path": selectedPath})
}

func (handler *Handler) listProjectTasks(response http.ResponseWriter, request *http.Request) {
	tasks, err := handler.tasks.ListByProject(request.Context(), request.PathValue("id"))
	if errors.Is(err, projectdomain.ErrProjectNotFound) {
		writeError(response, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(response, http.StatusInternalServerError, err)
		return
	}
	writeJSON(response, http.StatusOK, tasks)
}

func (handler *Handler) createProjectTask(response http.ResponseWriter, request *http.Request) {
	request.Body = http.MaxBytesReader(response, request.Body, 64*1024)
	var input struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		BaseBranch  string `json:"base_branch"`
	}
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		writeError(response, http.StatusBadRequest, errors.New("request must contain valid task JSON"))
		return
	}
	task, err := handler.tasks.Create(request.Context(), request.PathValue("id"), input.Title, input.Description, input.BaseBranch)
	if errors.Is(err, projectdomain.ErrProjectNotFound) {
		writeError(response, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(response, http.StatusUnprocessableEntity, err)
		return
	}
	writeJSON(response, http.StatusCreated, task)
}

func (handler *Handler) frontend() http.Handler {
	files := http.FileServer(http.FS(handler.assets))
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		if name == "." || name == "" {
			name = "index.html"
		}
		if _, err := fs.Stat(handler.assets, name); err != nil {
			request.URL.Path = "/index.html"
		}
		files.ServeHTTP(response, request)
	})
}

func protectLocalBrowser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		host := strings.ToLower(request.Host)
		hostname, port, err := net.SplitHostPort(host)
		if err != nil || port == "" || (hostname != "127.0.0.1" && hostname != "localhost") {
			http.Error(response, "invalid host", http.StatusForbidden)
			return
		}
		if origin := request.Header.Get("Origin"); origin != "" && !sameOrigin(origin, request.Host) {
			http.Error(response, "invalid origin", http.StatusForbidden)
			return
		}
		response.Header().Set("X-Content-Type-Options", "nosniff")
		response.Header().Set("X-Frame-Options", "DENY")
		response.Header().Set("Content-Security-Policy", "default-src 'self'; frame-ancestors 'none'; object-src 'none'; base-uri 'self'")
		next.ServeHTTP(response, request)
	})
}

func sameOrigin(origin, host string) bool {
	parsed, err := url.Parse(origin)
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	return strings.EqualFold(parsed.Host, host) && parsed.Scheme == "http"
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Cache-Control", "no-store")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeError(response http.ResponseWriter, status int, err error) {
	writeJSON(response, status, map[string]string{"error": fmt.Sprint(err)})
}
