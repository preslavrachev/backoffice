module sql-example

go 1.24

replace backoffice => ../../

require (
	github.com/jmoiron/sqlx v1.4.0
	github.com/mattn/go-sqlite3 v1.14.32
	github.com/preslavrachev/backoffice v0.0.0-20251010080646-cfa374ae58f5
)

require (
	github.com/a-h/templ v0.3.924 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
)
