package system

import (
	"net"
	"testing"
)

func TestProjectAccessURLRunning(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()

	if !projectAccessURLRunning("http://" + listener.Addr().String() + "/health") {
		t.Fatal("expected listening access URL to be running")
	}
}

func TestProjectAccessURLRunningRejectsUnavailableOrInvalidURL(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close listener: %v", err)
	}

	for _, accessURL := range []string{"", "://bad", "http://" + address} {
		if projectAccessURLRunning(accessURL) {
			t.Fatalf("expected %q to be stopped", accessURL)
		}
	}
}
