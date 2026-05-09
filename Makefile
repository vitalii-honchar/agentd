GO ?= go

.PHONY: install runserver test

install:
	$(GO) install ./cmd/agentd ./cmd/agentdserver

runserver:
	$(GO) run ./cmd/agentdserver

test:
	$(GO) test ./...
