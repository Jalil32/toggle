package slugs

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/gosimple/slug"
)

// Generate creates a URL-safe slug from input string
func Generate(input string) string {
	return slug.Make(input)
}

// WithFallback ensures uniqueness by appending UUID suffix
func WithFallback(input string) string {
	base := slug.Make(input)
	suffix := uuid.New().String()[:8]
	return fmt.Sprintf("%s-%s", base, suffix)
}
