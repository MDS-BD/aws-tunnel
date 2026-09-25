package tunnel

import (
	"context"
	"fmt"

	"github.com/alexbacchin/ssm-session-client/ssmclient"
	"github.com/aws/aws-sdk-go-v2/aws"
)

type Options struct {
	Host       string
	LocalPort  int
	RemotePort int
	Target     string
}

// Start opens an SSM port-forward session (AWS-StartPortForwardingSessionToRemoteHost) in the background.
// It returns once the local listener accepts connections; the returned channel reports when the session ends.
func Start(ctx context.Context, cfg aws.Config, opts Options) (<-chan error, error) {
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
