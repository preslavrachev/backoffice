# BackOffice

> Django Admin for Go - Build internal tools and admin panels in minutes, not days

Stop writing the same CRUD boilerplate. Define your structs, point BackOffice at your database, and get a professional admin UI instantly.

## Demo

![](https://preslav.me/img/backoffice-demo.gif)

## Why BackOffice?

**Before**: You used to spend days building user management, product catalog, order history pages. Each one needs: list view, search, pagination, forms, validation, edit modals. Repeat for every model.

**After**:
```go
admin.RegisterResource(&User{})
admin.RegisterResource(&Product{})
admin.RegisterResource(&Order{})
// Done. You have a complete admin panel with CRUD, relationships, auth.
```

You just saved 3 days. Use them to build features customers actually pay for.

## Features

- **Minimal Dependencies**: Only templ (UI), strcase (naming), godotenv (config), plus your database driver
- **Pure sql.DB Support**: Works with standard library `database/sql` - no ORM required
- **Universal Database Support**: Works with SQLite, PostgreSQL, MySQL, and other SQL databases
- **Selective Resource Registration**: Only explicitly registered structs appear in admin panel
- **Smart Resource Naming**: Auto-generated display names with intelligent pluralization
- **Full CRUD Operations**: Create, read, update, delete with form validation and pagination
- **Dynamic Forms**: Auto-generated based on struct field metadata with optional configuration
- **Modern UI**: Tailwind CSS styling with HTMX interactivity
- **Type-Safe Templates**: Pre-compiled templ components for reliable rendering
- **Fluent API**: Easy resource registration with method chaining
- **Relationship Support**: Many-to-one relationships with multiple display patterns
- **Authentication Middleware**: Pluggable auth system (basic auth, no auth, custom)
- **Derived Fields**: Dynamic fields calculated at runtime
- **Smart Column Mapping**: Automatic field-to-column resolution with GORM compatibility
- **SQL Debug Logging**: Enable with `DEBUG=true` for GORM-style query logging

## Quick Start

### Installation

```bash
go get github.com/preslavrachev/backoffice
```

### Basic Usage

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
    // Setup database
    db, _ := sql.Open("sqlite3", "admin.db")
    
    // Create schema
    db.Exec(`
        CREATE TABLE IF NOT EXISTS users (
            id INTEGER PRIMARY KEY AUTOINCREMENT,
            name TEXT NOT NULL,
            email TEXT UNIQUE NOT NULL,
            created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
            active BOOLEAN DEFAULT 1
        )
    `)
    
    // Create BackOffice admin
    adapter := sqladapter.New(db)
    admin := core.New(adapter, auth.WithNoAuth())
    
    // Register resources with fluent API
    admin.RegisterResource(&User{}).
        WithName("Customer").
        WithDefaultSort("CreatedAt", core.SortDesc).
        WithField("Name", func(f *core.FieldBuilder) {
            f.DisplayName("Full Name").Required(true).Searchable(true)
        }).
        WithField("Email", func(f *core.FieldBuilder) {
            f.DisplayName("Email Address").Required(true).Unique(true)
        })
    
    // Setup HTTP server
    http.Handle("/admin/", ui.Handler(admin, "/admin"))
    http.ListenAndServe(":8080", nil)
}
```

Visit `http://localhost:8080/admin/` to access the admin panel.

## Working Demo

Try the included example with Go workspace support:

```bash
# From project root
go run examples/sql-example/main.go

# With authentication
go run examples/sql-example/main.go -auth=basic

# With SQL debug logging
DEBUG=true go run examples/sql-example/main.go
```

The demo includes Department, User, Product, and Category models with relationships and comprehensive sample data.

## Go Workspace Architecture

Uses Go workspaces to keep core lean while examples can use convenience libraries:
```
go.work         # Core uses minimal deps (templ, strcase, godotenv)
examples/       # Examples can use sqlx, etc. without polluting core
```

## Resource Configuration

### Fluent API

```go
admin.RegisterResource(&Product{}).
    WithName("Item").
    WithPluralName("Items").
    WithDefaultSort("Price", core.SortDesc).
    WithField("Price", func(f *core.FieldBuilder) {
        f.DisplayName("Price ($)").Required(true)
    }).
    WithManyToOneField("Category", "Category", func(r *core.RelationshipBuilder) {
        r.DisplayField("Name").BadgeDisplay()
    })
```

### Derived Fields

Add computed fields that don't exist in your database:
```go
admin.RegisterResource(&User{}).
    WithDerivedField("AccountAge", "Account Age", func(user any) string {
        days := int(time.Since(user.(*User).CreatedAt).Hours() / 24)
        return fmt.Sprintf("%d days", days)
    })
```

### Database Column Mapping

Priority: `db:""` tag > `gorm:"column:"` > `json:""` > snake_case. Override with `WithDBColumnName()`.

### Relationship Support

Define many-to-one relationships:

```go
type User struct {
    DepartmentID *uint       `json:"department_id" db:"department_id"`
    Department   *Department `json:"department,omitempty" db:"-"`
}

admin.RegisterResource(&User{}).
    WithManyToOneField("Department", "Department", func(r *core.RelationshipBuilder) {
        r.DisplayField("Name").CompactDisplay()
    })
```

## Adapters

**Built-in**: SQL adapter using pure `database/sql` (SQLite, PostgreSQL, MySQL)
**Custom**: Implement `core.Adapter` interface for other data sources

## Contributing

**Contributors need**: Go 1.24+, [templ CLI](https://templ.guide/)
**End users need**: Just Go - templates are pre-compiled

Modify `.templ` files → run `templ generate` → commit both `.templ` and `*_templ.go`

## What's Implemented

✅ Full CRUD with pagination, sorting, validation
✅ Many-to-one relationships, derived fields
✅ Basic auth, session management
✅ HTMX-powered UI, Tailwind CSS

❌ Not yet: Many-to-many relationships, file uploads, RBAC