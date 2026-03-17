package browser

import "testing"

func TestNormalizeProxyKey(t *testing.T) {
	if got := normalizeProxyKey(""); got != "direct" {
		t.Fatalf("expected direct key, got %q", got)
	}

	proxy := "http://user:pass@127.0.0.1:8080"
	if got := normalizeProxyKey(proxy); got != proxy {
		t.Fatalf("expected proxy key to be stable, got %q", got)
	}
}
