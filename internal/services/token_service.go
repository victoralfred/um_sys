package services

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/victoralfred/um_sys/internal/domain/auth"
	"github.com/victoralfred/um_sys/internal/domain/user"
)

// TokenService implements JWT token operations
type TokenService struct {
	secretKey          []byte
	issuer             string
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
	userRepo           user.Repository
	tokenStore         auth.TokenStore
}

// NewTokenService creates a new token service
func NewTokenService(
	secretKey string,
	issuer string,
	accessTokenExpiry time.Duration,
	refreshTokenExpiry time.Duration,
	userRepo user.Repository,
) *TokenService {
	return &TokenService{
		secretKey:          []byte(secretKey),
		issuer:             issuer,
		accessTokenExpiry:  accessTokenExpiry,
		refreshTokenExpiry: refreshTokenExpiry,
		userRepo:           userRepo,
	}
}

// SetTokenStore sets the token store for blacklisting
func (s *TokenService) SetTokenStore(store auth.TokenStore) {
	s.tokenStore = store
}

// GenerateTokenPair generates access and refresh tokens for a user
func (s *TokenService) GenerateTokenPair(ctx context.Context, u *user.User) (*auth.TokenPair, error) {
	now := time.Now()

	// Generate JTI (JWT ID) for token revocation
	accessTokenID := uuid.New().String()
	refreshTokenID := uuid.New().String()

	// Create access token claims
	accessClaims := &auth.Claims{
		UserID:    u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Roles:     []string{"user"}, // TODO: Load actual roles from user
		TokenType: auth.AccessToken,
		ExpiresAt: now.Add(s.accessTokenExpiry),
		IssuedAt:  now,
		NotBefore: now,
		Subject:   u.ID.String(),
		Issuer:    s.issuer,
		Audience:  []string{s.issuer},
		JTI:       accessTokenID,
	}

	// Create refresh token claims
	refreshClaims := &auth.Claims{
		UserID:    u.ID,
		Email:     u.Email,
		Username:  u.Username,
		TokenType: auth.RefreshToken,
		ExpiresAt: now.Add(s.refreshTokenExpiry),
		IssuedAt:  now,
		NotBefore: now,
		Subject:   u.ID.String(),
		Issuer:    s.issuer,
		Audience:  []string{s.issuer},
		JTI:       refreshTokenID,
	}

	// Generate access token
	accessToken, err := s.generateToken(accessClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token
	refreshToken, err := s.generateToken(refreshClaims)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &auth.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(s.accessTokenExpiry.Seconds()),
		ExpiresAt:    accessClaims.ExpiresAt,
	}, nil
}

// ValidateToken validates a token and returns the claims
func (s *TokenService) ValidateToken(ctx context.Context, tokenString string, tokenType auth.TokenType) (*auth.Claims, error) {
	// Parse the token
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Extract claims
	mapClaims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return nil, auth.ErrInvalidToken
	}

	// Convert to our Claims struct
	claims, err := s.mapToClaims(mapClaims)
	if err != nil {
		return nil, err
	}

	// Verify token type
	if claims.TokenType != tokenType {
		return nil, fmt.Errorf("invalid token type: expected %s, got %s", tokenType, claims.TokenType)
	}

	// Check if token is revoked (if token store is available)
	if s.tokenStore != nil {
		revoked, err := s.IsTokenRevoked(ctx, claims.JTI)
		if err != nil {
			return nil, fmt.Errorf("failed to check token revocation: %w", err)
		}
		if revoked {
			return nil, auth.ErrTokenRevoked
		}
	}

	// Check expiration
	if time.Now().After(claims.ExpiresAt) {
		return nil, auth.ErrTokenExpired
	}

	return claims, nil
}

