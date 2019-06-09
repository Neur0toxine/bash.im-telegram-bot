.DEFAULT_GOAL := build

GO=$(shell which go)
PROJECT_DIR=$(shell pwd)
GOPATH=$(PROJECT_DIR)
SRC=$(PROJECT_DIR)/src
BIN=$(PROJECT_DIR)/bin/bash_im_bot

build: fmt deps
	@echo "- Building"
	@cd $(SRC) && $(GO) build -o $(BIN)
	@echo Built "$(BIN)"

run:
	@$(BIN)

deps:
	@echo "- Installing dependencies"
	@$(GO) mod tidy

fmt:
	@echo "- Running 'go fmt'"
	@$(GO) fmt $(SRC)