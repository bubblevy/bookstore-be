export DATABASE_URL="postgres://postgres:postgres@localhost:5432/bookstore?sslmode=disable"
migrate -database "$DATABASE_URL" -path ./migrations "$@"