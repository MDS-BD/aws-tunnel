package app

import "testing"

func TestListenFromSkipsBusyPort(t *testing.T) {
	busy, err := ListenFrom(0)
	if err != nil {
		t.Fatal(err)
	}
	defer busy.Close()

	base := listenPort(busy)
	l, err := ListenFrom(base)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	if got := listenPort(l); got <= base || got >= base+portScanRange {
		t.Errorf("ListenFrom(%d) = %d, want a port in (%d, %d)", base, got, base, base+portScanRange)
	}
}
