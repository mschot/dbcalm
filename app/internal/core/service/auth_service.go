package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/martijn/dbcalm/internal/core/domain"
	"github.com/martijn/dbcalm/internal/core/repository"
	"golang.org/x/crypto/bcrypt"
)

const (
	AuthCodeExpirationMinutes = 10
	TokenExpirationHours      = 1
	BcryptCost                = 10
)

type AuthService struct {
	userRepo     repository.UserRepository
	clientRepo   repository.ClientRepository
	authCodeRepo repository.AuthCodeRepository
	jwtSecret    string
	jwtAlgorithm string
}

func NewAuthService(
	userRepo repository.UserRepository,
	clientRepo repository.ClientRepository,
	authCodeRepo repository.AuthCodeRepository,
	jwtSecret string,
	jwtAlgorithm string,
) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		clientRepo:   clientRepo,
		authCodeRepo: authCodeRepo,
		jwtSecret:    jwtSecret,
		jwtAlgorithm: jwtAlgorithm,
	}
}

// HashPassword hashes a password using bcrypt
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against a hash
func (s *AuthService) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// AuthorizeUser authenticates a user and returns an auth code
func (s *AuthService) AuthorizeUser(ctx context.Context, username, password string) (*domain.AuthCode, error) {
	// Find user
	user, err := s.userRepo.FindByUsername(ctx, username)
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Verify password
	if !s.VerifyPassword(password, user.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Create auth code
	authCode := domain.NewAuthCode(username, []string{"all"}, AuthCodeExpirationMinutes)

	if err := s.authCodeRepo.Create(ctx, authCode); err != nil {
		return nil, fmt.Errorf("failed to create auth code: %w", err)
	}

	// Clean up expired codes
	_ = s.authCodeRepo.DeleteExpired(ctx)

	return authCode, nil
}

// ExchangeAuthCode exchanges an auth code for a JWT token
func (s *AuthService) ExchangeAuthCode(ctx context.Context, code string) (string, error) {
	// Find auth code
	authCode, err := s.authCodeRepo.FindByCode(ctx, code)
	if err != nil {
		return "", fmt.Errorf("invalid auth code")
	}

	// Check if expired
	if authCode.IsExpired() {
		_ = s.authCodeRepo.Delete(ctx, code)
		return "", fmt.Errorf("auth code expired")
	}

	// Generate JWT
	token, err := s.generateJWT(authCode.Username, "user", authCode.Scopes)
	if err != nil {
		return "", err
	}

	// Delete auth code (single use)
	_ = s.authCodeRepo.Delete(ctx, code)

	return token, nil
}

// AuthenticateClient authenticates a client and returns a JWT token
func (s *AuthService) AuthenticateClient(ctx context.Context, clientID, clientSecret string) (string, error) {
	// Find client
	client, err := s.clientRepo.FindByID(ctx, clientID)
	if err != nil {
		return "", fmt.Errorf("invalid client credentials")
	}

	// Verify secret
	if !s.VerifyPassword(clientSecret, client.Secret) {
		return "", fmt.Errorf("invalid client credentials")
	}

	// Generate JWT
	token, err := s.generateJWT(clientID, "client", client.Scopes)
	if err != nil {
		return "", err
	}

	return token, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *AuthService) ValidateToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if token.Method.Alg() != s.jwtAlgorithm {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// generateJWT generates a JWT token
func (s *AuthService) generateJWT(subject, subjectType string, scopes []string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(TokenExpirationHours * time.Hour)

	claims := TokenClaims{
		Subject:     subject,
		SubjectType: subjectType,
		Scopes:      scopes,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "dbcalm",
		},
	}

	var signingMethod jwt.SigningMethod
	switch s.jwtAlgorithm {
	case "HS256":
		signingMethod = jwt.SigningMethodHS256
	case "HS384":
		signingMethod = jwt.SigningMethodHS384
	case "HS512":
		signingMethod = jwt.SigningMethodHS512
	default:
		signingMethod = jwt.SigningMethodHS256
	}

	token := jwt.NewWithClaims(signingMethod, claims)
	tokenString, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// TokenClaims represents JWT claims
type TokenClaims struct {
	Subject     string   `json:"sub"`
	SubjectType string   `json:"sub_type"` // "user" or "client"
	Scopes      []string `json:"scopes"`
	jwt.RegisteredClaims
}
