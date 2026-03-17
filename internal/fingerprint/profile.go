package fingerprint

import (
	"hash/fnv"
	"regexp"
	"strings"

	"agentium/internal/model"
	"github.com/go-rod/rod/lib/proto"
)

type Geolocation struct {
	Latitude  float64
	Longitude float64
	Accuracy  float64
}

type Profile struct {
	UserAgent           string
	Platform            string
	Vendor              string
	WebGLVendor         string
	WebGLRenderer       string
	FingerprintSeed     uint32
	CanvasNoiseR        int
	CanvasNoiseG        int
	CanvasNoiseB        int
	AudioNoise          float64
	ViewportWidth       int
	ViewportHeight      int
	ScreenWidth         int
	ScreenHeight        int
	AvailWidth          int
	AvailHeight         int
	DeviceScaleFactor   float64
	ColorDepth          int
	PixelDepth          int
	Locale              string
	Languages           []string
	AcceptLanguage      string
	TimezoneID          string
	HardwareConcurrency int
	DeviceMemory        int
	MaxTouchPoints      int
	UserAgentMetadata   *proto.EmulationUserAgentMetadata
	Geolocation         *Geolocation
}

type geoResolver interface {
	Resolve(proxyURL string) (GeoData, error)
}

type Resolver struct {
	geo geoResolver
}

func NewResolver(geo geoResolver) *Resolver {
	return &Resolver{geo: geo}
}

func (r *Resolver) Resolve(options model.SessionOptions, browserUserAgent, sessionSeed string, native NativeMetrics) (Profile, error) {
	effectiveUA := strings.TrimSpace(options.UserAgent)
	if effectiveUA == "" {
		effectiveUA = strings.TrimSpace(browserUserAgent)
	}

	uaProfile := parseUserAgent(effectiveUA)
	geo := GeoData{}
	if r.geo != nil {
		resolved, err := r.geo.Resolve(options.Proxy)
		if err == nil {
			geo = resolved
		}
	}

	locale := strings.TrimSpace(options.Locale)
	if locale == "" {
		locale = geo.Locale
	}
	if locale == "" {
		locale = "en-US"
	}

	timezoneID := strings.TrimSpace(options.TimezoneID)
	if timezoneID == "" {
		timezoneID = geo.TimezoneID
	}

	languages := buildLanguages(locale)
	profile := Profile{
		UserAgent:           effectiveUA,
		Platform:            uaProfile.NavigatorPlatform,
		Vendor:              uaProfile.Vendor,
		WebGLVendor:         uaProfile.WebGLVendor,
		WebGLRenderer:       uaProfile.WebGLRenderer,
		Locale:              locale,
		Languages:           languages,
		AcceptLanguage:      strings.Join(languages, ","),
		TimezoneID:          timezoneID,
		HardwareConcurrency: 8,
		DeviceMemory:        8,
		MaxTouchPoints:      0,
		UserAgentMetadata:   uaProfile.Metadata,
	}
	templateKey := stableFingerprintKey(options, effectiveUA, locale, timezoneID, sessionSeed)
	seed := stableSeed(templateKey)
	profile.FingerprintSeed = seed
	profile.CanvasNoiseR, profile.CanvasNoiseG, profile.CanvasNoiseB = canvasNoise(seed)
	profile.AudioNoise = audioNoise(seed)
	template := selectHardwareTemplate(uaProfile.OSFamily, seed)
	profile.ViewportWidth = template.ViewportWidth
	profile.ViewportHeight = template.ViewportHeight
	profile.ScreenWidth = template.ScreenWidth
	profile.ScreenHeight = template.ScreenHeight
	profile.AvailWidth = template.AvailWidth
	profile.AvailHeight = template.AvailHeight
	profile.DeviceScaleFactor = template.DeviceScaleFactor
	profile.ColorDepth = template.ColorDepth
	profile.PixelDepth = template.PixelDepth
	if template.WebGLVendor != "" {
		profile.WebGLVendor = template.WebGLVendor
	}
	if template.WebGLRenderer != "" {
		profile.WebGLRenderer = template.WebGLRenderer
	}
	if template.HardwareConcurrency > 0 {
		profile.HardwareConcurrency = template.HardwareConcurrency
	}
	if template.DeviceMemory > 0 {
		profile.DeviceMemory = template.DeviceMemory
	}

	if strings.TrimSpace(options.UserAgent) == "" {
		applyNativeMetrics(&profile, native)
	}

	if geo.HasLocation() {
		profile.Geolocation = &Geolocation{
			Latitude:  geo.Latitude,
			Longitude: geo.Longitude,
			Accuracy:  geo.Accuracy,
		}
	}

	return profile, nil
}

