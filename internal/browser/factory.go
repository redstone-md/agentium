package browser

import (
	"fmt"
	"strings"
	"time"

	"agentium/internal/config"
	"agentium/internal/model"
	"agentium/internal/session"
	"agentium/internal/telemetry"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/go-rod/stealth"
	"github.com/google/uuid"
)

type Factory struct {
	config config.Config
}

func NewFactory(cfg config.Config) *Factory {
	return &Factory{config: cfg}
}

func (f *Factory) Create(options model.SessionOptions) (*session.Runtime, error) {
	launch := launcher.New().
		Leakless(true).
		Headless(false).
		NoSandbox(true).
		Set("disable-blink-features", "AutomationControlled")

	if f.config.ChromeBin != "" {
		launch = launch.Bin(f.config.ChromeBin)
	}

	if options.Proxy != "" {
		launch = launch.Proxy(options.Proxy)
	}

	controlURL, err := launch.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch chromium: %w", err)
	}

	rootBrowser := rod.New().ControlURL(controlURL).Timeout(90 * time.Second)
	if err := rootBrowser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	contextBrowser, err := rootBrowser.Incognito()
	if err != nil {
		_ = rootBrowser.Close()
		return nil, fmt.Errorf("create incognito context: %w", err)
	}

	page, err := stealth.Page(contextBrowser)
	if err != nil {
		_ = contextBrowser.Close()
		_ = rootBrowser.Close()
		return nil, fmt.Errorf("create page: %w", err)
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             f.config.DefaultWidth,
		Height:            f.config.DefaultHeight,
		DeviceScaleFactor: 1,
		Mobile:            false,
	}); err != nil {
		_ = page.Close()
		_ = contextBrowser.Close()
		_ = rootBrowser.Close()
		return nil, fmt.Errorf("set viewport: %w", err)
	}

	if err := applyPageEmulation(page, options); err != nil {
		_ = page.Close()
		_ = contextBrowser.Close()
		_ = rootBrowser.Close()
		return nil, err
	}

	restoreNetwork := page.EnableDomain(&proto.NetworkEnable{})

	tracker := telemetry.NewTracker(100)
	bindNetworkEvents(page, tracker)

	id := uuid.NewString()
	closeFn := func() error {
		restoreNetwork()
		if err := page.Close(); err != nil {
			_ = contextBrowser.Close()
			_ = rootBrowser.Close()
			return err
		}
		if err := contextBrowser.Close(); err != nil {
			_ = rootBrowser.Close()
			return err
		}
		return rootBrowser.Close()
	}

	return session.NewRuntime(id, options, rootBrowser, contextBrowser, page, tracker, closeFn), nil
}

func applyPageEmulation(page *rod.Page, options model.SessionOptions) error {
	if options.UserAgent != "" {
		if err := (proto.NetworkSetUserAgentOverride{
			UserAgent:      options.UserAgent,
			AcceptLanguage: options.Locale,
			Platform:       inferOverrideProfile(options.UserAgent, options.Locale).Platform,
		}).Call(page); err != nil {
			return fmt.Errorf("set user agent: %w", err)
		}
	}

	if options.TimezoneID != "" {
		if err := (proto.EmulationSetTimezoneOverride{TimezoneID: options.TimezoneID}).Call(page); err != nil {
			return fmt.Errorf("set timezone: %w", err)
		}
	}

	if options.Locale != "" {
		if err := (proto.EmulationSetLocaleOverride{Locale: normalizeLocale(options.Locale)}).Call(page); err != nil {
			return fmt.Errorf("set locale: %w", err)
		}
	}

	if err := ApplyStealth(page, options.UserAgent, options.Locale); err != nil {
		return err
	}

	return nil
}

func normalizeLocale(value string) string {
	return strings.ReplaceAll(value, "-", "_")
}

func bindNetworkEvents(page *rod.Page, tracker *telemetry.Tracker) {
	go page.EachEvent(
		func(event *proto.NetworkRequestWillBeSent) {
			tracker.Push(model.NetworkEvent{
				Timestamp:    time.Now(),
				RequestID:    string(event.RequestID),
				Method:       event.Request.Method,
				URL:          event.Request.URL,
				ResourceType: string(event.Type),
				Stage:        "request",
			})
		},
		func(event *proto.NetworkResponseReceived) {
			tracker.Push(model.NetworkEvent{
				Timestamp:    time.Now(),
				RequestID:    string(event.RequestID),
				URL:          event.Response.URL,
				Status:       int(event.Response.Status),
				ResourceType: string(event.Type),
				Stage:        "response",
			})
		},
		func(event *proto.NetworkLoadingFinished) {
			tracker.Push(model.NetworkEvent{
				Timestamp: time.Now(),
				RequestID: string(event.RequestID),
				Stage:     "finished",
			})
		},
		func(event *proto.NetworkLoadingFailed) {
			tracker.Push(model.NetworkEvent{
				Timestamp: time.Now(),
				RequestID: string(event.RequestID),
				Stage:     "failed",
				ErrorText: event.ErrorText,
			})
		},
	)()
}
