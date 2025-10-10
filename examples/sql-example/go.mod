module sql-example

go 1.24

replace backoffice => ../../

require (
	backoffice v0.0.0-00010101000000-000000000000
	github.com/jmoiron/sqlx v1.4.0
	github.com/mattn/go-sqlite3 v1.14.32
)

require (
	github.com/a-h/templ v0.3.924 // indirect
	github.com/iancoleman/strcase v0.3.0 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
)
