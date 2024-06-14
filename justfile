test-cover:
  go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

sqlc-generate:
  cd internal/storage/sqlc && sqlc generate 

tailwind:
  ./tailwindcss -i ./templates/input.css -o ./public/assets/css/style.css --watch 

templ:
  templ generate -watch -proxy="http://localhost:8000/"

new-migration name:
  migrate create -ext sql -dir ./db/migrations -seq "{{name}}"
