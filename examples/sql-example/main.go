package main

import (
	"backoffice/config"
	"backoffice/core"
	"backoffice/middleware/auth"
	"backoffice/ui"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	sqladapter "backoffice/adapters/sql"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

type Department struct {
	ID          uint   `json:"id" db:"id"`
	Name        string `json:"name" db:"name"`
	Location    string `json:"location" db:"location"`
	Budget      int    `json:"budget" db:"budget"`
	ManagerName string `json:"manager_name" db:"manager_name"`
	MemberCount int    `json:"member_count" db:"member_count"`
}

type User struct {
	ID           uint         `json:"id" db:"id"`
	Name         string       `json:"name" db:"name"`
	Email        string       `json:"email" db:"email"`
	DepartmentID *uint        `json:"department_id" db:"department_id"`
	Department   *Department  `json:"department,omitempty" db:"-"`
	CreatedAt    time.Time    `json:"created_at" db:"created_at"`
	Role         string       `json:"role" db:"role"`
	Active       bool         `json:"active" db:"active"`
	Status       string       `json:"status" db:"status"`
	TrialEndDate sql.NullTime `json:"trial_end_date" db:"trial_end_date"`
	CancelledAt  sql.NullTime `json:"cancelled_at" db:"cancelled_at"`
}

type Product struct {
	ID          uint      `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	Price       float64   `json:"price" db:"price"`
	CategoryID  uint      `json:"category_id" db:"category_id"`
	Category    *Category `json:"category,omitempty" db:"-"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Category struct {
	ID        uint      `json:"id" db:"id"`
	Name      string    `json:"name" db:"name"`
	ParentID  *uint     `json:"parent_id" db:"parent_id"`
	Parent    *Category `json:"parent,omitempty" db:"-"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	SortOrder int       `json:"sort_order" db:"sort_order"`
}

func main() {
	// Add flags
	debug := flag.Bool("debug", false, "Enable SQL debug logging")
	authMode := flag.String("auth", "none", "Authentication mode: none, basic")
	flag.Parse()

	// Set DEBUG environment variable if -debug flag is used
	if *debug {
		os.Setenv("DEBUG", "true")
		fmt.Println("🐛 SQL Debug mode enabled via DEBUG=true")
	}

	// Load configuration including DEBUG environment variable
	cfg := config.LoadConfig()

	// Open SQLite database
	db, err := sql.Open("sqlite3", "example.db")
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}
	defer db.Close()

	// Create sqlx wrapper
	dbx := sqlx.NewDb(db, "sqlite3")

	// Create database schema
	err = createSchema(dbx)
	if err != nil {
		log.Fatal("failed to create database schema:", err)
	}

	// Seed sample data
	seedData(dbx)

	// Create BackOffice admin with SQL adapter
	setupAdmin(dbx, *authMode, cfg)
}

func createSchema(db *sqlx.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS departments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		location TEXT NOT NULL,
		budget INTEGER NOT NULL,
		manager_name TEXT NOT NULL,
		member_count INTEGER NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		email TEXT UNIQUE NOT NULL,
		department_id INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		role TEXT NOT NULL,
		active BOOLEAN DEFAULT 1,
		status TEXT DEFAULT '',
		trial_end_date DATETIME,
		cancelled_at DATETIME,
		FOREIGN KEY (department_id) REFERENCES departments(id)
	);

	CREATE TABLE IF NOT EXISTS categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		parent_id INTEGER,
		is_active BOOLEAN DEFAULT 1,
		sort_order INTEGER DEFAULT 0,
		FOREIGN KEY (parent_id) REFERENCES categories(id)
	);

	CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT,
		price REAL NOT NULL,
		category_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (category_id) REFERENCES categories(id)
	);
	`

	_, err := db.Exec(schema)
	return err
}

func setupAdmin(db *sqlx.DB, authMode string, cfg *config.Config) {
	// Create SQL adapter with debug logging - pass the underlying sql.DB to the pure adapter
	sqlAdapter := sqladapter.NewWithDebug(db.DB, cfg.DebugEnabled)

	// Configure authentication based on mode
	var authConfig auth.AuthConfig
	switch authMode {
	case "basic":
		authConfig = auth.WithBasicAuthFromConfig()
		fmt.Println("🔐 Basic Authentication enabled")
		fmt.Println("   👤 Credentials loaded from environment/config")
	case "none":
	default:
		authConfig = auth.WithNoAuth()
		fmt.Println("🚫 Authentication disabled")
	}

	admin := core.New(sqlAdapter, authConfig)

	// Register Department with basic setup
	admin.RegisterResource(&Department{}).
		WithField("Name", func(f *core.FieldBuilder) {
			f.DisplayName("Department Name").Required(true).Searchable(true)
		}).
		WithField("ManagerName", func(f *core.FieldBuilder) {
			f.DisplayName("Manager")
		}).
		WithField("MemberCount", func(f *core.FieldBuilder) {
			f.DisplayName("Team Size")
		})

	// Register User with compact relationship display (default) and CreatedAt DESC sorting
	admin.RegisterResource(&User{}).
		WithName("Employee").
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
			if days == 0 {
				return "Today"
			} else if days == 1 {
				return "1 day"
			}
			return fmt.Sprintf("%d days", days)
		}, func(f *core.FieldBuilder) {
			f.SortBy("CreatedAt", core.SortDesc)
		}).
		WithDerivedField("CustomerStatus", "Customer Status", func(user any) string {
			u := user.(*User)
			// Trial ended
			if u.TrialEndDate.Valid && u.TrialEndDate.Time.Before(time.Now()) {
				if u.CancelledAt.Valid {
					return "Canceled: " + u.CancelledAt.Time.Format("Jan 02, 2006")
				}
				return "Trial Ended"
			}

			// Active customer
			if u.Status == "active" {
				return "Customer"
			}

			// Active trial
			if u.TrialEndDate.Valid {
				days := int(time.Until(u.TrialEndDate.Time).Hours() / 24)
				if days <= 0 {
					return "Trial: 0d left"
				}
				return fmt.Sprintf("Trial: %dd left", days)
			}

			return "Unknown"
		}).
		WithManyToOneField("Department", "Department", func(r *core.RelationshipBuilder) {
			r.DisplayField("Name").CompactDisplay() // Compact display in lists
		})

	// Register Product with badge relationship display and Price DESC sorting
	admin.RegisterResource(&Product{}).
		WithDefaultSort("Price", core.SortDesc).
		WithField("Name", func(f *core.FieldBuilder) {
			f.DisplayName("Product Name").Required(true).Searchable(true)
		}).
		WithField("Price", func(f *core.FieldBuilder) {
			f.DisplayName("Price ($)").Required(true)
		}).
		WithManyToOneField("Category", "Category", func(r *core.RelationshipBuilder) {
			r.DisplayField("Name").BadgeDisplay() // Badge display in lists
		})

	// Register Category with hierarchical relationship display and SortOrder ASC sorting
	admin.RegisterResource(&Category{}).
		WithDefaultSort("SortOrder", core.SortAsc).
		WithField("Name", func(f *core.FieldBuilder) {
			f.DisplayName("Category Name").Required(true).Searchable(true)
		}).
		WithField("IsActive", func(f *core.FieldBuilder) {
			f.DisplayName("Active")
		}).
		WithField("SortOrder", func(f *core.FieldBuilder) {
			f.DisplayName("Sort Order")
		}).
		WithField("ParentID", func(f *core.FieldBuilder) {
			f.ReadOnly(true) // Hide foreign key from main display
		}).
		WithManyToOneField("Parent", "Category", func(r *core.RelationshipBuilder) {
			r.DisplayField("Name").ForeignKey("ParentID").HierarchicalDisplay() // Hierarchical display in lists
		})

	// Setup HTTP server using UI package
	http.Handle("/admin/", ui.Handler(admin, "/admin"))

	fmt.Println()
	fmt.Println("🚀 BackOffice Admin Panel started!")
	fmt.Println("📱 Visit: http://localhost:8080/admin/")
	if authMode == "basic" {
		fmt.Println("🔐 Login required - use credentials above")
	}
	fmt.Println()
	fmt.Println("📊 Available Resources:")
	fmt.Println("  • Department - Basic resource (no relationships)")
	fmt.Println("  • Employee (User) - Compact relationship display")
	fmt.Println("  • Product - Badge relationship display")
	fmt.Println("  • Category - Hierarchical relationship display")
	fmt.Println()
	fmt.Println("🔗 Relationship Patterns Demonstrated:")
	fmt.Println("  • Employee → Department (Many-to-One, Compact)")
	fmt.Println("  • Product → Category (Many-to-One, Badge)")
	fmt.Println("  • Category → Parent Category (Many-to-One, Hierarchical)")
	fmt.Println()
	fmt.Println("🌐 API endpoints:")
	fmt.Println("  GET /admin/api/Department")
	fmt.Println("  GET /admin/api/User")
	fmt.Println("  GET /admin/api/Product")
	fmt.Println("  GET /admin/api/Category")
	fmt.Println()
	fmt.Println("💡 Usage examples:")
	fmt.Println("  # No authentication:")
	fmt.Println("  go run examples/sql-example/main.go")
	fmt.Println("  # With basic authentication:")
	fmt.Println("  go run examples/sql-example/main.go -auth=basic")
	fmt.Println("  # With SQL debug logging:")
	fmt.Println("  DEBUG=true go run examples/sql-example/main.go")
	fmt.Println("  # With debug flag (equivalent):")
	fmt.Println("  go run examples/sql-example/main.go -debug")
	fmt.Println("  # Authentication + debug:")
	fmt.Println("  go run examples/sql-example/main.go -auth=basic -debug")

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func seedData(db *sqlx.DB) {
	// Clear existing data
	db.Exec("DELETE FROM products")
	db.Exec("DELETE FROM categories")
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM departments")

	// Create departments (15+ for pagination)
	departments := []Department{
		{Name: "Engineering", Location: "San Francisco", Budget: 500000, ManagerName: "Alice Johnson", MemberCount: 12},
		{Name: "Marketing", Location: "New York", Budget: 300000, ManagerName: "Bob Wilson", MemberCount: 8},
		{Name: "Sales", Location: "Chicago", Budget: 250000, ManagerName: "Carol Davis", MemberCount: 15},
		{Name: "HR", Location: "Austin", Budget: 150000, ManagerName: "Dave Brown", MemberCount: 5},
		{Name: "Product", Location: "Seattle", Budget: 450000, ManagerName: "Eva White", MemberCount: 10},
		{Name: "Design", Location: "Los Angeles", Budget: 280000, ManagerName: "Frank Green", MemberCount: 6},
		{Name: "Finance", Location: "Boston", Budget: 320000, ManagerName: "Grace Miller", MemberCount: 8},
		{Name: "Legal", Location: "Washington DC", Budget: 200000, ManagerName: "Henry Taylor", MemberCount: 4},
		{Name: "Operations", Location: "Denver", Budget: 380000, ManagerName: "Ivy Anderson", MemberCount: 14},
		{Name: "Security", Location: "Phoenix", Budget: 220000, ManagerName: "Jack Thomas", MemberCount: 7},
		{Name: "Quality Assurance", Location: "Portland", Budget: 180000, ManagerName: "Kelly Jackson", MemberCount: 9},
		{Name: "Customer Success", Location: "Miami", Budget: 160000, ManagerName: "Liam Garcia", MemberCount: 11},
		{Name: "Data Science", Location: "Dallas", Budget: 400000, ManagerName: "Mia Rodriguez", MemberCount: 8},
		{Name: "DevOps", Location: "Atlanta", Budget: 350000, ManagerName: "Noah Lewis", MemberCount: 6},
		{Name: "Business Development", Location: "Nashville", Budget: 240000, ManagerName: "Olivia Walker", MemberCount: 5},
	}

	for _, dept := range departments {
		_, err := db.NamedExec(`
			INSERT INTO departments (name, location, budget, manager_name, member_count) 
			VALUES (:name, :location, :budget, :manager_name, :member_count)
		`, dept)
		if err != nil {
			log.Printf("Error inserting department: %v", err)
		}
	}

	// Create parent categories (6 for variety)
	parentCategories := []Category{
		{Name: "Electronics", IsActive: true, SortOrder: 1},
		{Name: "Books", IsActive: true, SortOrder: 2},
		{Name: "Clothing", IsActive: true, SortOrder: 3},
		{Name: "Sports", IsActive: true, SortOrder: 4},
		{Name: "Home & Garden", IsActive: true, SortOrder: 5},
		{Name: "Automotive", IsActive: true, SortOrder: 6},
	}

	var parentIDs []uint
	for _, cat := range parentCategories {
		result, err := db.NamedExec(`
			INSERT INTO categories (name, is_active, sort_order) 
			VALUES (:name, :is_active, :sort_order)
		`, cat)
		if err != nil {
			log.Printf("Error inserting parent category: %v", err)
			continue
		}
		id, _ := result.LastInsertId()
		parentIDs = append(parentIDs, uint(id))
	}

	// Create child categories (20+ for pagination)
	childCategories := []struct {
		Name      string
		ParentIdx int
		IsActive  bool
		SortOrder int
	}{
		// Electronics subcategories
		{"Laptops", 0, true, 1},
		{"Phones", 0, true, 2},
		{"Tablets", 0, true, 3},
		{"Headphones", 0, true, 4},
		{"Smart Watches", 0, true, 5},

		// Books subcategories
		{"Programming", 1, true, 1},
		{"Fiction", 1, true, 2},
		{"Science", 1, true, 3},
		{"History", 1, true, 4},
		{"Biography", 1, true, 5},

		// Clothing subcategories
		{"T-Shirts", 2, true, 1},
		{"Jeans", 2, true, 2},
		{"Dresses", 2, true, 3},
		{"Shoes", 2, true, 4},
		{"Accessories", 2, false, 5},

		// Sports subcategories
		{"Fitness", 3, true, 1},
		{"Outdoor", 3, true, 2},
		{"Team Sports", 3, true, 3},
		{"Water Sports", 3, true, 4},
		{"Winter Sports", 3, true, 5},

		// Home & Garden subcategories
		{"Furniture", 4, true, 1},
		{"Garden Tools", 4, true, 2},
		{"Decor", 4, true, 3},
		{"Kitchen", 4, true, 4},

		// Automotive subcategories
		{"Parts", 5, true, 1},
		{"Tools", 5, true, 2},
		{"Accessories", 5, true, 3},
	}

	var childIDs []uint
	for _, cat := range childCategories {
		result, err := db.Exec(`
			INSERT INTO categories (name, parent_id, is_active, sort_order) 
			VALUES (?, ?, ?, ?)
		`, cat.Name, parentIDs[cat.ParentIdx], cat.IsActive, cat.SortOrder)
		if err != nil {
			log.Printf("Error inserting child category: %v", err)
			continue
		}
		id, _ := result.LastInsertId()
		childIDs = append(childIDs, uint(id))
	}

	// Create products with relationships (30+ for pagination)
	products := []struct {
		Name        string
		Description string
		Price       float64
		CategoryIdx int
	}{
		// Electronics products
		{"MacBook Pro", "Professional laptop", 1299.99, 0},
		{"ThinkPad X1", "Business laptop", 999.99, 0},
		{"Dell XPS 13", "Ultrabook laptop", 849.99, 0},
		{"iPhone 15 Pro", "Latest smartphone", 699.99, 1},
		{"Samsung Galaxy S24", "Android flagship", 649.99, 1},
		{"Google Pixel 8", "AI-powered phone", 599.99, 1},
		{"iPad Pro", "Professional tablet", 799.99, 2},
		{"Surface Pro", "2-in-1 tablet", 899.99, 2},
		{"Sony WH-1000XM5", "Noise cancelling headphones", 299.99, 3},
		{"AirPods Pro", "Wireless earbuds", 249.99, 3},
		{"Apple Watch Ultra", "Rugged smartwatch", 799.99, 4},
		{"Garmin Fenix 7", "GPS fitness watch", 699.99, 4},

		// Books
		{"Go Programming Guide", "Learn Go programming", 49.99, 5},
		{"Clean Code", "A handbook of agile software craftsmanship", 39.99, 5},
		{"System Design Interview", "An insider's guide", 44.99, 5},
		{"The Hobbit", "Fantasy adventure novel", 14.99, 6},
		{"1984", "Dystopian social science fiction", 13.99, 6},
		{"Dune", "Science fiction epic", 16.99, 6},
		{"A Brief History of Time", "Cosmology for general readers", 18.99, 7},
		{"Sapiens", "A brief history of humankind", 22.99, 8},
		{"Steve Jobs", "Biography of Apple's founder", 17.99, 9},

		// Clothing
		{"Cotton T-Shirt", "Comfortable cotton tee", 19.99, 10},
		{"Premium T-Shirt", "High-quality organic cotton", 29.99, 10},
		{"Levi's 501 Jeans", "Classic straight leg jeans", 89.99, 11},
		{"Skinny Jeans", "Modern fit denim", 69.99, 11},
		{"Summer Dress", "Light cotton dress", 79.99, 12},
		{"Running Shoes", "Lightweight running shoes", 129.99, 13},
		{"Leather Belt", "Genuine leather accessory", 49.99, 14},

		// Sports equipment
		{"Yoga Mat", "Non-slip exercise mat", 39.99, 15},
		{"Camping Tent", "2-person waterproof tent", 199.99, 16},
		{"Basketball", "Official size basketball", 29.99, 17},
	}

	for _, prod := range products {
		if prod.CategoryIdx < len(childIDs) {
			_, err := db.Exec(`
				INSERT INTO products (name, description, price, category_id, created_at) 
				VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP)
			`, prod.Name, prod.Description, prod.Price, childIDs[prod.CategoryIdx])
			if err != nil {
				log.Printf("Error inserting product: %v", err)
			}
		}
	}

	// Create users with department relationships
	now := time.Now()
	trialEnd := sql.NullTime{Time: now.Add(5 * 24 * time.Hour), Valid: true}       // Trial ending in 5 days
	trialEnded := sql.NullTime{Time: now.Add(-10 * 24 * time.Hour), Valid: true}   // Trial ended 10 days ago
	cancelledDate := sql.NullTime{Time: now.Add(-5 * 24 * time.Hour), Valid: true} // Cancelled 5 days ago

	users := []struct {
		Name         string
		Email        string
		DeptIdx      *int
		Role         string
		Active       bool
		Status       string
		CreatedDays  int
		TrialEndDate sql.NullTime
		CancelledAt  sql.NullTime
	}{
		// Engineering team
		{"John Doe", "john@example.com", &[]int{0}[0], "Senior Developer", true, "active", -30, sql.NullTime{}, sql.NullTime{}},
		{"Jane Smith", "jane@example.com", &[]int{0}[0], "Frontend Developer", true, "", -2, trialEnd, sql.NullTime{}},
		{"Alex Johnson", "alex@example.com", &[]int{0}[0], "Backend Developer", true, "active", -25, sql.NullTime{}, sql.NullTime{}},
		{"Emily Chen", "emily@example.com", &[]int{0}[0], "DevOps Engineer", true, "active", -18, sql.NullTime{}, sql.NullTime{}},
		{"Ryan Miller", "ryan@example.com", &[]int{0}[0], "Tech Lead", true, "active", -45, sql.NullTime{}, sql.NullTime{}},

		// Marketing team
		{"Mike Johnson", "mike@example.com", &[]int{1}[0], "Marketing Manager", true, "active", 0, sql.NullTime{}, sql.NullTime{}},
		{"Sarah Wilson", "sarah@example.com", &[]int{1}[0], "Content Creator", true, "", -15, trialEnded, sql.NullTime{}},
		{"Amanda Rodriguez", "amanda@example.com", &[]int{1}[0], "Digital Marketer", true, "active", -22, sql.NullTime{}, sql.NullTime{}},
		{"Kevin Park", "kevin@example.com", &[]int{1}[0], "SEO Specialist", true, "active", -12, sql.NullTime{}, sql.NullTime{}},

		// Sales team
		{"David Brown", "david@example.com", &[]int{2}[0], "Sales Representative", true, "", -20, trialEnded, cancelledDate},
		{"Michelle Garcia", "michelle@example.com", &[]int{2}[0], "Account Manager", true, "active", -35, sql.NullTime{}, sql.NullTime{}},
		{"Robert Kim", "robert@example.com", &[]int{2}[0], "Sales Director", true, "active", -50, sql.NullTime{}, sql.NullTime{}},
		{"Jennifer Lee", "jennifer@example.com", &[]int{2}[0], "Business Development", true, "active", -8, sql.NullTime{}, sql.NullTime{}},

		// HR team
		{"Lisa Davis", "lisa@example.com", &[]int{3}[0], "HR Specialist", true, "active", -1, sql.NullTime{}, sql.NullTime{}},
		{"Michael Thompson", "michael@example.com", &[]int{3}[0], "Recruiter", true, "active", -28, sql.NullTime{}, sql.NullTime{}},
		{"Nicole Williams", "nicole@example.com", &[]int{3}[0], "HR Manager", true, "active", -40, sql.NullTime{}, sql.NullTime{}},

		// Contractors and inactive users
		{"Tom Anderson", "tom@example.com", nil, "Contractor", false, "", -7, sql.NullTime{}, sql.NullTime{}},
		{"Rachel Green", "rachel@example.com", nil, "Freelance Designer", false, "", -13, sql.NullTime{}, sql.NullTime{}},
	}

	for _, user := range users {
		var deptID *uint
		if user.DeptIdx != nil && *user.DeptIdx < len(parentIDs) {
			// Use department ID from the inserted departments
			var id uint
			err := db.Get(&id, "SELECT id FROM departments ORDER BY id LIMIT 1 OFFSET ?", *user.DeptIdx)
			if err == nil {
				deptID = &id
			}
		}

		createdAt := now.Add(time.Duration(user.CreatedDays) * 24 * time.Hour)

		_, err := db.Exec(`
			INSERT INTO users (name, email, department_id, role, active, status, created_at, trial_end_date, cancelled_at) 
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, user.Name, user.Email, deptID, user.Role, user.Active, user.Status, createdAt,
			user.TrialEndDate.Time, user.CancelledAt.Time)
		if err != nil {
			log.Printf("Error inserting user %s: %v", user.Name, err)
		}
	}
}
