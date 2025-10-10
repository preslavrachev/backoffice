package sql

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// SQLLogger provides GORM-style SQL debug logging
type SQLLogger struct {
	enabled bool
	mu      sync.RWMutex
}

// NewSQLLogger creates a new SQL logger
func NewSQLLogger(enabled bool) *SQLLogger {
	return &SQLLogger{
		enabled: enabled,
	}
}

// IsEnabled returns whether SQL logging is enabled
func (l *SQLLogger) IsEnabled() bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.enabled
}

// SetEnabled enables or disables SQL logging
func (l *SQLLogger) SetEnabled(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.enabled = enabled
}

// LogQuery logs a SELECT query with execution time and row count
func (l *SQLLogger) LogQuery(query string, args []any, duration time.Duration, rowCount int) {
	if !l.IsEnabled() {
		return
	}

	formattedQuery := l.formatQuery(query)
	log.Printf("[SQL] [%.2fms] [rows:%d] %s %s",
		float64(duration.Nanoseconds())/1e6,
		rowCount,
		formattedQuery,
		l.formatArgs(args))
}

// LogExec logs an INSERT/UPDATE/DELETE query with execution time and affected rows
func (l *SQLLogger) LogExec(query string, args []any, duration time.Duration, result sql.Result) {
	if !l.IsEnabled() {
		return
	}

	formattedQuery := l.formatQuery(query)
	rowsAffected := int64(-1)

	if result != nil {
		if affected, err := result.RowsAffected(); err == nil {
			rowsAffected = affected
		}
	}

	if rowsAffected >= 0 {
		log.Printf("[SQL] [%.2fms] [rows:%d] %s %s",
			float64(duration.Nanoseconds())/1e6,
			rowsAffected,
			formattedQuery,
			l.formatArgs(args))
	} else {
		log.Printf("[SQL] [%.2fms] %s %s",
			float64(duration.Nanoseconds())/1e6,
			formattedQuery,
			l.formatArgs(args))
	}
}

// LogError logs a query that resulted in an error
func (l *SQLLogger) LogError(query string, args []any, duration time.Duration, err error) {
	if !l.IsEnabled() {
		return
	}

	formattedQuery := l.formatQuery(query)
	log.Printf("[SQL] [%.2fms] [ERROR] %s %s - %v",
		float64(duration.Nanoseconds())/1e6,
		formattedQuery,
		l.formatArgs(args),
		err)
}

// formatQuery cleans up the SQL query for better readability
func (l *SQLLogger) formatQuery(query string) string {
	// Remove extra whitespace and normalize
	query = strings.TrimSpace(query)
	query = strings.ReplaceAll(query, "\n", " ")
	query = strings.ReplaceAll(query, "\t", " ")

	// Collapse multiple spaces into single spaces
	for strings.Contains(query, "  ") {
		query = strings.ReplaceAll(query, "  ", " ")
	}

	return query
}

// formatArgs formats the query arguments for logging
func (l *SQLLogger) formatArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}

	var formatted []string
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			formatted = append(formatted, fmt.Sprintf(`"%s"`, v))
		case nil:
			formatted = append(formatted, "NULL")
		default:
			formatted = append(formatted, fmt.Sprintf("%v", v))
		}
	}

	return fmt.Sprintf("[Args: [%s]]", strings.Join(formatted, ", "))
}
