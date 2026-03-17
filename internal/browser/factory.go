package browser

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"agentium/internal/config"
	"agentium/internal/fingerprint"
	"agentium/internal/model"
	"agentium/internal/session"
	"agentium/internal/telemetry"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
	"github.com/google/uuid"
)

type Factory struct {
	config   config.Config
	resolver *fingerprint.Resolver
	mu       sync.Mutex
	browsers map[string]*sharedBrowser
	idleTTL  time.Duration
}

type sharedBrowser struct {
	rootBrowser    *rod.Browser
	refCount       int
	lastReleasedAt time.Time
}

const idleBrowserTTL = 30 * time.Second

func NewFactory(cfg config.Config) *Factory {
	return &Factory{
		config:   cfg,
		resolver: fingerprint.NewResolver(fingerprint.NewGeoResolver(cfg.GeoIPEndpoint, cfg.GeoIPTimeout)),
		browsers: make(map[string]*sharedBrowser),
		idleTTL:  idleBrowserTTL,
	}
}

func (f *Factory) Create(options model.SessionOptions) (*session.Runtime, error) {
	if options.EffectiveSessionMode() == model.SessionModePersistent {
		return f.createPersistent(options)
	}

	return f.createIncognito(options)
}

func (f *Factory) createIncognito(options model.SessionOptions) (*session.Runtime, error) {
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

	page, err := contextBrowser.Page(proto.TargetCreateTarget{})
	if err != nil {
		_ = contextBrowser.Close()
		f.releaseRootBrowser(proxyKey)
		return nil, fmt.Errorf("create page: %w", err)
	}

	id := uuid.NewString()

	browserVersion, err := rootBrowser.Version()
	if err != nil {
		_ = page.Close()
		_ = contextBrowser.Close()
		f.releaseRootBrowser(proxyKey)
		return nil, fmt.Errorf("read browser version: %w", err)
	}

	nativeMetrics, err := fingerprint.Probe(page)
	if err != nil {
		_ = page.Close()
		_ = contextBrowser.Close()
		f.releaseRootBrowser(proxyKey)
		return nil, fmt.Errorf("probe native browser metrics: %w", err)
	}

	profile, err := f.resolver.Resolve(options, browserVersion.UserAgent, id, nativeMetrics)
	if err != nil {
		_ = page.Close()
		_ = contextBrowser.Close()
		f.releaseRootBrowser(proxyKey)
		return nil, fmt.Errorf("resolve browser profile: %w", err)
	}

	if err := configurePage(page, profile, f.config); err != nil {
		_ = page.Close()
		_ = contextBrowser.Close()
		f.releaseRootBrowser(proxyKey)
		return nil, err
	}

	restoreNetwork := page.EnableDomain(&proto.NetworkEnable{})

	tracker := telemetry.NewTracker(100)
	bindNetworkEvents(page, tracker)

	closeFn := func() error {
		restoreNetwork()
		closeErr := closeSessionResources(page, contextBrowser)
		f.releaseRootBrowser(proxyKey)
		return closeErr
	}

	return session.NewRuntime(id, options, rootBrowser, contextBrowser, page, tracker, closeFn), nil
}

func (f *Factory) createPersistent(options model.SessionOptions) (*session.Runtime, error) {
	profileDir, err := os.MkdirTemp("", "agentium-profile-*")
	if err != nil {
		return nil, fmt.Errorf("create persistent profile dir: %w", err)
	}

	rootBrowser, err := f.launchRootBrowser(options.Proxy, profileDir)
	if err != nil {
		_ = removeProfileDir(profileDir)
		return nil, err
	}

	page, err := rootBrowser.Page(proto.TargetCreateTarget{})
	if err != nil {
		_ = rootBrowser.Close()
		_ = removeProfileDir(profileDir)
		return nil, fmt.Errorf("create page: %w", err)
	}

	id := uuid.NewString()

	browserVersion, err := rootBrowser.Version()
	if err != nil {
		_ = page.Close()
		_ = rootBrowser.Close()
		_ = removeProfileDir(profileDir)
		return nil, fmt.Errorf("read browser version: %w", err)
	}

	nativeMetrics, err := fingerprint.Probe(page)
	if err != nil {
		_ = page.Close()
		_ = rootBrowser.Close()
		_ = removeProfileDir(profileDir)
		return nil, fmt.Errorf("probe native browser metrics: %w", err)
	}

	profile, err := f.resolver.Resolve(options, browserVersion.UserAgent, id, nativeMetrics)
	if err != nil {
		_ = page.Close()
		_ = rootBrowser.Close()
		_ = removeProfileDir(profileDir)
		return nil, fmt.Errorf("resolve browser profile: %w", err)
	}

	if err := configurePage(page, profile, f.config); err != nil {
		_ = page.Close()
		_ = rootBrowser.Close()
		_ = removeProfileDir(profileDir)
		return nil, err
	}

	restoreNetwork := page.EnableDomain(&proto.NetworkEnable{})

	tracker := telemetry.NewTracker(100)
	bindNetworkEvents(page, tracker)

	closeFn := func() error {
		restoreNetwork()
		closeErr := closeSessionResources(page, rootBrowser)
		if removeErr := removeProfileDir(profileDir); closeErr == nil && removeErr != nil {
			return removeErr
		}
		return closeErr
	}

	return session.NewRuntime(id, options, rootBrowser, rootBrowser, page, tracker, closeFn), nil
}

