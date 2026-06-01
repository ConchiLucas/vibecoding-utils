services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
      args:
        PROJECT_IMAGE: {{ .ImageName }}
    image: {{ .ImageName }}
    container_name: {{ .ContainerName }}
    restart: unless-stopped
    labels:
      easy-deploy.project: "{{ .ProjectName }}"
      easy-deploy.container: "{{ .ContainerName }}"
    ports:
      - "{{ .AppPort }}:{{ .AppPort }}"
    environment:
      APP_ENV: prod
      APP_HOST: 0.0.0.0
      APP_PORT: {{ .AppPort }}
      BUSINESS_CLICKHOUSE_HOST: host.docker.internal
      CH_HOST: host.docker.internal
      REDIS_HOST: host.docker.internal
    extra_hosts:
      - "host.docker.internal:host-gateway"
    volumes:
      - ./logs:/app/logs
