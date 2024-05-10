package sessions

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ValidateRespons struct {
	IsValid bool   `json:"isValid"`
	Message string `json:"message"`
}
