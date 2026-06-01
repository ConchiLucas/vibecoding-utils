FROM golang:{{ .GoVersion }}-alpine AS builder

WORKDIR /build
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server .

FROM alpine:latest

WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
ENV TZ=Asia/Shanghai

COPY --from=builder /build/server .
{{ if .GoConfigCopyCommand }}{{ .GoConfigCopyCommand }}{{ else }}COPY config.yaml config.yaml{{ end }}

EXPOSE {{ .AppPort }}

CMD ["./server"]
