FROM node:{{ .NodeVersion }}-alpine AS builder

WORKDIR /app

COPY package.json ./
{{ .CopyLockFileCommand }}
RUN {{ .InstallCommand }}

COPY . .
RUN {{ .BuildCommand }}

FROM nginx:1.27-alpine AS runner

COPY --from=builder /app/{{ .DistDir }} /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
