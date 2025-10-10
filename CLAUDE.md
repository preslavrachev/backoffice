BackOffice is a data-agnostic Go library providing a Django Admin-style CRUD admin panel for Go applications.

## Core Features
- **Minimal Dependencies**: Core library uses minimal external dependencies (templ for UI, strcase for field naming)
- **Pure sql.DB Support**: Built-in SQL adapter using standard `database/sql`
- **Selective Resource Registration**: Only explicitly registered structs appear in admin panel
- **Smart Resource Naming**: Three-tier priority system (fluent builder > config > auto-generated)
- **Intelligent Pluralization**: Handles irregular English plurals (Person->People, Mouse->Mice) via strcase library
- **Custom Display Names**: Via fluent builder API `RegisterResource(&User{}).WithName("Customer").WithPluralName("Customers")`
- **Full CRUD Operations**: Create, read, update, delete with form validation and pagination
- **Dynamic Forms**: Auto-generated based on struct field metadata with optional configuration
- **Go Workspace Support**: Dependency isolation allowing examples to use convenience libraries
- **Middleware Support**: Authentication, authorization, logging, and custom middleware
- **Many-to-One Relationships**: Auto-detected with multiple display patterns (compact, badge, hierarchical)
- **Derived Fields**: Dynamic fields calculated at runtime with configurable sorting
- **Professional UI Patterns**: Modern admin interface with HTMX interactions

## Tech Stack
**Backend**: Go 1.24+, net/http or any HTTP router (Gin, Echo, Chi), templ  
**Frontend**: Tailwind CSS, HTMX, minimal Alpine.js (considering vanilla JS migration)  
**Build**: Go modules with adapter architecture, pre-compiled templ components

## Development Philosophy
**HTML-first approach**: Use semantic HTML as the foundation
- Standard HTML forms, links, and elements for core functionality
- HTMX for asynchronous enhancements (delete operations, form submissions)
- Alpine.js only when absolutely necessary (state management, complex interactions)
- Prefer vanilla JavaScript over Alpine.js when possible
- **Simplest solution first** - avoid over-engineering with unnecessary JavaScript

## Database Field Guidelines
**Smart Column Mapping**: BackOffice automatically maps Go struct fields to database columns using a priority system:
- **Priority Order**: Explicit override > `db:` tag > `gorm:` tag > `json:` tag > snake_case fallback
- **GORM Compatibility**: Supports `gorm:"column:name"` tags for seamless migration from GORM projects
- **Legacy Database Support**: Use `WithDBColumnName("custom_column")` for non-standard column names
- **Relationship Exclusion**: `db:"-"` excludes fields from database operations (for relationships)
- **Primary Key Detection**: Uses reflection to find fields with `db:"id"` or ID field patterns

**Nullable Time Fields**: Always use `sql.NullTime` instead of `*time.Time` for nullable timestamp fields
- **Why `sql.NullTime`**: Explicit database semantics, avoids relationship detection issues
- **Avoid `*time.Time`**: Can trigger relationship detection heuristics, less explicit intent
- **Pattern**: Fields ending in "At" (CreatedAt, UpdatedAt, CancelledAt) should use appropriate types
- **Example**: `CancelledAt sql.NullTime \`db:"cancelled_at\`` instead of `CancelledAt *time.Time`

## Project Structure
```
backoffice/
├── go.work                     # Go workspace configuration
├── go.mod                      # Core library dependencies
├── core/                       # Core library
│   ├── admin.go               # Main BackOffice struct and configuration
│   ├── resource.go            # Resource interface and metadata
│   ├── field.go               # Field information and validation
│   └── adapter.go             # Base adapter interface
├── adapters/                   # Data source adapters
│   └── sql/                   # Pure sql.DB adapter
│       ├── adapter.go         # Pure database/sql implementation
│       └── adapter_test.go    # Comprehensive test suite
├── ui/                         # Templ UI components
│   ├── handlers.go            # HTTP handlers for CRUD operations
│   ├── layout.templ           # Base layout component
│   ├── list.templ             # Resource list view
│   ├── form.templ             # Create/edit form
│   ├── detail.templ           # Resource detail view
│   ├── sidepane.templ         # Side panel forms
│   └── *_templ.go            # Generated Go code
├── middleware/auth/           # Authentication middleware
├── config/                    # Configuration utilities
└── examples/sql-example/      # Working demo with isolated dependencies
    ├── go.mod                 # Can use convenience libraries (sqlx)
    └── main.go                # Demo with pure sql.DB adapter
```

