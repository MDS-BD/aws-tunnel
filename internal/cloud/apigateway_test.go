package cloud

import "testing"

func TestAPIHost(t *testing.T) {
	got := API{ID: "abc123"}.Host("eu-west-1")
	if expected := "abc123.execute-api.eu-west-1.amazonaws.com"; got != expected {
		t.Errorf("host mismatch:\nGot: %s\nExpected: %s", got, expected)
	}
}
