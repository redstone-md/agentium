package config

import (
	"flag"
	"fmt"
	"io"
)

type CLIOptions struct {
	Mode        string
	PrintConfig bool
}

func ParseArgs(args []string, base Config) (Config, CLIOptions, error) {
	fs := flag.NewFlagSet("agentium", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	mode := fs.String("mode", "http", "runtime mode: http or mcp-stdio")
	httpAddr := fs.String("http-addr", base.HTTPAddr, "HTTP bind address")
	chromeBin := fs.String("chrome-bin", base.ChromeBin, "absolute path to Chrome/Chromium")
	headless := fs.Bool("headless", base.Headless, "run Chromium in headless mode")
	leakless := fs.Bool("leakless", base.UseLeakless, "use Rod leakless helper process management")
	viewportWidth := fs.Int("viewport-width", base.DefaultWidth, "default viewport width")
	viewportHeight := fs.Int("viewport-height", base.DefaultHeight, "default viewport height")
	printConfig := fs.Bool("print-config", false, "print effective startup config and exit")

	if err := fs.Parse(args); err != nil {
		return Config{}, CLIOptions{}, err
	}

	if *mode != "http" && *mode != "mcp-stdio" {
		return Config{}, CLIOptions{}, fmt.Errorf("invalid mode %q: expected http or mcp-stdio", *mode)
	}

	cfg := base
	cfg.HTTPAddr = *httpAddr
	cfg.ChromeBin = *chromeBin
	cfg.Headless = *headless
	cfg.UseLeakless = *leakless
	cfg.DefaultWidth = *viewportWidth
	cfg.DefaultHeight = *viewportHeight

	return cfg, CLIOptions{
		Mode:        *mode,
		PrintConfig: *printConfig,
	}, nil
}