## Data Adapter Interface
```go
// Adapter defines the interface for data source adapters
type Adapter interface {
    // Query operations with pagination and sorting
    Find(ctx context.Context, resource *Resource, query *Query) (*Result, error)
    GetByID(ctx context.Context, resource *Resource, id any) (any, error)
    
    // Mutation operations
    Create(ctx context.Context, resource *Resource, data any) error
    Update(ctx context.Context, resource *Resource, id any, data any) error
    Delete(ctx context.Context, resource *Resource, id any) error
    
    // Metadata operations
    GetSchema(resource *Resource) (*Schema, error)
    ValidateData(resource *Resource, data any) error
    
    // Advanced operations
    Count(ctx context.Context, resource *Resource, filters map[string]any) (int64, error)
    Search(ctx context.Context, resource *Resource, query string) ([]any, error)
}

// Query represents pagination, sorting, and filtering parameters
type Query struct {
    Offset     int
    Limit      int
    Sort       string
    SortDir    SortDirection
    Filters    map[string]any
}

// Result represents paginated query results
type Result struct {
    Data       []any
    Total      int64
    HasMore    bool
}
```

## Resource Registration
Simple struct registration with fluent configuration:
```go
type User struct {
    ID        uint      `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    Active    bool      `json:"active" db:"active"`
}

// Register with BackOffice using pure SQL adapter
sqlAdapter := sqladapter.New(db) // db is *sql.DB
admin := core.New(sqlAdapter, auth.WithNoAuth())
admin.RegisterResource(&User{}).
    WithName("Customer").
    WithPluralName("Customers").
    WithDefaultSort("CreatedAt", core.SortDesc).
    WithField("Name", func(f *core.FieldBuilder) {
        f.DisplayName("Full Name").Required(true).Searchable(true)
    }).
    WithField("Email", func(f *core.FieldBuilder) {
        f.DisplayName("Email Address").Required(true).Unique(true)
    }).
    WithDerivedField("AccountAge", "Account Age", func(user any) string {
        u := user.(*User)
        days := int(time.Since(u.CreatedAt).Hours() / 24)
        return fmt.Sprintf("%d days", days)
    }, func(f *core.FieldBuilder) {
        f.SortBy("CreatedAt", core.SortDesc)
    })
```

## Resource Naming (Priority Order)
1. **Fluent builder**: `RegisterResource(&User{}).WithName("Customer").WithPluralName("Customers")`
2. **Config properties**: `backoffice.SetResourceConfig("User", backoffice.ResourceConfig{DisplayName: "Customer"})`
3. **Auto-generated**: `BlogPost` -> "Blog Post" / "Blog Posts" with smart pluralization

## Quick Start
**Prerequisites**: Go 1.24+, templ CLI (for development)

```bash
# For development (if modifying templ files):
templ generate  # Regenerates *_templ.go files from *.templ files

# Run the demo with Go workspace support
go run examples/sql-example/main.go

# With authentication
go run examples/sql-example/main.go -auth=basic

# With SQL debug logging (helpful for checking app state)
DEBUG=true go run examples/sql-example/main.go

# Admin Panel: http://localhost:8080/admin/
```

**Basic Usage**:
```go
package main

import (
    "database/sql"
    "net/http"
    "time"
    
    sqladapter "backoffice/adapters/sql"
    "backoffice/core"
    "backoffice/middleware/auth"
    "backoffice/ui"
    
    _ "github.com/mattn/go-sqlite3"
)

