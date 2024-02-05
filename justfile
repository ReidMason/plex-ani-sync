test-cover:
  go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

sqlc-generate:
  cd db/sqlc && sqlc generate 