func applyNativeMetrics(profile *Profile, native NativeMetrics) {
	if native.ColorDepth > 0 {
		profile.ColorDepth = native.ColorDepth
	}
	if native.PixelDepth > 0 {
		profile.PixelDepth = native.PixelDepth
	}
}

type uaProfile struct {
	NavigatorPlatform string
	Vendor            string
	WebGLVendor       string
	WebGLRenderer     string
	Metadata          *proto.EmulationUserAgentMetadata
	OSFamily          string
}

var (
	chromeVersionPattern  = regexp.MustCompile(`(?:Chrome|Chromium|Edg|EdgA|EdgiOS)/([0-9.]+)`)
	macOSVersionPattern   = regexp.MustCompile(`Mac OS X ([0-9_]+)`)
	windowsVersionPattern = regexp.MustCompile(`Windows NT ([0-9.]+)`)
)

func parseUserAgent(userAgent string) uaProfile {
	version := extractChromeVersion(userAgent)
	major := majorVersion(version)
	platform := "Windows"
	navigatorPlatform := "Win32"
	platformVersion := "10.0.0"
	architecture := "x86"
	bitness := "64"
	mobile := strings.Contains(userAgent, "Mobile")
	webGLVendor := "Google Inc. (Intel)"
	webGLRenderer := "ANGLE (Intel, Intel(R) UHD Graphics Direct3D11 vs_5_0 ps_5_0, D3D11)"
	osFamily := "windows"

	switch {
	case strings.Contains(userAgent, "Macintosh"):
		platform = "macOS"
		navigatorPlatform = "MacIntel"
		platformVersion = normalizePlatformVersion(findMatch(macOSVersionPattern, userAgent, 1), "10.15.7")
		webGLVendor = "Intel Inc."
		webGLRenderer = "Intel Iris OpenGL Engine"
		osFamily = "mac"
	case strings.Contains(userAgent, "Linux"):
		platform = "Linux"
		navigatorPlatform = "Linux x86_64"
		platformVersion = "6.1.0"
		webGLVendor = "Google Inc. (Intel)"
		webGLRenderer = "ANGLE (Intel, Mesa Intel(R) UHD Graphics 620 (KBL GT2), OpenGL 4.6)"
		osFamily = "linux"
	case strings.Contains(userAgent, "Windows"):
		platformVersion = normalizePlatformVersion(findMatch(windowsVersionPattern, userAgent, 1), "10.0.0")
	}

	if strings.Contains(userAgent, "ARM") || strings.Contains(strings.ToLower(userAgent), "aarch64") {
		architecture = "arm"
	}

	metadata := &proto.EmulationUserAgentMetadata{
		Brands:          buildBrands(major, version, false),
		FullVersionList: buildBrands(major, version, true),
		FullVersion:     version,
		Platform:        platform,
		PlatformVersion: platformVersion,
		Architecture:    architecture,
		Model:           "",
		Mobile:          mobile,
		Bitness:         bitness,
	}

	return uaProfile{
		NavigatorPlatform: navigatorPlatform,
		Vendor:            "Google Inc.",
		WebGLVendor:       webGLVendor,
		WebGLRenderer:     webGLRenderer,
		Metadata:          metadata,
		OSFamily:          osFamily,
	}
}

