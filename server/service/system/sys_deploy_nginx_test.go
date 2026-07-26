package system

import "testing"

func TestRewriteAPIProxyPassToStripPrefixDoesNotRewriteNonAPILocation(t *testing.T) {
	content := `server {
    location /api/ {
        proxy_pass http://host.docker.internal:6015/;
    }

    location ^~ /ai-file-navigation/ {
        proxy_pass http://host.docker.internal:6015;
    }
}
`

	got := rewriteAPIProxyPassToStripPrefix(content, 6015)
	if got != content {
		t.Fatalf("rewrite changed a non-API proxy:\n%s", got)
	}
}
