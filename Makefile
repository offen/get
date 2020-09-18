DOCKER_TAG ?= latest

build: main.go Dockerfile go.mod
	@docker build -t offen/get:${DOCKER_TAG} .

serve:
	@PORT=4000 go run main.go
