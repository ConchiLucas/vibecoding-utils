PROJECT_IMAGE_NAME ?= {{ .ImageName }}

.PHONY: build-project

build-project:
	docker build -f Dockerfile.project -t $(PROJECT_IMAGE_NAME) .
