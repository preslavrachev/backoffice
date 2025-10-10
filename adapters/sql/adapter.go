package sql

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"backoffice/core"

	"github.com/iancoleman/strcase"
)

// Adapter implements the core.Adapter interface using pure sql.DB
type Adapter struct {
	db     *sql.DB
	logger *SQLLogger
}

// New creates a new SQL adapter
func New(db *sql.DB) *Adapter {
	return &Adapter{
		db:     db,
		logger: NewSQLLogger(false), // Default to disabled
	}
}

// NewWithDebug creates a new SQL adapter with debug logging enabled
func NewWithDebug(db *sql.DB, debugEnabled bool) *Adapter {
	return &Adapter{
		db:     db,
		logger: NewSQLLogger(debugEnabled),
	}
}

// SetDebugEnabled enables or disables SQL debug logging
func (a *Adapter) SetDebugEnabled(enabled bool) {
	a.logger.SetEnabled(enabled)
}

// loggedQueryContext wraps QueryContext with logging
func (a *Adapter) loggedQueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	start := time.Now()
	rows, err := a.db.QueryContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		a.logger.LogError(query, args, duration, err)
		return nil, err
	}

	// We'll log the row count after scanning in the calling function
	return rows, nil
}

// loggedExecContext wraps ExecContext with logging
func (a *Adapter) loggedExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	result, err := a.db.ExecContext(ctx, query, args...)
	duration := time.Since(start)

	if err != nil {
		a.logger.LogError(query, args, duration, err)
		return nil, err
	}

	a.logger.LogExec(query, args, duration, result)
	return result, nil
}

// getTableName extracts table name from resource or derives it from model type
func (a *Adapter) getTableName(resource *core.Resource) string {
	if resource.TableName != "" {
		return resource.TableName
	}

	// Convert struct name to snake_case table name
	modelType := resource.ModelType
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	return strcase.ToSnake(modelType.Name()) + "s"
}

// scanRowIntoStruct scans a sql.Rows into a struct using reflection
func (a *Adapter) scanRowIntoStruct(rows *sql.Rows, dest any) error {
	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	destValue := reflect.ValueOf(dest).Elem()
	destType := destValue.Type()

	// Create slice to hold scan values
	valuePtrs := make([]any, len(columns))

	// Map column names to struct fields
	fieldMap := make(map[string]reflect.Value)
	for i := 0; i < destType.NumField(); i++ {
		field := destType.Field(i)
		fieldValue := destValue.Field(i)

		// Get column name from db tag or convert field name to snake_case
		columnName := field.Tag.Get("db")
		if columnName == "" || columnName == "-" {
			columnName = strcase.ToSnake(field.Name)
		}

		fieldMap[columnName] = fieldValue
	}

	// Set up scan destinations
	for i, column := range columns {
		if fieldValue, exists := fieldMap[column]; exists && fieldValue.CanSet() {
			// Create a pointer to the field for scanning
			valuePtrs[i] = fieldValue.Addr().Interface()
		} else {
			// Unknown column, scan into a discard variable
			var discard any
			valuePtrs[i] = &discard
		}
	}

	return rows.Scan(valuePtrs...)
}

