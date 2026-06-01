ARG PROJECT_IMAGE={{ .ImageName }}
FROM ${PROJECT_IMAGE}

ENV PYTHONUNBUFFERED=1

WORKDIR /app

COPY . .

EXPOSE {{ .AppPort }}

{{ .PythonStartCommand }}
