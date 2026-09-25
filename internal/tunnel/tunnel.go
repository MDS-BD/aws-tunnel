package tunnel

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/alexbacchin/ssm-session-client/ssmclient"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type Options struct {
	Host       string
	LocalPort  int
	RemotePort int
	Target     string
}

// Start opens an SSM port-forward session in the background: AWS-StartPortForwardingSessionToRemoteHost when
// Host is set, AWS-StartPortForwardingSession (a port on the target itself) otherwise.
// It returns once the local listener accepts connections; the returned channel reports when the session ends.
func Start(ctx context.Context, cfg aws.Config, opts Options) (<-chan error, error) {
	// The client opens the SSM session before listening: check the port first so a busy one fails fast.
	l, err := net.Listen("tcp", net.JoinHostPort("localhost", strconv.Itoa(opts.LocalPort)))
	if err != nil {
		return nil, fmt.Errorf("local port %d: %w", opts.LocalPort, err)
	}
	_ = l.Close()

	ready := make(chan struct{})
	done := make(chan error, 1)

	go func() {
		done <- ssmclient.PortForwardingSessionWithContext(ctx, cfg, &ssmclient.PortForwardingInput{
			EnableReconnect: true,
			Host:            opts.Host,
			LocalPort:       opts.LocalPort,
			ReadyCh:         ready,
			RemotePort:      opts.RemotePort,
			Target:          opts.Target,
		})
	}()

	select {
	case <-ready:
		return done, nil
	case err := <-done:
		return nil, fmt.Errorf("ssm port forwarding: %w", err)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
