package cloud

import (
	"net/url"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
)

func TestCacheIAMToken(t *testing.T) {
	cfg := aws.Config{
		Credentials: credentials.NewStaticCredentialsProvider("AKIDEXAMPLE", "secret", "session"),
		Region:      "eu-west-1",
	}

	cases := []struct {
		cache        Cache
		resourceType string
	}{
		{Cache{Name: "my-group"}, ""},
		{Cache{Name: "my-serverless", Serverless: true}, "ServerlessCache"},
	}
	for _, c := range cases {
		token, err := CacheIAMToken(t.Context(), cfg, c.cache, "app")
		if err != nil {
			t.Fatalf("CacheIAMToken failed: %v", err)
		}
		if !strings.HasPrefix(token, c.cache.Name+"/?") {
			t.Errorf("token must start with the cache name and no scheme: %s", token)
		}

		q, err := url.ParseQuery(token[strings.Index(token, "?")+1:])
		if err != nil {
			t.Fatalf("token query does not parse: %v", err)
		}
		expected := map[string]string{
			"Action":               "connect",
			"ResourceType":         c.resourceType,
			"User":                 "app",
			"X-Amz-Algorithm":      "AWS4-HMAC-SHA256",
			"X-Amz-Expires":        "900",
			"X-Amz-Security-Token": "session",
		}
		for k, v := range expected {
			if got := q.Get(k); got != v {
				t.Errorf("%s: %s = %q, expected %q", c.cache.Name, k, got, v)
			}
		}
		if !strings.Contains(q.Get("X-Amz-Credential"), "/eu-west-1/elasticache/aws4_request") {
			t.Errorf("not scoped to elasticache: %s", q.Get("X-Amz-Credential"))
		}
		if q.Get("X-Amz-Signature") == "" {
			t.Errorf("signature missing")
		}
	}
}