func buildLanguages(locale string) []string {
	if locale == "" {
		return []string{"en-US", "en"}
	}

	primary := strings.Split(locale, "-")[0]
	languages := []string{locale}
	if primary != "" && primary != locale {
		languages = append(languages, primary)
	}
	return languages
}

func extractChromeVersion(userAgent string) string {
	matches := chromeVersionPattern.FindStringSubmatch(userAgent)
	if len(matches) == 2 {
		return matches[1]
	}
	return "114.0.0.0"
}

func majorVersion(version string) string {
	part := strings.Split(version, ".")[0]
	if part == "" {
		return "114"
	}
	return part
}

func buildBrands(major, version string, full bool) []*proto.EmulationUserAgentBrandVersion {
	chromiumVersion := major
	notABrandVersion := "99"
	if full {
		chromiumVersion = version
		notABrandVersion = "99.0.0.0"
	}

	return []*proto.EmulationUserAgentBrandVersion{
		{Brand: "Chromium", Version: chromiumVersion},
		{Brand: "Google Chrome", Version: chromiumVersion},
		{Brand: "Not:A-Brand", Version: notABrandVersion},
	}
}

func normalizePlatformVersion(value, fallback string) string {
	if value == "" {
		return fallback
	}

	value = strings.ReplaceAll(value, "_", ".")
	if strings.Count(value, ".") == 1 {
		return value + ".0"
	}
	return value
}

func findMatch(pattern *regexp.Regexp, value string, group int) string {
	matches := pattern.FindStringSubmatch(value)
	if len(matches) <= group {
		return ""
	}
	return matches[group]
}

type hardwareTemplate struct {
	ViewportWidth       int
	ViewportHeight      int
	ScreenWidth         int
	ScreenHeight        int
	AvailWidth          int
	AvailHeight         int
	DeviceScaleFactor   float64
	ColorDepth          int
	PixelDepth          int
	WebGLVendor         string
	WebGLRenderer       string
	HardwareConcurrency int
	DeviceMemory        int
}

