FROM python:{{ .PythonVersion }}-alpine

ENV PYTHONUNBUFFERED=1

WORKDIR /app

RUN if command -v apk >/dev/null 2>&1; then apk add --no-cache build-base linux-headers; fi

{{ .PythonDependencyCopyCommand }}

{{ .PythonDependencyInstallCommand }}
