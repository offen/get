DOCKER_TAG ?= latest

build: main.go Dockerfile
	@docker build -t offen/get:${DOCKER_TAG} .
