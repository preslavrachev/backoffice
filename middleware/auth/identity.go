package auth

// Identity represents a user identifier that can be of any type
// Common types include string, int64, uint, or custom types
type Identity any

// IdentityHelpers provides utility functions for working with Identity values
type IdentityHelpers struct{}

// GetString returns the identity as a string, or empty string if not convertible
func (IdentityHelpers) GetString(id Identity) string {
	if s, ok := id.(string); ok {
		return s
	}
	return ""
}

// GetInt64 returns the identity as int64, or 0 if not convertible
// Also handles conversion from int, int32, uint, uint32
func (IdentityHelpers) GetInt64(id Identity) int64 {
	switch v := id.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case int32:
		return int64(v)
	case uint:
		return int64(v)
	case uint32:
		return int64(v)
	case uint64:
		// Note: potential overflow, but we return what we can
		return int64(v)
	}
	return 0
}

// GetUint returns the identity as uint, or 0 if not convertible
// Also handles conversion from int types (positive values only)
func (IdentityHelpers) GetUint(id Identity) uint {
	switch v := id.(type) {
	case uint:
		return v
	case uint32:
		return uint(v)
	case uint64:
		// Note: potential overflow, but we return what we can
		return uint(v)
	case int:
		if v >= 0 {
			return uint(v)
		}
	case int32:
		if v >= 0 {
			return uint(v)
		}
	case int64:
		if v >= 0 {
			return uint(v)
		}
	}
	return 0
}

// GetInt returns the identity as int, or 0 if not convertible
func (IdentityHelpers) GetInt(id Identity) int {
	switch v := id.(type) {
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		// Note: potential overflow on 32-bit systems
		return int(v)
	case uint:
		return int(v)
	case uint32:
		return int(v)
	}
	return 0
}

// Global helper instance for convenience
var ID = IdentityHelpers{}
