package app

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/alexbacchin/ssm-session-client/config"
	"github.com/alexbacchin/ssm-session-client/ssmclient"
	"github.com/rs/zerolog"
	"golang.org/x/term"
)

type ShellCmd struct {
	Document string `env:"AWS_TUNNEL_SHELL_DOCUMENT" help:"SSM session document, e.g. AWS-StartInteractiveCommand; defaults to the account's shell document."`
}

// Validate runs at parse time, before any AWS call.
func (c *ShellCmd) Validate() error {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return errors.New("shell needs an interactive terminal")
	}
	return nil
}

func (c *ShellCmd) Run(ctx context.Context, s *Session) error {
	target, err := s.PickTarget(ctx)
	if err != nil {
		return err
	}
	s.SaveCache(ctx)

	// The SSM client reads the session document from its global flags.
	config.Flags().Shell.DocumentName = c.Document

	zerolog.Ctx(ctx).Info().Str("target", target).Msg("starting shell session — exit the remote shell to leave")
	// The session owns the terminal: it switches it to raw mode, so Ctrl+C reaches the remote shell.
	if err := ssmclient.ShellSession(s.Cfg, target); err != nil {
		return fmt.Errorf("ssm shell session: %w", err)
	}
	return nil
}