var windowsTemplates = []hardwareTemplate{
	{ViewportWidth: 1365, ViewportHeight: 728, ScreenWidth: 1366, ScreenHeight: 768, AvailWidth: 1366, AvailHeight: 728, DeviceScaleFactor: 1, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Google Inc. (Intel)", WebGLRenderer: "ANGLE (Intel, Intel(R) UHD Graphics 620 Direct3D11 vs_5_0 ps_5_0, D3D11)", HardwareConcurrency: 4, DeviceMemory: 8},
	{ViewportWidth: 1536, ViewportHeight: 824, ScreenWidth: 1536, ScreenHeight: 864, AvailWidth: 1536, AvailHeight: 824, DeviceScaleFactor: 1, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Google Inc. (Intel)", WebGLRenderer: "ANGLE (Intel, Intel(R) Iris(R) Xe Graphics Direct3D11 vs_5_0 ps_5_0, D3D11)", HardwareConcurrency: 8, DeviceMemory: 8},
	{ViewportWidth: 1600, ViewportHeight: 860, ScreenWidth: 1600, ScreenHeight: 900, AvailWidth: 1600, AvailHeight: 860, DeviceScaleFactor: 1, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Google Inc. (AMD)", WebGLRenderer: "ANGLE (AMD, AMD Radeon(TM) Graphics Direct3D11 vs_5_0 ps_5_0, D3D11)", HardwareConcurrency: 8, DeviceMemory: 16},
	{ViewportWidth: 1920, ViewportHeight: 1040, ScreenWidth: 1920, ScreenHeight: 1080, AvailWidth: 1920, AvailHeight: 1040, DeviceScaleFactor: 1, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Google Inc. (NVIDIA)", WebGLRenderer: "ANGLE (NVIDIA, NVIDIA GeForce GTX 1650 Direct3D11 vs_5_0 ps_5_0, D3D11)", HardwareConcurrency: 8, DeviceMemory: 16},
}

var macTemplates = []hardwareTemplate{
	{ViewportWidth: 1440, ViewportHeight: 860, ScreenWidth: 1440, ScreenHeight: 900, AvailWidth: 1440, AvailHeight: 860, DeviceScaleFactor: 2, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Intel Inc.", WebGLRenderer: "Intel Iris OpenGL Engine", HardwareConcurrency: 8, DeviceMemory: 8},
	{ViewportWidth: 1512, ViewportHeight: 945, ScreenWidth: 1512, ScreenHeight: 982, AvailWidth: 1512, AvailHeight: 945, DeviceScaleFactor: 2, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Apple", WebGLRenderer: "Apple M1", HardwareConcurrency: 8, DeviceMemory: 8},
	{ViewportWidth: 1680, ViewportHeight: 1010, ScreenWidth: 1680, ScreenHeight: 1050, AvailWidth: 1680, AvailHeight: 1010, DeviceScaleFactor: 2, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "AMD", WebGLRenderer: "AMD Radeon Pro 5300M OpenGL Engine", HardwareConcurrency: 8, DeviceMemory: 16},
}

var linuxTemplates = []hardwareTemplate{
	{ViewportWidth: 1365, ViewportHeight: 728, ScreenWidth: 1366, ScreenHeight: 768, AvailWidth: 1366, AvailHeight: 728, DeviceScaleFactor: 1, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Google Inc. (Intel)", WebGLRenderer: "ANGLE (Intel, Mesa Intel(R) UHD Graphics 620 (KBL GT2), OpenGL 4.6)", HardwareConcurrency: 4, DeviceMemory: 8},
	{ViewportWidth: 1600, ViewportHeight: 860, ScreenWidth: 1600, ScreenHeight: 900, AvailWidth: 1600, AvailHeight: 860, DeviceScaleFactor: 1, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Google Inc. (AMD)", WebGLRenderer: "ANGLE (AMD, AMD Radeon Vega 8 Graphics (raven, LLVM 15.0.7), OpenGL 4.6)", HardwareConcurrency: 8, DeviceMemory: 8},
	{ViewportWidth: 1920, ViewportHeight: 1040, ScreenWidth: 1920, ScreenHeight: 1080, AvailWidth: 1920, AvailHeight: 1040, DeviceScaleFactor: 1, ColorDepth: 24, PixelDepth: 24, WebGLVendor: "Google Inc. (NVIDIA)", WebGLRenderer: "ANGLE (NVIDIA, NVIDIA GeForce GTX 1660 Ti/PCIe/SSE2, OpenGL 4.6)", HardwareConcurrency: 8, DeviceMemory: 16},
}

func selectHardwareTemplate(osFamily string, seed uint32) hardwareTemplate {
	templates := windowsTemplates
	switch osFamily {
	case "mac":
		templates = macTemplates
	case "linux":
		templates = linuxTemplates
	}

	return templates[int(seed)%len(templates)]
}

func stableSeed(value string) uint32 {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(value))
	return hasher.Sum32()
}

func stableFingerprintKey(options model.SessionOptions, effectiveUA, locale, timezoneID, sessionSeed string) string {
	if value := strings.TrimSpace(options.ProfileID); value != "" {
		return "profile:" + value
	}

	parts := []string{
		strings.TrimSpace(options.Proxy),
		strings.TrimSpace(effectiveUA),
		strings.TrimSpace(locale),
		strings.TrimSpace(timezoneID),
	}
	if joined := strings.Join(parts, "|"); strings.Trim(joined, "|") != "" {
		return joined
	}

	return sessionSeed
}

func canvasNoise(seed uint32) (int, int, int) {
	r := int(seed%5) - 2
	g := int((seed/7)%5) - 2
	b := int((seed/13)%5) - 2
	if r == 0 && g == 0 && b == 0 {
		r = 1
	}
	return r, g, b
}

func audioNoise(seed uint32) float64 {
	step := float64((seed%17)+1) / 1000000
	if seed%2 == 0 {
		return step
	}
	return -step
}
