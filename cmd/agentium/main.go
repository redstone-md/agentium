package main

import (
	"context"
	"flag"
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
	mode := flag.String("mode", "http", "http or mcp-stdio")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := config.Load()
	service := app.NewService(cfg)
	defer func() {
		if err := service.Close(); err != nil {
			log.Printf("shutdown cleanup failed: %v", err)
		}
	}()

	switch *mode {
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
