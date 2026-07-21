package auth

// Request/response shapes for the auth endpoints. Keep transport DTOs here,
// separate from handler/service logic, so the API surface is easy to find.

// credentials is the signup/login request body.
type credentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// userView is the public representation of a user (never exposes the hash).
type userView struct {
	ID    int64  `json:"id"`
	Email string `json:"email"`
}
