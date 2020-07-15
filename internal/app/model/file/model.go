package file

type UpdateRequest struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}