// Find retrieves records for a resource with comprehensive querying support
func (a *Adapter) Find(ctx context.Context, resource *core.Resource, query *core.Query) (*core.Result, error) {
	if query == nil {
		return nil, fmt.Errorf("query cannot be nil")
	}

	// Apply default sorting if none specified
	query.ApplyDefaultSort(resource)

	tableName := a.getTableName(resource)

	// Build SELECT clause
	selectClause := fmt.Sprintf("SELECT * FROM %s", tableName)

	// Build WHERE clause
	var whereConditions []string
	var args []any
	argIndex := 1

	for field, value := range query.Filters {
		// Resolve field name to database column name
		columnName := resource.GetColumnName(field)
		whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", columnName))
		args = append(args, value)
		argIndex++
	}

	// Build ORDER BY clause
	var orderClauses []string
	for _, sort := range query.Sort {
		direction := "ASC"
		if sort.Direction == core.SortDesc {
			direction = "DESC"
		}

		// Check if this is a field-level sort request (from UI click)
		// If so, check for field-level sort configuration first
		fieldSort := resource.GetFieldSortConfiguration(sort.Field)
		if fieldSort != nil {
			// Use field-level sort configuration for which columns to sort by,
			// but respect the user's requested direction
			for _, fs := range fieldSort {
				fsDirection := "ASC"
				if direction == "DESC" {
					fsDirection = "DESC"
				}
				columnName := resource.GetColumnName(fs.Field)
				orderClauses = append(orderClauses, fmt.Sprintf("%s %s", columnName, fsDirection))
			}
		} else {
			// Check if this field should be sortable (handle derived fields without sort config)
			if resource.IsFieldSortable(sort.Field) {
				// Use default behavior: resolve field name to database column name
				columnName := resource.GetColumnName(sort.Field)
				orderClauses = append(orderClauses, fmt.Sprintf("%s %s", columnName, direction))
			}
			// If field is not sortable (derived field without config), skip it
		}
	}

	// Construct full query
	queryStr := selectClause
	if len(whereConditions) > 0 {
		queryStr += " WHERE " + strings.Join(whereConditions, " AND ")
	}
	if len(orderClauses) > 0 {
		queryStr += " ORDER BY " + strings.Join(orderClauses, ", ")
	}

	// Count total records (before applying limit/offset)
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	if len(whereConditions) > 0 {
		countQuery += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	var totalCount int64
	start := time.Now()
	err := a.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount)
	duration := time.Since(start)
	if err != nil {
		a.logger.LogError(countQuery, args, duration, err)
		return nil, fmt.Errorf("failed to count records: %w", err)
	}
	a.logger.LogQuery(countQuery, args, duration, 1)

	// Apply pagination
	queryStr += fmt.Sprintf(" LIMIT %d OFFSET %d", query.Pagination.Limit, query.Pagination.Offset)

	// Execute query
	start = time.Now()
	rows, err := a.loggedQueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Create slice to hold results
	sliceType := reflect.SliceOf(resource.ModelType)
	results := reflect.New(sliceType).Elem()

	// Scan results
	for rows.Next() {
		// Create new instance of the model type
		item := reflect.New(resource.ModelType.Elem()).Interface()

		if err := a.scanRowIntoStruct(rows, item); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		results = reflect.Append(results, reflect.ValueOf(item))
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Log the query with row count
	duration = time.Since(start)
	a.logger.LogQuery(queryStr, args, duration, results.Len())

	// Convert results to []any
	var items []any
	for i := 0; i < results.Len(); i++ {
		items = append(items, results.Index(i).Interface())
	}

	// Calculate if there are more results
	currentResultCount := int64(len(items))
	hasMore := (int64(query.Pagination.Offset) + currentResultCount) < totalCount

	return &core.Result{
		Items:      items,
		TotalCount: totalCount,
		HasMore:    hasMore,
		Query:      *query,
	}, nil
}

// GetAll retrieves all records for a resource with optional filters (legacy method)
func (a *Adapter) GetAll(ctx context.Context, resource *core.Resource, filters map[string]any) ([]any, error) {
	tableName := a.getTableName(resource)

	queryStr := fmt.Sprintf("SELECT * FROM %s", tableName)

	var whereConditions []string
	var args []any

	for field, value := range filters {
		// Resolve field name to database column name
		columnName := resource.GetColumnName(field)
		whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", columnName))
		args = append(args, value)
	}

	if len(whereConditions) > 0 {
		queryStr += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	start := time.Now()
	rows, err := a.loggedQueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	var items []any
	for rows.Next() {
		item := reflect.New(resource.ModelType.Elem()).Interface()

		if err := a.scanRowIntoStruct(rows, item); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Log the query with row count
	duration := time.Since(start)
	a.logger.LogQuery(queryStr, args, duration, len(items))

	return items, nil
}

// GetByID retrieves a single record by its ID
func (a *Adapter) GetByID(ctx context.Context, resource *core.Resource, id any) (any, error) {
	tableName := a.getTableName(resource)
	primaryKey := resource.PrimaryKey
	if primaryKey == "" {
		primaryKey = "id"
	}

	// Resolve primary key field to column name
	primaryKeyColumn := resource.GetColumnName(primaryKey)
	queryStr := fmt.Sprintf("SELECT * FROM %s WHERE %s = ?", tableName, primaryKeyColumn)

	result := reflect.New(resource.ModelType.Elem()).Interface()

	start := time.Now()
	rows, err := a.loggedQueryContext(ctx, queryStr, id)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	rowCount := 0
	if !rows.Next() {
		duration := time.Since(start)
		a.logger.LogQuery(queryStr, []any{id}, duration, 0)
		return nil, fmt.Errorf("record with id %v not found", id)
	}
	rowCount = 1

	if err := a.scanRowIntoStruct(rows, result); err != nil {
		return nil, fmt.Errorf("failed to scan row: %w", err)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error reading row: %w", err)
	}

	// Log the query with row count
	duration := time.Since(start)
	a.logger.LogQuery(queryStr, []any{id}, duration, rowCount)

	return result, nil
}

// Create creates a new record
func (a *Adapter) Create(ctx context.Context, resource *core.Resource, data any) error {
	tableName := a.getTableName(resource)

	// Use reflection to build INSERT statement
	dataVal := reflect.ValueOf(data).Elem()
	dataType := reflect.TypeOf(data).Elem()

	var columns []string
	var placeholders []string
	var values []any

	for i := 0; i < dataVal.NumField(); i++ {
		field := dataVal.Field(i)
		fieldType := dataType.Field(i)

		// Skip ID field for auto-increment
		if fieldType.Name == "ID" || fieldType.Name == resource.IDField {
			continue
		}

		// Use resource's column name resolution
		columnName := resource.GetColumnName(fieldType.Name)
		columns = append(columns, columnName)
		placeholders = append(placeholders, "?")
		values = append(values, field.Interface())
	}

	queryStr := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s)",
		tableName,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := a.loggedExecContext(ctx, queryStr, values...)
	if err != nil {
		return fmt.Errorf("failed to create record: %w", err)
	}

	return nil
}

// Update updates an existing record with partial updates
func (a *Adapter) Update(ctx context.Context, resource *core.Resource, id any, data any) error {
	tableName := a.getTableName(resource)
	primaryKey := resource.PrimaryKey
	if primaryKey == "" {
		primaryKey = "id"
	}
	primaryKeyColumn := resource.GetColumnName(primaryKey)

	// Check if record exists first
	_, err := a.GetByID(ctx, resource, id)
	if err != nil {
		return err
	}

	// Build update statement from data
	dataVal := reflect.ValueOf(data).Elem()
	dataType := reflect.TypeOf(data).Elem()

	var setClauses []string
	var values []any

	for i := 0; i < dataVal.NumField(); i++ {
		field := dataVal.Field(i)
		fieldType := dataType.Field(i)

		// Skip ID/primary key fields
		if fieldType.Name == resource.IDField || fieldType.Name == "ID" {
			continue
		}

		// Only include non-zero values (fields that were actually set)
		if !field.IsZero() {
			// Use resource's column name resolution
			columnName := resource.GetColumnName(fieldType.Name)
			setClauses = append(setClauses, fmt.Sprintf("%s = ?", columnName))
			values = append(values, field.Interface())
		}
	}

	if len(setClauses) == 0 {
		// No fields to update
		return nil
	}

	// Add ID to values for WHERE clause
	values = append(values, id)

	queryStr := fmt.Sprintf(
		"UPDATE %s SET %s WHERE %s = ?",
		tableName,
		strings.Join(setClauses, ", "),
		primaryKeyColumn,
	)

	_, err = a.loggedExecContext(ctx, queryStr, values...)
	if err != nil {
		return fmt.Errorf("failed to update record: %w", err)
	}

	return nil
}

// Delete deletes a record by ID
func (a *Adapter) Delete(ctx context.Context, resource *core.Resource, id any) error {
	tableName := a.getTableName(resource)
	primaryKey := resource.PrimaryKey
	if primaryKey == "" {
		primaryKey = "id"
	}
	primaryKeyColumn := resource.GetColumnName(primaryKey)

	// Check if record exists first
	_, err := a.GetByID(ctx, resource, id)
	if err != nil {
		return err
	}

	queryStr := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, primaryKeyColumn)

	_, err = a.loggedExecContext(ctx, queryStr, id)
	if err != nil {
		return fmt.Errorf("failed to delete record: %w", err)
	}

	return nil
}

// GetSchema returns schema information for the resource
func (a *Adapter) GetSchema(resource *core.Resource) (*core.Schema, error) {
	schema := &core.Schema{
		Fields:     make([]core.FieldInfo, len(resource.Fields)),
		PrimaryKey: resource.PrimaryKey,
		TableName:  a.getTableName(resource),
		Metadata:   make(map[string]any),
	}

	copy(schema.Fields, resource.Fields)

	return schema, nil
}

// ValidateData validates data before operations
func (a *Adapter) ValidateData(resource *core.Resource, data any) error {
	if data == nil {
		return fmt.Errorf("data cannot be nil")
	}

	dataType := reflect.TypeOf(data)
	if dataType != resource.ModelType && dataType.Elem() != resource.ModelType.Elem() {
		return fmt.Errorf("data type mismatch: expected %v, got %v", resource.ModelType, dataType)
	}

	return nil
}

// Count returns the total number of records
func (a *Adapter) Count(ctx context.Context, resource *core.Resource, filters map[string]any) (int64, error) {
	tableName := a.getTableName(resource)

	queryStr := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)

	var whereConditions []string
	var args []any

	for field, value := range filters {
		// Resolve field name to database column name
		columnName := resource.GetColumnName(field)
		whereConditions = append(whereConditions, fmt.Sprintf("%s = ?", columnName))
		args = append(args, value)
	}

	if len(whereConditions) > 0 {
		queryStr += " WHERE " + strings.Join(whereConditions, " AND ")
	}

	var count int64
	start := time.Now()
	err := a.db.QueryRowContext(ctx, queryStr, args...).Scan(&count)
	duration := time.Since(start)
	if err != nil {
		a.logger.LogError(queryStr, args, duration, err)
		return 0, fmt.Errorf("failed to count records: %w", err)
	}
	a.logger.LogQuery(queryStr, args, duration, 1)

	return count, nil
}

// Search performs a basic text search across searchable fields
func (a *Adapter) Search(ctx context.Context, resource *core.Resource, searchQuery string) ([]any, error) {
	tableName := a.getTableName(resource)

	// Build search query for searchable fields
	var conditions []string
	var args []any

	for _, field := range resource.Fields {
		if field.Searchable && field.Type == "string" {
			// Use resource's column name resolution
			columnName := resource.GetColumnName(field.Name)
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", columnName))
			args = append(args, "%"+searchQuery+"%")
		}
	}

	if len(conditions) == 0 {
		// If no searchable fields, return empty results
		return []any{}, nil
	}

	// Join conditions with OR
	whereClause := strings.Join(conditions, " OR ")
	queryStr := fmt.Sprintf("SELECT * FROM %s WHERE %s", tableName, whereClause)

	start := time.Now()
	rows, err := a.loggedQueryContext(ctx, queryStr, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search records: %w", err)
	}
	defer rows.Close()

	var items []any
	for rows.Next() {
		item := reflect.New(resource.ModelType.Elem()).Interface()

		if err := a.scanRowIntoStruct(rows, item); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Log the query with row count
	duration := time.Since(start)
	a.logger.LogQuery(queryStr, args, duration, len(items))

	return items, nil
}
