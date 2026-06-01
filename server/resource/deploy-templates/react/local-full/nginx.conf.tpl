server {
    listen 80;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

{{- range .ObjectStorageProxyPrefixes }}
    location ^~ /api/{{ .Prefix }}/ {
        rewrite ^/api/(.*)$ /$1 break;
        proxy_pass {{ .Endpoint }};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    location ^~ /{{ .Prefix }}/ {
        proxy_pass {{ .Endpoint }};
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

{{- end }}
    location /api/ {
{{- if .APIProxyStripPrefix }}
        proxy_pass http://host.docker.internal:{{ .BackendDeployPort }}/;
{{- else }}
        proxy_pass http://host.docker.internal:{{ .BackendDeployPort }};
{{- end }}
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

{{- if .HasWebSocket }}
    location /ws {
        proxy_pass http://host.docker.internal:{{ .WebSocketDeployPort }};
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
{{- end }}

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?)$ {
        expires 7d;
        add_header Cache-Control "public, immutable";
    }
}
