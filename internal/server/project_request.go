package server

type projectRequest struct {
	Operation  string `json:"operation"`
	Name       string `json:"name"`
	Path       string `json:"path"`
	ParentPath string `json:"parent_path"`
}
