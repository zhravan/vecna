BINARY := vecna
VERSION ?= $(shell cat version.txt 2>/dev/null | tr -d ' \n\r' || echo "dev")
LDFLAGS := -ldflags "-s -w -X github.com/zhravan/vecna/cmd.Version=$(VERSION)"

.PHONY: build run clean install lint test version bump-patch bump-minor bump-major

build:
	go build $(LDFLAGS) -o bin/$(BINARY) .

run:
	go run .

clean:
	rm -rf bin/

install:
	go install $(LDFLAGS) .

lint:
	golangci-lint run

test:
	go test -v ./...

# Version (semver from version.txt). Bump before every master merge/push.
version:
	@echo "$(VERSION)"

# Bump patch version (x.y.z -> x.y.z+1). Run before merging to master.
bump-patch:
	@v=$$(cat version.txt | tr -d ' \n\r'); \
	maj=$$(echo "$$v" | cut -d. -f1); \
	min=$$(echo "$$v" | cut -d. -f2); \
	pat=$$(echo "$$v" | cut -d. -f3); \
	new="$$maj.$$min.$$((pat+1))"; \
	echo "$$new" > version.txt && echo "version.txt -> $$new"

# Bump minor version (x.y.z -> x.(y+1).0).
bump-minor:
	@v=$$(cat version.txt | tr -d ' \n\r'); \
	maj=$$(echo "$$v" | cut -d. -f1); \
	min=$$(echo "$$v" | cut -d. -f2); \
	new="$$maj.$$((min+1)).0"; \
	echo "$$new" > version.txt && echo "version.txt -> $$new"

# Bump major version (x.y.z -> (x+1).0.0).
bump-major:
	@v=$$(cat version.txt | tr -d ' \n\r'); \
	maj=$$(echo "$$v" | cut -d. -f1); \
	new="$$((maj+1)).0.0"; \
	echo "$$new" > version.txt && echo "version.txt -> $$new"
