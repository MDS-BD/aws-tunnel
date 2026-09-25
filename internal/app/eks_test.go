package app

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

func TestTokenCache(t *testing.T) {
	calls := 0
	expiration := time.Now().Add(10 * time.Minute)
	tokens := &tokenCache{fetch: func(context.Context) (token.Token, error) {
		calls++
		return token.Token{Token: "k8s-aws-v1.abc", Expiration: expiration}, nil
	}}

	for range 3 {
		if got, err := tokens.Get(t.Context()); err != nil || got != "k8s-aws-v1.abc" {
			t.Fatalf("Get = %q, %v", got, err)
		}
	}
	if calls != 1 {
		t.Errorf("valid token must be reused, fetched %d times", calls)
	}

	expiration = time.Now().Add(30 * time.Second)
	tokens.token.Expiration = expiration
	if _, err := tokens.Get(t.Context()); err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if calls != 2 {
		t.Errorf("token about to expire must be renewed, fetched %d times", calls)
	}
}

func TestWriteKubeconfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	path, err := writeKubeconfig("prod", 8443)
	if err != nil {
		t.Fatalf("writeKubeconfig failed: %v", err)
	}
	cfg, err := clientcmd.LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}
	if cfg.CurrentContext != "prod" {
		t.Errorf("current context mismatch: %s", cfg.CurrentContext)
	}
	if got := cfg.Clusters["prod"].Server; got != "http://localhost:8443" {
		t.Errorf("server mismatch: %s", got)
	}
}
