# Partly taken from https://about.gitlab.com/blog/2017/11/27/go-tools-and-gitlab-how-to-do-continuous-integration-like-a-boss/

.PHONY: all dep build test coverage coverhtml lint

all: build

lint: ## Lint the files
	golint -set_exit_status ./...

test: ## Run unittests
	go test -short ./...

race: dep ## Run data race detector
	go test -race -short ./...

msan: dep ## Run memory sanitizer
	CC=clang CXX=clang++ go test -msan -short ./...

dep: ## Get the dependencies
	go get -v -d ./...

build: dep ## Build the binary file
	go build -i -v main.go
