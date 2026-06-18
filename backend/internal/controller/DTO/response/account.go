package response

type AccountResponse struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type LoginResponse struct {
	Account      AccountResponse `json:"account"`
	Token        string          `json:"token"`
	RefreshToken string          `json:"refresh_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