// RefreshTokens generates new token pair from refresh token
func (s *TokenService) RefreshTokens(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	// Validate refresh token
	claims, err := s.ValidateToken(ctx, refreshToken, auth.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	// Get fresh user data (in case roles or status changed)
	if s.userRepo != nil {
		freshUser, err := s.userRepo.GetByID(ctx, claims.UserID)
		if err != nil {
			return nil, fmt.Errorf("failed to get user: %w", err)
		}

		// Check if user is still active
		if freshUser.Status != user.StatusActive {
			return nil, auth.ErrAccountInactive
		}

		// Generate new token pair
		return s.GenerateTokenPair(ctx, freshUser)
	}

	// If no user repo, generate with existing claims (less secure)
	u := &user.User{
		ID:       claims.UserID,
		Email:    claims.Email,
		Username: claims.Username,
		Status:   user.StatusActive,
	}

	return s.GenerateTokenPair(ctx, u)
}

// RevokeToken revokes a token (adds to blacklist)
func (s *TokenService) RevokeToken(ctx context.Context, tokenID string) error {
	if s.tokenStore == nil {
		// If no token store, can't revoke
		return nil
	}

	// Store with max expiry time (we'll check both access and refresh token expiry)
	maxExpiry := s.accessTokenExpiry
	if s.refreshTokenExpiry > maxExpiry {
		maxExpiry = s.refreshTokenExpiry
	}

	return s.tokenStore.Store(ctx, tokenID, uuid.Nil, time.Now().Add(maxExpiry))
}

// IsTokenRevoked checks if a token is revoked
func (s *TokenService) IsTokenRevoked(ctx context.Context, tokenID string) (bool, error) {
	if s.tokenStore == nil {
		return false, nil
	}

	return s.tokenStore.Exists(ctx, tokenID)
}

// generateToken generates a JWT token from claims
func (s *TokenService) generateToken(claims *auth.Claims) (string, error) {
	// Convert to JWT claims
	jwtClaims := jwt.MapClaims{
		"user_id":    claims.UserID.String(),
		"email":      claims.Email,
		"username":   claims.Username,
		"roles":      claims.Roles,
		"token_type": string(claims.TokenType),
		"exp":        claims.ExpiresAt.Unix(),
		"iat":        claims.IssuedAt.Unix(),
		"nbf":        claims.NotBefore.Unix(),
		"sub":        claims.Subject,
		"iss":        claims.Issuer,
		"aud":        claims.Audience,
		"jti":        claims.JTI,
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)

	// Sign token
	tokenString, err := token.SignedString(s.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// mapToClaims converts JWT MapClaims to our Claims struct
func (s *TokenService) mapToClaims(m jwt.MapClaims) (*auth.Claims, error) {
	// Parse user ID
	userIDStr, ok := m["user_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid user_id in token")
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse user_id: %w", err)
	}

	// Parse expiration
	exp, ok := m["exp"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid exp in token")
	}

	// Parse issued at
	iat, ok := m["iat"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid iat in token")
	}

	// Parse not before
	nbf, ok := m["nbf"].(float64)
	if !ok {
		return nil, fmt.Errorf("invalid nbf in token")
	}

	// Parse roles
	var roles []string
	if rolesInterface, ok := m["roles"].([]interface{}); ok {
		for _, r := range rolesInterface {
			if role, ok := r.(string); ok {
				roles = append(roles, role)
			}
		}
	}

	// Parse audience
	var audience []string
	if audInterface, ok := m["aud"].([]interface{}); ok {
		for _, a := range audInterface {
			if aud, ok := a.(string); ok {
				audience = append(audience, aud)
			}
		}
	}

	// Parse token type
	tokenTypeStr, _ := m["token_type"].(string)

	return &auth.Claims{
		UserID:    userID,
		Email:     m["email"].(string),
		Username:  m["username"].(string),
		Roles:     roles,
		TokenType: auth.TokenType(tokenTypeStr),
		ExpiresAt: time.Unix(int64(exp), 0),
		IssuedAt:  time.Unix(int64(iat), 0),
		NotBefore: time.Unix(int64(nbf), 0),
		Subject:   m["sub"].(string),
		Issuer:    m["iss"].(string),
		Audience:  audience,
		JTI:       m["jti"].(string),
	}, nil
}

// RevokeRefreshToken revokes a refresh token by parsing it and revoking its JTI
func (s *TokenService) RevokeRefreshToken(ctx context.Context, refreshToken string) error {
	// Parse the refresh token to get the JTI
	token, err := jwt.Parse(refreshToken, func(token *jwt.Token) (interface{}, error) {
		// Check signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.secretKey, nil
	})

	// If we can't parse the token, consider it already invalid
	if err != nil {
		return nil
	}

	// Extract claims and revoke the token
	if mapClaims, ok := token.Claims.(jwt.MapClaims); ok {
		if jti, ok := mapClaims["jti"].(string); ok && jti != "" {
			return s.RevokeToken(ctx, jti)
		}
	}

	return nil
}
