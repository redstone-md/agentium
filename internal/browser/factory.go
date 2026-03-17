package browser

import (
	"fmt"
	"strings"
	"sync"
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
	config   config.Config
	mu       sync.Mutex
	browsers map[string]*sharedBrowser
}

type sharedBrowser struct {
	rootBrowser *rod.Browser
	refCount    int
}

func NewFactory(cfg config.Config) *Factory {
	return &Factory{
		config:   cfg,
		browsers: make(map[string]*sharedBrowser),
	}
}

func (f *Factory) Create(options model.SessionOptions) (*session.Runtime, error) {
	proxyKey := normalizeProxyKey(options.Proxy)
	rootBrowser, err := f.acquireRootBrowser(proxyKey, options.Proxy)
	if err != nil {
		return nil, err
	}

	contextBrowser, err := rootBrowser.Incognito()
	if err != nil {
		f.releaseRootBrowser(proxyKey)
		return nil, fmt.Errorf("create incognito context: %w", err)
	}

	page, err := stealth.Page(contextBrowser)
	if err != nil {
		_ = contextBrowser.Close()
		f.releaseRootBrowser(proxyKey)
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
		f.releaseRootBrowser(proxyKey)
		return nil, fmt.Errorf("set viewport: %w", err)
	}

	if err := applyPageEmulation(page, options); err != nil {
		_ = page.Close()
		_ = contextBrowser.Close()
		f.releaseRootBrowser(proxyKey)
		return nil, err
	}

	restoreNetwork := page.EnableDomain(&proto.NetworkEnable{})

	tracker := telemetry.NewTracker(100)
	bindNetworkEvents(page, tracker)

	id := uuid.NewString()
	closeFn := func() error {
		restoreNetwork()
		closeErr := closeSessionResources(page, contextBrowser)
		f.releaseRootBrowser(proxyKey)
		return closeErr
	}

	return session.NewRuntime(id, options, rootBrowser, contextBrowser, page, tracker, closeFn), nil
}

func (f *Factory) acquireRootBrowser(proxyKey, proxyURL string) (*rod.Browser, error) {
	f.mu.Lock()
	if browser := f.browsers[proxyKey]; browser != nil {
		browser.refCount++
		root := browser.rootBrowser
		f.mu.Unlock()
		return root, nil
	}
	f.mu.Unlock()

	rootBrowser, err := f.launchRootBrowser(proxyURL)
	if err != nil {
		return nil, err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if existing := f.browsers[proxyKey]; existing != nil {
		existing.refCount++
		_ = rootBrowser.Close()
		return existing.rootBrowser, nil
	}

	f.browsers[proxyKey] = &sharedBrowser{
		rootBrowser: rootBrowser,
		refCount:    1,
	}
	return rootBrowser, nil
}

func (f *Factory) releaseRootBrowser(proxyKey string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	browser := f.browsers[proxyKey]
	if browser == nil {
		return
	}

	browser.refCount--
	if browser.refCount <= 0 {
		_ = browser.rootBrowser.Close()
		delete(f.browsers, proxyKey)
	}
}

func (f *Factory) launchRootBrowser(proxyURL string) (*rod.Browser, error) {
	launch := launcher.New().
		Leakless(true).
		Headless(false).
		NoSandbox(true).
		Set("disable-blink-features", "AutomationControlled")

	if f.config.ChromeBin != "" {
		launch = launch.Bin(f.config.ChromeBin)
	}

	if proxyURL != "" {
		launch = launch.Proxy(proxyURL)
	}

	controlURL, err := launch.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch chromium: %w", err)
	}

	rootBrowser := rod.New().ControlURL(controlURL).Timeout(90 * time.Second)
	if err := rootBrowser.Connect(); err != nil {
		return nil, fmt.Errorf("connect browser: %w", err)
	}

	return rootBrowser, nil
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

func normalizeProxyKey(proxyURL string) string {
	if proxyURL == "" {
		return "direct"
	}
	return proxyURL
}

func closeSessionResources(page *rod.Page, contextBrowser *rod.Browser) error {
	if err := page.Close(); err != nil {
		_ = contextBrowser.Close()
		return err
	}

	if err := contextBrowser.Close(); err != nil {
		return err
	}

	return nil
}

func bindNetworkEvents(page *rod.Page, tracker *telemetry.Tracker) {
	go page.EachEvent(
		func(event *proto.NetworkRequestWillBeSent) {
			tracker.OnRequest(model.NetworkEvent{
				Timestamp:    time.Now(),
				RequestID:    string(event.RequestID),
				Method:       event.Request.Method,
				URL:          event.Request.URL,
				ResourceType: string(event.Type),
				Stage:        "request",
			})
		},
		func(event *proto.NetworkResponseReceived) {
			tracker.OnResponse(string(event.RequestID), int(event.Response.Status), time.Now())
		},
		func(event *proto.NetworkLoadingFinished) {
			tracker.OnFinished(string(event.RequestID), time.Now())
		},
		func(event *proto.NetworkLoadingFailed) {
			tracker.OnFailed(string(event.RequestID), event.ErrorText, time.Now())
		},
	)()
}
