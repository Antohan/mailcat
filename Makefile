export PROJECT_ROOT=$(shell pwd)

test:
	@go test ./...

lint:
	@golangci-lint run ./...

reinstall:
	@go build -o mailcat ./cmd/mailcat \
		&& mv mailcat /usr/local/bin \
		&& echo "Приложение обновлено"
