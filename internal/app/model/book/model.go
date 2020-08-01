package book

type UpdateRequest struct {
	Title string   `json:"title"`
	Tags  []string `json:"tags"`
}
