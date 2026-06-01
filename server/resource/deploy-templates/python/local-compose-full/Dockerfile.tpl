ARG PROJECT_IMAGE={{ .ImageName }}
FROM ${PROJECT_IMAGE}

WORKDIR /app

ENV PYTHONUNBUFFERED=1

COPY . .

EXPOSE {{ .AppPort }}

{{ .PythonStartCommand }}
