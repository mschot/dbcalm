package dto

// AuthorizeRequest represents the authorization request
type AuthorizeRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthorizeResponse represents the authorization response
type AuthorizeResponse struct {
	Code string `json:"code"`
}

// TokenRequest represents the token request
type TokenRequest struct {
	GrantType    string `json:"grant_type" binding:"required"` // "authorization_code" or "client_credentials"
	Code         string `json:"code"`                          // For authorization_code
	ClientID     string `json:"client_id"`                     // For client_credentials
	ClientSecret string `json:"client_secret"`                 // For client_credentials
}

// TokenResponse represents the token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // In seconds
}
