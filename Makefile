GOVERSION=$(shell go version)
GOOS=$(word 1,$(subst /, ,$(word $(words $(GOVERSION)), $(GOVERSION))))
GOARCH=$(word 2,$(subst /, ,$(word $(words $(GOVERSION)), $(GOVERSION))))

installdeps: glide-$(GOOS)-$(GOARCH)/glide
	PATH=glide-$(GOOS)-$(GOARCH):$(PATH) glide install

glide-$(GOOS)-$(GOARCH):
	@echo "Creating $(@F)"
	@mkdir -p $(@F)
	
glide-$(GOOS)-$(GOARCH)/glide:
	@$(MAKE) glide-$(GOOS)-$(GOARCH)
	@wget -O - https://github.com/Masterminds/glide/releases/download/v0.12.3/glide-v0.12.3-$(GOOS)-$(GOARCH).tar.gz | tar xvz
	@mv $(GOOS)-$(GOARCH)/glide glide-$(GOOS)-$(GOARCH)
	@rm -rf $(GOOS)-$(GOARCH)

test:
	go test -v $(shell glide-$(GOOS)-$(GOARCH)/glide novendor)

generate:
	go generate
