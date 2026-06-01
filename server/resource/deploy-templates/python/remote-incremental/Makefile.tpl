IMAGE_NAME ?= {{ .ImageName }}
PROJECT_IMAGE_NAME ?= {{ .ImageName }}
REMOTE_IMAGE_TAR ?= $(IMAGE_NAME).tar

.PHONY: ensure-project build-remote-incremental package-remote-incremental

ensure-project:
	@docker image inspect $(PROJECT_IMAGE_NAME) >/dev/null 2>&1 || docker build -f Dockerfile.project -t $(PROJECT_IMAGE_NAME) .

build-remote-incremental: ensure-project
	docker build -f Dockerfile.remote --build-arg PROJECT_IMAGE=$(PROJECT_IMAGE_NAME) -t $(IMAGE_NAME) .

package-remote-incremental: build-remote-incremental
	docker save -o $(REMOTE_IMAGE_TAR) $(IMAGE_NAME)