type User struct {
    ID        uint      `json:"id" db:"id"`
    Name      string    `json:"name" db:"name"`
    Email     string    `json:"email" db:"email"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    Active    bool      `json:"active" db:"active"`
}

func main() {
    // Setup pure sql.DB database connection
    db, _ := sql.Open("sqlite3", "admin.db")
    
    // Create schema manually (no AutoMigrate)
    db.Exec(`CREATE TABLE IF NOT EXISTS users (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        name TEXT NOT NULL,
        email TEXT UNIQUE NOT NULL,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        active BOOLEAN DEFAULT 1
    )`)
    
    // Create BackOffice with pure SQL adapter
    adapter := sqladapter.New(db)
    admin := core.New(adapter, auth.WithNoAuth())
    
    // Register resources with fluent API
    admin.RegisterResource(&User{}).
        WithName("Customer").
        WithDefaultSort("CreatedAt", core.SortDesc).
        WithField("Name", func(f *core.FieldBuilder) {
            f.DisplayName("Full Name").Required(true).Searchable(true)
        }).
        WithDerivedField("AccountAge", "Account Age", func(user any) string {
            u := user.(*User)
            days := int(time.Since(u.CreatedAt).Hours() / 24)
            return fmt.Sprintf("%d days", days)
        }, func(f *core.FieldBuilder) {
            f.SortBy("CreatedAt", core.SortDesc)
        })
    
    // Setup HTTP routes
    http.Handle("/admin/", ui.Handler(admin, "/admin"))
    http.ListenAndServe(":8080", nil)
}
```

## Configuration
Resources are configured using the fluent builder API:

```go
type User struct {
    ID           uint        `json:"id" db:"id"`
    Name         string      `json:"name" db:"name"`
    DepartmentID *uint       `json:"department_id" db:"department_id"`
    Department   *Department `json:"department,omitempty" db:"-"`
    CreatedAt    time.Time   `json:"created_at" db:"created_at"`
}

admin.RegisterResource(&User{}).
    WithName("Customer").
    WithPluralName("Customers").
    WithDefaultSort("CreatedAt", core.SortDesc).
    WithField("Name", func(f *core.FieldBuilder) {
        f.DisplayName("Full Name").Required(true).Searchable(true)
    }).
    WithDerivedField("AccountAge", "Account Age", func(user any) string {
        u := user.(*User)
        days := int(time.Since(u.CreatedAt).Hours() / 24)
        return fmt.Sprintf("%d days", days)
    }, func(f *core.FieldBuilder) {
        f.SortBy("CreatedAt", core.SortDesc)
    }).
    WithManyToOneField("Department", "Department", func(r *core.RelationshipBuilder) {
        r.DisplayField("Name").CompactDisplay()
    })
```

## Adapters

### Built-in SQL Adapter
The pure `sql.DB` adapter is fully implemented and supports:
- **Standard Library Based**: Uses Go standard library `database/sql` without ORM dependencies
- **Universal Compatibility**: Works with SQLite, PostgreSQL, MySQL, and other SQL databases
- **Reflection-Based Scanning**: Type-safe struct mapping without external dependencies
- **Manual Schema Management**: Explicit DDL for production reliability
- **Full CRUD Operations**: Create, read, update, delete with pagination and sorting
- **Relationship Support**: Many-to-one relationships with automatic foreign key detection
- **Performance Optimized**: Efficient queries with proper indexing and prepared statements

### Architecture Benefits
- **Minimal Dependencies**: Core library uses only essential dependencies (templ, strcase)
- **Database Agnostic**: Works with any SQL database supported by Go's `database/sql`
- **Production Ready**: Manual schema management avoids unexpected migrations
- **Testable**: Easy to mock and test with standard Go testing patterns
- **Go Workspace Isolation**: Examples can use convenience libraries independently

Custom adapters can be implemented by satisfying the `core.Adapter` interface.

## Build & Deployment Strategy

### Minimal-Dependency Architecture
**Core Library**: Uses minimal external dependencies for maximum compatibility
- **Essential Dependencies**: `templ` (UI generation), `strcase` (field naming), `godotenv` (config), plus database drivers
- **Pre-compiled Templates**: Templ files compiled to Go code and committed to version control
- **High Compatibility**: Minimal dependency footprint reduces conflicts

### Go Workspace Support
**Development Flexibility**: Examples can use convenience libraries while keeping core clean
```
backoffice/
├── go.work              # Workspace configuration
├── go.mod               # Core (stdlib only)
└── examples/sql-example/
    └── go.mod          # Can use sqlx, etc.
```

**Benefits:**
- **Clean Separation**: Core library maintains minimal dependencies
- **Developer Experience**: Examples can use ergonomic libraries (sqlx, etc.)
- **Production Ready**: Minimal dependencies suitable for enterprise environments
- **Easy Integration**: Small dependency footprint reduces conflicts

### Development Workflow
**Contributors**: Need `templ` CLI to modify UI components
**Users**: Can consume library without any additional dependencies
**Examples**: Use Go workspace to run: `go run examples/sql-example/main.go`

**Why This Architecture?**
- Minimal runtime dependencies for core library (templ, strcase, godotenv)
- Faster builds with pre-compiled templates (no generation step for users)
- Standard Go tooling compatibility
- Enterprise-friendly with vetted, minimal dependency chain

## Current Status

**✅ IMPLEMENTED:**
- **Minimal-Dependency Architecture**: Pure `sql.DB` implementation with minimal external dependencies
- **Universal SQL Support**: Works with SQLite, PostgreSQL, MySQL, and other SQL databases
- **Core CRUD Operations**: Full create, read, update, delete with pagination and sorting
- **Smart Column Mapping**: Automatic field-to-column resolution with GORM compatibility
- **Resource Registration**: Fluent API with intelligent defaults and customization
- **Professional UI**: Tailwind CSS styling with HTMX interactions and smooth animations
- **Relationship Support**: Many-to-one relationships with multiple display patterns
- **Authentication Middleware**: Pluggable auth system (basic auth, no auth, custom)
- **Field Customization**: Derived fields, validation, display names, search configuration
- **Go Workspace Support**: Dependency isolation for development and examples
- **Comprehensive Testing**: Full test suite with 15+ test cases covering all scenarios
- **Working Demo Application**: Complete example with relationships and sample data
- **SQL Debug Logging**: GORM-style query logging with `DEBUG=true` environment variable

**Key Features:**
- **UI Patterns**: Side panes for quick edits, full pages for detailed views, toast notifications
- **Relationships**: Auto-detected with compact, badge, or hierarchical display patterns
- **Authentication**: Pluggable auth system with built-in basic auth and extensible interface
- **Smart Column Mapping**: Priority-based field-to-column resolution with GORM/sqlx compatibility
- **Reflection-Based**: Automatic field discovery and type-safe struct mapping
- **Production Ready**: Manual schema management and explicit DDL for reliability

## Architecture

**Key Design Patterns:**
- **Minimal-Dependency Core**: Lean approach with only essential dependencies for maximum compatibility
- **Adapter Pattern**: Pluggable data sources with pure `sql.DB` as reference implementation
- **Reflection-Based Discovery**: Automatic field detection and struct mapping without external libs
- **HTML-First UI**: Semantic HTML enhanced with HTMX for smooth async interactions
- **Pre-compiled Templates**: Templ components compiled to Go code for zero runtime dependencies
- **Go Workspace Isolation**: Examples can use convenience libraries while core stays clean
- **Manual Schema Management**: Explicit DDL instead of automatic migrations for production safety

**Implementation Highlights:**
- **Pure sql.DB Scanning**: Custom reflection-based row scanning without sqlx or similar
- **Prepared Statements**: Efficient query execution with parameter binding
- **Relationship Loading**: Foreign key detection and related data fetching
- **Type Safety**: Compile-time template validation and runtime type checking
- **Database Agnostic**: Works with any database supporting Go's `database/sql`

## Known Limitations & Future Enhancements

**Not Yet Implemented:**
- **OTP Authentication**: Stub exists in `middleware/auth/otp.go` but panics if called
- **Many-to-Many Relationships**: Only many-to-one is currently supported
- **Role-Based Access Control**: Authentication exists but no fine-grained permissions
- **File Uploads**: No support for file/image fields yet

**Post-MVP Enhancements (Future):**
- **Additional Adapters**: Ent adapter, NoSQL adapters (MongoDB, DynamoDB)
- **Advanced Query Features**: Complex filtering, full-text search, aggregations
- **File Upload Support**: Image and document handling with storage backends
- **Custom Field Types**: Rich text editors, date pickers, select components
- **Bulk Operations**: Multi-select actions, batch updates, bulk imports
- **Export Functionality**: CSV, Excel, JSON export with filtering
- **Advanced Relationships**: Many-to-many, polymorphic relationships
- **Audit Trails**: Change tracking and version history
- **Real-time Updates**: WebSocket-based live data updates
- **Performance Optimizations**: Query optimization, caching strategies
- **Security Enhancements**: Role-based access control, field-level permissions