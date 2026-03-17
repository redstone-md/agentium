package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agentium/internal/api"
	"agentium/internal/app"
	"agentium/internal/config"
	"agentium/internal/mcpserver"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, cli, err := config.ParseArgs(os.Args[1:], config.Load())
	if err != nil {
		log.Fatal(err)
	}
	if cli.PrintConfig {
		log.Printf("mode=%s http_addr=%s headless=%t chrome_bin=%q leakless=%t viewport=%dx%d",
			cli.Mode,
			cfg.HTTPAddr,
			cfg.Headless,
			cfg.ChromeBin,
			cfg.UseLeakless,
			cfg.DefaultWidth,
			cfg.DefaultHeight,
		)
		return
	}

	log.Printf("starting agentium mode=%s http_addr=%s headless=%t leakless=%t viewport=%dx%d",
		cli.Mode,
		cfg.HTTPAddr,
		cfg.Headless,
		cfg.UseLeakless,
		cfg.DefaultWidth,
		cfg.DefaultHeight,
	)

	service := app.NewService(cfg)
	defer func() {
		if err := service.Close(); err != nil {
			log.Printf("shutdown cleanup failed: %v", err)
		}
	}()

	switch cli.Mode {
	case "mcp-stdio":
		if err := mcpserver.New(service).RunStdio(ctx); err != nil {
			log.Fatal(err)
		}
	default:
		server := echo.New()
		server.HideBanner = true
		server.Use(middleware.Recover())
		server.Use(middleware.RequestLogger())

		api.NewHandler(service).Register(server)
		registerMCPSSE(server, service)

		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				log.Printf("http shutdown failed: %v", err)
			}
		}()

		if err := server.Start(cfg.HTTPAddr); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}
}

func registerMCPSSE(server *echo.Echo, service *app.Service) {
	handler := mcp.NewSSEHandler(func(_ *http.Request) *mcp.Server {
		return mcpserver.New(service).Build()
	}, nil)

	server.Any("/mcp", echo.WrapHandler(handler))
	server.Any("/mcp/*", echo.WrapHandler(handler))
}
