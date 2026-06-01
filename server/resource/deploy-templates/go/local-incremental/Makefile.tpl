IMAGE_NAME ?= {{ .ImageName }}

.PHONY: deploy-incremental deploy-full stop

deploy-incremental:
	docker compose up --build -d

deploy-full:
	docker compose build --no-cache
	docker compose up -d

stop:
	docker compose down
