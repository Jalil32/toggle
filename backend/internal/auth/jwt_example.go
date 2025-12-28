package auth

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

// Example usage of JWT verification in your Go backend

// InitJWTVerifier initializes the JWT verifier with environment variables
func InitJWTVerifier() *JWTVerifier {
	jwksURL := os.Getenv("BETTER_AUTH_JWKS_URL")
	if jwksURL == "" {
		jwksURL = "http://localhost:3000/api/auth/jwks"
	}

	issuer := os.Getenv("BETTER_AUTH_ISSUER")
	if issuer == "" {
		issuer = "http://localhost:3000"
	}

	audience := os.Getenv("BETTER_AUTH_AUDIENCE")
	if audience == "" {
		audience = "http://localhost:3000"
	}

	return NewJWTVerifier(jwksURL, issuer, audience)
}

// Example: Protected HTTP handler that requires authentication
func ExampleProtectedHandler(verifier *JWTVerifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get user from context (set by middleware)
		user, ok := GetUserFromContext(r.Context())
		if !ok {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Use user information
		response := map[string]interface{}{
			"message": "This is a protected endpoint",
			"user": map[string]string{
				"id":    user.UserID,
				"email": user.Email,
				"name":  user.Name,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode response: %v", err)
		}
	}
}

// Example: Setting up routes with JWT middleware
func ExampleRouterSetup() {
	verifier := InitJWTVerifier()

	mux := http.NewServeMux()

	// Public endpoint - no auth required
	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("OK")); err != nil {
			log.Printf("Failed to write response: %v", err)
		}
	})

	// Protected endpoint - auth required
	protectedHandler := ExampleProtectedHandler(verifier)
	mux.Handle("/api/v1/me", verifier.Middleware(protectedHandler))

	// Protected endpoints for tenants
	mux.Handle("/api/v1/me/tenants", verifier.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, _ := GetUserFromContext(r.Context())
		log.Printf("User %s (%s) accessing tenants", user.Name, user.Email)
		// Your tenant logic here
	})))

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// Example: Manual token verification (without middleware)
func ExampleManualVerification() {
	verifier := InitJWTVerifier()

	http.HandleFunc("/api/v1/custom", func(w http.ResponseWriter, r *http.Request) {
		// Extract token manually
		token, err := ExtractTokenFromHeader(r.Header.Get("Authorization"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Verify token manually
		claims, err := verifier.VerifyToken(r.Context(), token)
		if err != nil {
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		// Use claims
		log.Printf("Authenticated user: %s (%s)", claims.Name, claims.Email)
		w.WriteHeader(http.StatusOK)
	})
}
