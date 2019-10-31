GOVERSION=$(shell go version)
GOOS=$(word 1,$(subst /, ,$(word $(words $(GOVERSION)), $(GOVERSION))))
GOARCH=$(word 2,$(subst /, ,$(word $(words $(GOVERSION)), $(GOVERSION))))
VERSION=$(patsubst "%",%,$(lastword $(shell grep 'const Version' schemalex.go)))
ARTIFACTS_DIR=$(CURDIR)/artifacts/$(VERSION)
RELEASE_DIR=$(CURDIR)/release/$(VERSION)
SRC_FILES = $(wildcard *.go model/*.go diff/*.go cmd/schemalex/*.go internal/*/*.go)
GITHUB_USERNAME=schemalex

installdeps:
	go get -d
	go mod tidy

test:
	go test -v ./...

generate:
	go generate

check-diff:
	@./scripts/check-diff.sh

$(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH):
	@mkdir -p $@

build: schemalint schemalex schemadiff

schemalex: $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemalex$(SUFFIX)

schemalint: $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemalint$(SUFFIX)

schemadiff: $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemadiff$(SUFFIX)

$(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemalint$(SUFFIX): $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH) $(SRC_FILES)
	echo " * Building schemalint for $(GOOS)/$(GOARCH)..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemalint$(SUFFIX) cmd/schemalint/schemalint.go

$(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemalex$(SUFFIX): $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH) $(SRC_FILES)
	@echo " * Building schemalex for $(GOOS)/$(GOARCH)..."
	@go build -o $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemalex$(SUFFIX) cmd/schemalex/schemalex.go

$(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemadiff$(SUFFIX): $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH) $(SRC_FILES)
	echo " * Building schemadiff for $(GOOS)/$(GOARCH)..."
	go build -ldflags "-X main.version=$(VERSION)" -o $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/schemadiff$(SUFFIX) cmd/schemadiff/schemadiff.go

all: build-linux-amd64 build-linux-386 build-darwin-amd64 build-darwin-386 build-windows-amd64 build-windows-386

build-windows-amd64:
	@$(MAKE) build GOOS=windows GOARCH=amd64 SUFFIX=.exe

build-windows-386:
	@$(MAKE) build GOOS=windows GOARCH=386 SUFFIX=.exe

build-linux-amd64:
	@$(MAKE) build GOOS=linux GOARCH=amd64

build-linux-386:
	@$(MAKE) build GOOS=linux GOARCH=386

build-darwin-amd64:
	@$(MAKE) build GOOS=darwin GOARCH=amd64

build-darwin-386:
	@$(MAKE) build GOOS=darwin GOARCH=386

$(RELEASE_DIR)/schemalex_$(GOOS)_$(GOARCH):
	@mkdir -p $@

$(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/Changes: $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH) Changes
	@echo " * Copying Changes..."
	@cp Changes $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)

$(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/README.md: $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH) README.md
	@echo " * Copying README.md..."
	@cp README.md $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)

release-changes: $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/Changes
release-readme: $(ARTIFACTS_DIR)/schemalex_$(GOOS)_$(GOARCH)/README.md

release-windows-amd64:
	@$(MAKE) build release-changes release-readme release-zip GOOS=windows GOARCH=amd64

release-windows-386:
	@$(MAKE) build release-changes release-readme release-zip GOOS=windows GOARCH=386

release-linux-amd64:
	@$(MAKE) build release-changes release-readme release-targz GOOS=linux GOARCH=amd64

release-linux-386:
	@$(MAKE) build release-changes release-readme release-targz GOOS=linux GOARCH=386

release-darwin-amd64:
	@$(MAKE) build release-changes release-readme release-zip GOOS=darwin GOARCH=amd64

release-darwin-386:
	@$(MAKE) build release-changes release-readme release-zip GOOS=darwin GOARCH=386

release-tarbz: $(RELEASE_DIR)/schemalex_$(GOOS)_$(GOARCH)
	@echo " * Creating tar.bz for $(GOOS)/$(GOARCH)"
	@tar -cjf $(RELEASE_DIR)/schemalex_$(GOOS)_$(GOARCH).tar.bz2 -C $(ARTIFACTS_DIR) schemalex_$(GOOS)_$(GOARCH)

release-targz: $(RELEASE_DIR)/schemalex_$(GOOS)_$(GOARCH)
	@echo " * Creating tar.gz for $(GOOS)/$(GOARCH)"
	tar -czf $(RELEASE_DIR)/schemalex_$(GOOS)_$(GOARCH).tar.gz -C $(ARTIFACTS_DIR) schemalex_$(GOOS)_$(GOARCH)

release-zip: $(RELEASE_DIR)/schemalex_$(GOOS)_$(GOARCH)
	@echo " * Creating zip for $(GOOS)/$(GOARCH)"
	cd $(ARTIFACTS_DIR) && zip -9 $(RELEASE_DIR)/schemalex_$(GOOS)_$(GOARCH).zip schemalex_$(GOOS)_$(GOARCH)/*

release-files: release-windows-386 release-windows-amd64 release-linux-386 release-linux-amd64 release-darwin-386 release-darwin-amd64

release-github-token: github_token
	@echo "file `github_token` is required"

release-upload: release-files release-github-token
	ghr -u $(GITHUB_USERNAME) -t $(shell cat github_token) --draft --replace $(VERSION) $(RELEASE_DIR)
