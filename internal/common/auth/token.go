package auth

import "time"

// TokenType is HTTP Authorization Header's token type
type TokenType string

const (
	// TokenTypeBasic is Basic type token
	TokenTypeBasic TokenType = "Basic"
	// TokenTypeBearer is Bearer type token
	TokenTypeBearer TokenType = "Bearer"
)

type Token struct {
	// Type is "Basic" or "Bearer"
	Type TokenType
	// Value...
	Value string
}

type TokenResponse struct {
	Token        string    `json:"token"`
	AccessToken  string    `json:"access_token"`
	ExpiresIn    int       `json:"expires_in"`
	IssuedAt     time.Time `json:"issued_at"`
	RefreshToken string    `json:"refresh_token"`
}
