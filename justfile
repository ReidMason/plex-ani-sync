test-cover:
  go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

sqlc-generate:
  cd db/sqlc && sqlc generate 

migrate-up:
  migrate -source file://db/migrations -database pgx://testuser:testpass@localhost:5432/plexAnilistSync up

build:
  npx tailwindcss -i ./view/input.css -o ./public/assets/css/style.css && templ generate && go build -o ./tmp/main .
