package config

import "testing"

func TestLoadReadsHeadlessEnv(t *testing.T) {
	t.Setenv("AGENTIUM_HEADLESS", "true")

	cfg := Load()

	if !cfg.Headless {
		t.Fatal("expected headless mode to be enabled from env")
	}
}

func TestParseArgsOverridesEnvBackedConfig(t *testing.T) {
	base := Config{
		HTTPAddr:      ":8080",
		ChromeBin:     "/usr/bin/chromium",
		Headless:      false,
		DefaultWidth:  1280,
		DefaultHeight: 800,
		UseLeakless:   true,
	}

	cfg, cli, err := ParseArgs([]string{
		"-mode", "mcp-stdio",
		"-http-addr", ":9090",
		"-chrome-bin", "/custom/chrome",
		"-headless=true",
		"-leakless=false",
		"-viewport-width", "1440",
		"-viewport-height", "900",
	}, base)
	if err != nil {
		t.Fatalf("ParseArgs returned error: %v", err)
	}

	if cli.Mode != "mcp-stdio" {
		t.Fatalf("expected mcp-stdio mode, got %q", cli.Mode)
	}
	if cfg.HTTPAddr != ":9090" || cfg.ChromeBin != "/custom/chrome" {
		t.Fatal("expected CLI flags to override http address and chrome path")
	}
	if !cfg.Headless || cfg.UseLeakless {
		t.Fatal("expected CLI flags to override headless and leakless")
	}
	if cfg.DefaultWidth != 1440 || cfg.DefaultHeight != 900 {
		t.Fatal("expected viewport overrides to be applied")
	}
}
