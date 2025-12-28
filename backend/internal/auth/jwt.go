package auth

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenExpired      = errors.New("token expired")
	ErrInvalidSignature  = errors.New("invalid signature")
	ErrMissingAuthHeader = errors.New("missing authorization header")
	ErrInvalidAuthHeader = errors.New("invalid authorization header format")
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

const userContextKey contextKey = "user"

// JWKS represents the JSON Web Key Set structure
type JWKS struct {
	Keys []JWK `json:"keys"`
}

// JWK represents a JSON Web Key
type JWK struct {
	Kty string `json:"kty"` // Key Type (e.g., "OKP" for Ed25519)
	Crv string `json:"crv"` // Curve (e.g., "Ed25519")
	X   string `json:"x"`   // Public key coordinate
	Kid string `json:"kid"` // Key ID
}

// BetterAuthClaims represents the JWT claims from Better Auth
type BetterAuthClaims struct {
	jwt.RegisteredClaims
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// JWTVerifier handles JWT token verification using JWKS
type JWTVerifier struct {
	jwksURL    string
	issuer     string
	audience   string
	jwks       *JWKS
	jwksMutex  sync.RWMutex
	httpClient *http.Client
}

// NewJWTVerifier creates a new JWT verifier
// jwksURL: The URL to fetch JWKS (e.g., "http://localhost:3000/api/auth/jwks")
// issuer: The expected issuer (e.g., "http://localhost:3000")
// audience: The expected audience (e.g., "http://localhost:3000")
func NewJWTVerifier(jwksURL, issuer, audience string) *JWTVerifier {
	return &JWTVerifier{
		jwksURL:  jwksURL,
		issuer:   issuer,
		audience: audience,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// fetchJWKS fetches the JWKS from the Better Auth server
func (v *JWTVerifier) fetchJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create JWKS request: %w", err)
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("JWKS endpoint returned status %d: %s", resp.StatusCode, string(body))
	}

	var jwks JWKS
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("failed to decode JWKS: %w", err)
	}

	v.jwksMutex.Lock()
	v.jwks = &jwks
	v.jwksMutex.Unlock()

	return nil
}

// getPublicKey retrieves the public key for the given key ID
func (v *JWTVerifier) getPublicKey(kid string) (ed25519.PublicKey, error) {
	v.jwksMutex.RLock()
	jwks := v.jwks
	v.jwksMutex.RUnlock()

	if jwks == nil {
		return nil, errors.New("JWKS not loaded")
	}

	for _, key := range jwks.Keys {
		if key.Kid == kid {
			if key.Kty != "OKP" || key.Crv != "Ed25519" {
				return nil, fmt.Errorf("unsupported key type: %s/%s", key.Kty, key.Crv)
			}

			// Decode base64url-encoded public key
			pubKeyBytes, err := base64.RawURLEncoding.DecodeString(key.X)
			if err != nil {
				return nil, fmt.Errorf("failed to decode public key: %w", err)
			}

			if len(pubKeyBytes) != ed25519.PublicKeySize {
				return nil, fmt.Errorf("invalid public key size: expected %d, got %d",
					ed25519.PublicKeySize, len(pubKeyBytes))
			}

			return ed25519.PublicKey(pubKeyBytes), nil
		}
	}

	return nil, fmt.Errorf("key with kid %s not found", kid)
}

// VerifyToken verifies a JWT token and returns the claims
func (v *JWTVerifier) VerifyToken(ctx context.Context, tokenString string) (*BetterAuthClaims, error) {
	// Ensure JWKS is loaded
	v.jwksMutex.RLock()
	jwksLoaded := v.jwks != nil
	v.jwksMutex.RUnlock()

	if !jwksLoaded {
		if err := v.fetchJWKS(ctx); err != nil {
			return nil, fmt.Errorf("failed to load JWKS: %w", err)
		}
	}

	// Parse token
	token, err := jwt.ParseWithClaims(tokenString, &BetterAuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if token.Method.Alg() != jwt.SigningMethodEdDSA.Alg() {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		// Get key ID from token header
		kid, ok := token.Header["kid"].(string)
		if !ok {
			return nil, errors.New("missing kid in token header")
		}

		// Fetch public key
		publicKey, err := v.getPublicKey(kid)
		if err != nil {
			// Try refreshing JWKS if key not found
			if strings.Contains(err.Error(), "not found") {
				if refreshErr := v.fetchJWKS(ctx); refreshErr != nil {
					return nil, fmt.Errorf("failed to refresh JWKS: %w", refreshErr)
				}
				publicKey, err = v.getPublicKey(kid)
			}
			if err != nil {
				return nil, err
			}
		}

		return publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		if errors.Is(err, jwt.ErrSignatureInvalid) {
			return nil, ErrInvalidSignature
		}
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*BetterAuthClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	// Verify issuer
	if v.issuer != "" && claims.Issuer != v.issuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", v.issuer, claims.Issuer)
	}

	// Verify audience
	if v.audience != "" {
		validAudience := false
		for _, aud := range claims.Audience {
			if aud == v.audience {
				validAudience = true
				break
			}
		}
		if !validAudience {
			return nil, fmt.Errorf("invalid audience: expected %s", v.audience)
		}
	}

	return claims, nil
}

// ExtractTokenFromHeader extracts the Bearer token from the Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", ErrMissingAuthHeader
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return "", ErrInvalidAuthHeader
	}

	return parts[1], nil
}

// Middleware returns an HTTP middleware that verifies JWT tokens
func (v *JWTVerifier) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, err := ExtractTokenFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		claims, err := v.VerifyToken(r.Context(), token)
		if err != nil {
			status := http.StatusUnauthorized
			if errors.Is(err, ErrTokenExpired) {
				status = http.StatusUnauthorized
			}
			http.Error(w, "Unauthorized: "+err.Error(), status)
			return
		}

		// Add claims to request context
		ctx := context.WithValue(r.Context(), userContextKey, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetUserFromContext retrieves user claims from the request context
func GetUserFromContext(ctx context.Context) (*BetterAuthClaims, bool) {
	claims, ok := ctx.Value(userContextKey).(*BetterAuthClaims)
	return claims, ok
}
