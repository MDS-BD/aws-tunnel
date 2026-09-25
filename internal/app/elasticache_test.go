package app

import (
	"testing"

	"github.com/MDS-BD/aws-tunnel/internal/cloud"
)

func TestCacheConnectHint(t *testing.T) {
	host := "master.my-group.abc123.euw1.cache.amazonaws.com"
	cases := []struct {
		cache    cloud.Cache
		user     string
		iam      bool
		expected string
	}{
		{cloud.Cache{Host: host}, "default", false, "redis-cli -h 127.0.0.1 -p 6380"},
		{cloud.Cache{Host: host, TLS: true}, "default", false, "redis-cli -h 127.0.0.1 -p 6380 --tls --insecure --sni " + host},
		{cloud.Cache{Host: host, TLS: true}, "app", true, "redis-cli -h 127.0.0.1 -p 6380 --tls --insecure --sni " + host + " --user app --askpass"},
		{cloud.Cache{Host: host}, "app", false, "redis-cli -h 127.0.0.1 -p 6380 --user app --askpass"},
	}
	for _, c := range cases {
		if got := cacheConnectHint(c.cache, "6380", c.user, c.iam); got != c.expected {
			t.Errorf("cacheConnectHint mismatch:\nGot: %s\nExpected: %s", got, c.expected)
		}
	}
}
