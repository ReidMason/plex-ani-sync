test-cover:
  go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

sqlc-generate:
  cd db/sqlc && sqlc generate 

migrate-up:
  migrate -source file://db/migrations -database pgx://admin:admin@localhost:5432/plexanilistsync up

tailwind:
  npx tailwindcss -i ./templates/input.css -o ./public/assets/css/style.css --watch 

templ:
  templ generate -watch -proxy="http://localhost:8000/"
