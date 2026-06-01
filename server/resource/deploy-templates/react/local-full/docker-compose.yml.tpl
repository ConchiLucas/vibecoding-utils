services:
  {{ .ContainerName }}:
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
      - "{{ .FrontendDeployPort }}:80"
