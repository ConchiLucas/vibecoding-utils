services:
  app:
    build:
      context: .
      dockerfile: Dockerfile
    image: {{ .ImageName }}
    container_name: {{ .ContainerName }}
    restart: unless-stopped
    labels:
      easy-deploy.project: "{{ .ProjectName }}"
      easy-deploy.container: "{{ .ContainerName }}"
    ports:
      - "{{ .BackendDeployPort }}:{{ .AppPort }}"
{{- if .HasWebSocket }}
      - "{{ .WebSocketDeployPort }}:{{ .WebSocketPort }}"
{{- end }}
    environment:
      SPRING_DATASOURCE_URL: jdbc:postgresql://host.docker.internal:5432/{{ .DatabaseName }}
      SPRING_DATASOURCE_USERNAME: {{ .DatabaseUsername }}
      SPRING_DATASOURCE_PASSWORD: {{ .DatabasePassword }}
      SPRING_DATA_REDIS_HOST: {{ .RedisHost }}
      SPRING_DATA_REDIS_PORT: {{ .RedisPort }}
      SPRING_DATA_REDIS_PASSWORD: {{ .RedisPassword }}
{{- if .HasWebSocket }}
      NETTY_WEBSOCKET_PORT: {{ .WebSocketPort }}
{{- end }}
      JAVA_OPTS: "-Xms256m -Xmx512m"
    volumes:
      - ./logs:/app/logs
