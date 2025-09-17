run:
	go run cmd/main.go

test:
	go test -count=1 .\internal\...

test-coverage:
	go test -count=1 -coverprofile=coverage.out .\internal\...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

lint:
	golangci-lint fmt && golangci-lint run

migration-create:
	goose create $(name) sql

migration-up:
	goose up

migration-down:
	goose down

swagger:
	swag init -g .\cmd\main.go -o swagger
