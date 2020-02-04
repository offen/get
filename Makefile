build: main.go
	@go mod download
	@go build -o ./bin/get main.go
