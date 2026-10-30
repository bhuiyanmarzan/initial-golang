package identity

// StatusResponse describes module availability, not authenticated identity.
type StatusResponse struct {
	Module string `json:"module"`
	Status string `json:"status"`
}
