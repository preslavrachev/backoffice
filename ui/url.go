package ui

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// AdminURLBuilder provides a fluent interface for building admin panel URLs
type AdminURLBuilder struct {
	basePath string
	params   url.Values
}

// NewAdminURL creates a new URL builder for the given resource
func NewAdminURL(resourceName string) *AdminURLBuilder {
	return &AdminURLBuilder{
		basePath: "/admin/" + url.PathEscape(resourceName),
		params:   make(url.Values),
	}
}

// PreserveFromRequest copies all user-facing parameters from the current request
// Skips internal parameters like "load_more" that shouldn't be preserved
func (b *AdminURLBuilder) PreserveFromRequest(r *http.Request) *AdminURLBuilder {
	for k, v := range r.URL.Query() {
		if !isInternalParam(k) {
			b.params[k] = v
		}
	}
	return b
}

// WithSort sets sorting parameters
func (b *AdminURLBuilder) WithSort(field, direction string) *AdminURLBuilder {
	if field != "" {
		b.params.Set("sort", field)
		if direction != "" {
			b.params.Set("direction", direction)
		}
	}
	return b
}

// WithPagination sets pagination parameters
func (b *AdminURLBuilder) WithPagination(offset, limit int) *AdminURLBuilder {
	b.params.Set("offset", strconv.Itoa(offset))
	b.params.Set("limit", strconv.Itoa(limit))
	return b
}

// WithFilter adds a filter parameter
func (b *AdminURLBuilder) WithFilter(key, value string) *AdminURLBuilder {
	if key != "" && value != "" {
		b.params.Set(key, value)
	}
	return b
}

// WithLoadMore adds the load_more internal parameter
func (b *AdminURLBuilder) WithLoadMore() *AdminURLBuilder {
	b.params.Set("load_more", "true")
	return b
}

// WithParam sets an arbitrary parameter
func (b *AdminURLBuilder) WithParam(key, value string) *AdminURLBuilder {
	if key != "" {
		b.params.Set(key, value)
	}
	return b
}

// RemoveParam removes a parameter
func (b *AdminURLBuilder) RemoveParam(key string) *AdminURLBuilder {
	b.params.Del(key)
	return b
}

// String builds and returns the final URL
func (b *AdminURLBuilder) String() string {
	if len(b.params) == 0 {
		return b.basePath
	}
	return b.basePath + "?" + b.params.Encode()
}

// isInternalParam checks if a parameter is internal and should not be preserved
// when building new URLs based on current request
func isInternalParam(key string) bool {
	internalParams := []string{
		"load_more",   // HTMX load more trigger
		"success",     // Success messages
		"resource",    // Resource name for messages
		"highlighted", // Item highlighting
	}

	for _, param := range internalParams {
		if strings.EqualFold(key, param) {
			return true
		}
	}
	return false
}