func (f *Factory) acquireRootBrowser(proxyKey, proxyURL string) (*rod.Browser, error) {
	f.mu.Lock()
	if browser := f.browsers[proxyKey]; browser != nil {
		browser.refCount++
		browser.lastReleasedAt = time.Time{}
		root := browser.rootBrowser
		f.mu.Unlock()
		return root, nil
	}
	f.mu.Unlock()

	rootBrowser, err := f.launchRootBrowser(proxyURL, "")
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
	browser := f.browsers[proxyKey]
	if browser == nil {
		f.mu.Unlock()
		return
	}

	browser.refCount--
	if browser.refCount > 0 {
		f.mu.Unlock()
		return
	}

	browser.refCount = 0
	browser.lastReleasedAt = time.Now()
	releasedAt := browser.lastReleasedAt
	f.mu.Unlock()

	go f.closeIdleBrowser(proxyKey, releasedAt)
}

func (f *Factory) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	var firstErr error
	for key, browser := range f.browsers {
		if browser.rootBrowser != nil {
			if err := browser.rootBrowser.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		delete(f.browsers, key)
	}

	return firstErr
}

func (f *Factory) closeIdleBrowser(proxyKey string, releasedAt time.Time) {
	time.Sleep(f.idleTTL)

	f.mu.Lock()
	defer f.mu.Unlock()

	browser := f.browsers[proxyKey]
	if browser == nil || browser.refCount > 0 || !browser.lastReleasedAt.Equal(releasedAt) {
		return
	}

	if browser.rootBrowser != nil {
		_ = browser.rootBrowser.Close()
	}
	delete(f.browsers, proxyKey)
}

