package fingerprint

import (
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

func (r *Resolver) Resolve(options model.SessionOptions, browserUserAgent string) (Profile, error) {
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
		Locale:              locale,
		Languages:           languages,
		AcceptLanguage:      strings.Join(languages, ","),
		TimezoneID:          timezoneID,
		HardwareConcurrency: 8,
		DeviceMemory:        8,
		MaxTouchPoints:      0,
		UserAgentMetadata:   uaProfile.Metadata,
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

type uaProfile struct {
	NavigatorPlatform string
	Vendor            string
	Metadata          *proto.EmulationUserAgentMetadata
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

	switch {
	case strings.Contains(userAgent, "Macintosh"):
		platform = "macOS"
		navigatorPlatform = "MacIntel"
		platformVersion = normalizePlatformVersion(findMatch(macOSVersionPattern, userAgent, 1), "10.15.7")
	case strings.Contains(userAgent, "Linux"):
		platform = "Linux"
		navigatorPlatform = "Linux x86_64"
		platformVersion = "6.1.0"
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
		Metadata:          metadata,
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
