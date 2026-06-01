ARG PROJECT_IMAGE={{ .ImageName }}
FROM ${PROJECT_IMAGE}

ENV PYTHONUNBUFFERED=1

WORKDIR /app

RUN if command -v apk >/dev/null 2>&1; then apk add --no-cache build-base linux-headers; fi

COPY requirements.txt ./

RUN python -m pip install --no-cache-dir -r requirements.txt -i https://pypi.tuna.tsinghua.edu.cn/simple
