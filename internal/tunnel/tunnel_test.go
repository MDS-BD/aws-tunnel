package tunnel

import (
	"net"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestStartBusyPort(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Listen failed: %v", err)
	}
	defer l.Close()
	port := l.Addr().(*net.TCPAddr).Port

	// No credentials in cfg: reaching AWS would fail differently, so this proves the port is checked first.
	_, err = Start(t.Context(), aws.Config{}, Options{Host: "example.com", LocalPort: port, RemotePort: 443, Target: "i-1"})
	if err == nil || !strings.Contains(err.Error(), "local port") {
		t.Errorf("expected a local port error, got %v", err)
	}
}
