# Docker Hub username - change this to your username
DOCKER_USERNAME ?= nsavage
IMAGE_NAME = file-browser
TAG ?= latest

# Build and tag the image
build:
	docker build --platform=linux/amd64 -t $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG) .

# Push to Docker Hub
push: build
	docker push $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG)

# Build and push with version tag
release: build
	docker tag $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG) $(DOCKER_USERNAME)/$(IMAGE_NAME):v1.0.0
	docker push $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG)
	docker push $(DOCKER_USERNAME)/$(IMAGE_NAME):v1.0.0

# Run locally for testing
test: build
	docker run --rm -p 8080:8080 -v $(PWD)/test-data:/data $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG)

# Clean up
clean:
	docker rmi $(DOCKER_USERNAME)/$(IMAGE_NAME):$(TAG) || true
	docker rmi $(DOCKER_USERNAME)/$(IMAGE_NAME):v1.0.0 || true

.PHONY: build push release test clean