func (f *Factory) launchRootBrowser(proxyURL, userDataDir string) (*rod.Browser, error) {
	launch := launcher.New().
		Leakless(f.config.UseLeakless).
		Headless(false).
		NoSandbox(true).
		Set("disable-blink-features", "AutomationControlled")

	if chromeBin := resolveChromeBinary(f.config.ChromeBin); chromeBin != "" {
		launch = launch.Bin(chromeBin)
	}

	if proxyURL != "" {
		launch = launch.Proxy(proxyURL)
	}

	if strings.TrimSpace(userDataDir) != "" {
		launch = launch.UserDataDir(userDataDir)
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

func configurePage(page *rod.Page, profile fingerprint.Profile, cfg config.Config) error {
	width := profile.ViewportWidth
	height := profile.ViewportHeight
	if width == 0 {
		width = cfg.DefaultWidth
	}
	if height == 0 {
		height = cfg.DefaultHeight
	}
	scaleFactor := profile.DeviceScaleFactor
	if scaleFactor <= 0 {
		scaleFactor = 1
	}
	screenWidth := profile.ScreenWidth
	screenHeight := profile.ScreenHeight
	if screenWidth == 0 {
		screenWidth = width
	}
	if screenHeight == 0 {
		screenHeight = height
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width:             width,
		Height:            height,
		DeviceScaleFactor: scaleFactor,
		Mobile:            false,
		ScreenWidth:       &screenWidth,
		ScreenHeight:      &screenHeight,
	}); err != nil {
		return fmt.Errorf("set viewport: %w", err)
	}

	if err := applyPageEmulation(page, profile); err != nil {
		return err
	}

	return nil
}

func applyPageEmulation(page *rod.Page, profile fingerprint.Profile) error {
	if profile.UserAgent != "" {
		if err := (proto.NetworkSetUserAgentOverride{
			UserAgent:         profile.UserAgent,
			AcceptLanguage:    profile.AcceptLanguage,
			Platform:          profile.Platform,
			UserAgentMetadata: profile.UserAgentMetadata,
		}).Call(page); err != nil {
			return fmt.Errorf("set user agent: %w", err)
		}

		if err := (proto.EmulationSetUserAgentOverride{
			UserAgent:         profile.UserAgent,
			AcceptLanguage:    profile.AcceptLanguage,
			Platform:          profile.Platform,
			UserAgentMetadata: profile.UserAgentMetadata,
		}).Call(page); err != nil {
			return fmt.Errorf("set emulated user agent: %w", err)
		}
	}

	if extraHeaders := buildExtraHeaders(profile); len(extraHeaders) > 0 {
		if _, err := page.SetExtraHeaders(extraHeaders); err != nil {
			return fmt.Errorf("set extra http headers: %w", err)
		}
	}

	if profile.Platform != "" {
		if err := (proto.EmulationSetNavigatorOverrides{
			Platform: profile.Platform,
		}).Call(page); err != nil {
			return fmt.Errorf("set navigator platform: %w", err)
		}
	}

	if err := (proto.EmulationSetAutomationOverride{Enabled: false}).Call(page); err != nil {
		return fmt.Errorf("disable automation override: %w", err)
	}

	if err := (proto.EmulationSetHardwareConcurrencyOverride{
		HardwareConcurrency: profile.HardwareConcurrency,
	}).Call(page); err != nil {
		return fmt.Errorf("set hardware concurrency: %w", err)
	}

	if profile.TimezoneID != "" {
		if err := (proto.EmulationSetTimezoneOverride{TimezoneID: profile.TimezoneID}).Call(page); err != nil {
			return fmt.Errorf("set timezone: %w", err)
		}
	}

	if profile.Locale != "" {
		if err := (proto.EmulationSetLocaleOverride{Locale: normalizeLocale(profile.Locale)}).Call(page); err != nil {
			return fmt.Errorf("set locale: %w", err)
		}
	}

	if profile.Geolocation != nil {
		if err := (proto.EmulationSetGeolocationOverride{
			Latitude:  &profile.Geolocation.Latitude,
			Longitude: &profile.Geolocation.Longitude,
			Accuracy:  &profile.Geolocation.Accuracy,
		}).Call(page); err != nil {
			return fmt.Errorf("set geolocation: %w", err)
		}
	}

	if err := ApplyStealth(page, profile); err != nil {
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

func buildExtraHeaders(profile fingerprint.Profile) []string {
	headers := []string{}
	appendHeader := func(key, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		headers = append(headers, key, value)
	}

	appendHeader("Accept-Language", acceptLanguageHeaderValue(profile))

	if profile.UserAgentMetadata == nil {
		return headers
	}

	appendHeader("Sec-CH-UA", formatBrands(profile.UserAgentMetadata.Brands))
	appendHeader("Sec-CH-UA-Mobile", boolClientHint(profile.UserAgentMetadata.Mobile))
	appendHeader("Sec-CH-UA-Platform", quotedClientHint(profile.UserAgentMetadata.Platform))
	appendHeader("Sec-CH-UA-Platform-Version", quotedClientHint(profile.UserAgentMetadata.PlatformVersion))
	appendHeader("Sec-CH-UA-Arch", quotedClientHint(profile.UserAgentMetadata.Architecture))
	appendHeader("Sec-CH-UA-Bitness", quotedClientHint(profile.UserAgentMetadata.Bitness))
	appendHeader("Sec-CH-UA-Model", quotedClientHint(profile.UserAgentMetadata.Model))
	appendHeader("Sec-CH-UA-Full-Version", quotedClientHint(profile.UserAgentMetadata.FullVersion))
	appendHeader("Sec-CH-UA-Full-Version-List", formatBrands(profile.UserAgentMetadata.FullVersionList))

	return headers
}

func acceptLanguageHeaderValue(profile fingerprint.Profile) string {
	if len(profile.Languages) == 0 {
		return strings.TrimSpace(profile.AcceptLanguage)
	}

	parts := make([]string, 0, len(profile.Languages))
	for index, language := range profile.Languages {
		language = strings.TrimSpace(language)
		if language == "" {
			continue
		}
		if index == 0 {
			parts = append(parts, language)
			continue
		}
		parts = append(parts, fmt.Sprintf("%s;q=0.%d", language, 10-index))
	}

	return strings.Join(parts, ",")
}

func formatBrands(brands []*proto.EmulationUserAgentBrandVersion) string {
	parts := make([]string, 0, len(brands))
	for _, brand := range brands {
		if brand == nil || strings.TrimSpace(brand.Brand) == "" || strings.TrimSpace(brand.Version) == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf(`"%s";v="%s"`, brand.Brand, brand.Version))
	}
	return strings.Join(parts, ", ")
}

func quotedClientHint(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return fmt.Sprintf(`"%s"`, value)
}

func boolClientHint(value bool) string {
	if value {
		return "?1"
	}
	return "?0"
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
