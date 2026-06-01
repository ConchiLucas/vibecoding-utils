PROJECT_IMAGE_NAME ?= {{ .ImageName }}

.PHONY: ensure-project build-deps

ensure-project:
	@docker image inspect $(PROJECT_IMAGE_NAME) >/dev/null 2>&1 || (echo "项目镜像 $(PROJECT_IMAGE_NAME) 不存在，请先执行【构建项目镜像】" && exit 1)

build-deps: ensure-project
	docker build -f Dockerfile.deps --build-arg PROJECT_IMAGE=$(PROJECT_IMAGE_NAME) -t $(PROJECT_IMAGE_NAME) .
