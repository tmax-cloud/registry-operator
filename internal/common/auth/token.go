package auth

import "time"

type Token struct {
	// Type is "Basic" or "Bearer"
	Type  string
	Value string
}

type TokenResponse struct {
	Token        string    `json:"token"`
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	RefreshToken string    `json:"refresh_token"`
}
