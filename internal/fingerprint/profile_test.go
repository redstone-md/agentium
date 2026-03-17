package fingerprint

import (
	"testing"

	"agentium/internal/model"
)

type stubGeoResolver struct {
	geo GeoData
	err error
}

func (s stubGeoResolver) Resolve(string) (GeoData, error) {
	return s.geo, s.err
}

func TestResolverUsesBrowserUserAgentWhenRequestOmitsIt(t *testing.T) {
	resolver := NewResolver(stubGeoResolver{
		geo: GeoData{
			CountryCode: "CZ",
			Locale:      "cs-CZ",
			TimezoneID:  "Europe/Prague",
			Latitude:    50.08,
			Longitude:   14.43,
			Accuracy:    20,
		},
	})

	profile, err := resolver.Resolve(model.SessionOptions{}, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36", "session-a", NativeMetrics{})
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}

	if profile.Platform != "MacIntel" {
		t.Fatalf("expected MacIntel platform, got %q", profile.Platform)
	}
	if profile.Locale != "cs-CZ" {
		t.Fatalf("expected locale from geo, got %q", profile.Locale)
	}
	if profile.TimezoneID != "Europe/Prague" {
		t.Fatalf("expected timezone from geo, got %q", profile.TimezoneID)
	}
	if profile.Geolocation == nil {
		t.Fatal("expected geolocation to be populated")
	}
	if profile.UserAgentMetadata == nil || profile.UserAgentMetadata.Platform != "macOS" {
		t.Fatal("expected user agent metadata to match macOS")
	}
	if profile.WebGLRenderer == "" {
		t.Fatalf("expected macOS preset to populate a renderer, got %q", profile.WebGLRenderer)
	}
	if profile.ScreenWidth == 0 || profile.ScreenHeight == 0 {
		t.Fatal("expected hardware template to populate screen dimensions")
	}
}

func TestResolverHonorsExplicitOverrides(t *testing.T) {
	resolver := NewResolver(stubGeoResolver{
		geo: GeoData{
			CountryCode: "US",
			Locale:      "en-US",
			TimezoneID:  "America/New_York",
		},
	})

	profile, err := resolver.Resolve(model.SessionOptions{
		Locale:     "de-DE",
		TimezoneID: "Europe/Berlin",
		UserAgent:  "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
	}, "", "session-b", NativeMetrics{})
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}

	if profile.Locale != "de-DE" {
		t.Fatalf("expected explicit locale, got %q", profile.Locale)
	}
	if profile.TimezoneID != "Europe/Berlin" {
		t.Fatalf("expected explicit timezone, got %q", profile.TimezoneID)
	}
	if profile.Platform != "Win32" {
		t.Fatalf("expected Win32 platform, got %q", profile.Platform)
	}
	if profile.WebGLVendor == "" || profile.WebGLRenderer == "" {
		t.Fatalf("expected windows hardware preset to populate webgl profile, got %q / %q", profile.WebGLVendor, profile.WebGLRenderer)
	}
	if profile.DeviceScaleFactor != 1 {
		t.Fatalf("expected windows device scale factor 1, got %v", profile.DeviceScaleFactor)
	}
	if profile.AudioNoise == 0 {
		t.Fatal("expected audio noise to be populated")
	}
}

func TestResolverUsesStableFingerprintForMatchingInputs(t *testing.T) {
	resolver := NewResolver(stubGeoResolver{
		geo: GeoData{
			CountryCode: "ES",
			Locale:      "es-ES",
			TimezoneID:  "Europe/Madrid",
		},
	})

	first, err := resolver.Resolve(model.SessionOptions{}, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", "session-1", NativeMetrics{})
	if err != nil {
		t.Fatalf("resolve first profile: %v", err)
	}

	second, err := resolver.Resolve(model.SessionOptions{}, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", "session-2", NativeMetrics{})
	if err != nil {
		t.Fatalf("resolve second profile: %v", err)
	}

	if first.ScreenWidth != second.ScreenWidth || first.ScreenHeight != second.ScreenHeight || first.ViewportWidth != second.ViewportWidth {
		t.Fatal("expected identical inputs to keep a stable hardware template")
	}
	if first.AudioNoise != second.AudioNoise || first.CanvasNoiseR != second.CanvasNoiseR || first.CanvasNoiseG != second.CanvasNoiseG || first.CanvasNoiseB != second.CanvasNoiseB {
		t.Fatal("expected identical inputs to keep stable media fingerprint noise")
	}
}

func TestResolverUsesProfileIDForDistinctStableProfiles(t *testing.T) {
	resolver := NewResolver(stubGeoResolver{
		geo: GeoData{
			CountryCode: "ES",
			Locale:      "es-ES",
			TimezoneID:  "Europe/Madrid",
		},
	})

	first, err := resolver.Resolve(model.SessionOptions{ProfileID: "alpha"}, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", "session-1", NativeMetrics{})
	if err != nil {
		t.Fatalf("resolve first profile: %v", err)
	}

	second, err := resolver.Resolve(model.SessionOptions{ProfileID: "beta"}, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", "session-2", NativeMetrics{})
	if err != nil {
		t.Fatalf("resolve second profile: %v", err)
	}

	if first.ScreenWidth == second.ScreenWidth && first.ScreenHeight == second.ScreenHeight && first.ViewportWidth == second.ViewportWidth &&
		first.AudioNoise == second.AudioNoise && first.CanvasNoiseR == second.CanvasNoiseR && first.CanvasNoiseG == second.CanvasNoiseG && first.CanvasNoiseB == second.CanvasNoiseB {
		t.Fatal("expected different profile IDs to produce different stable profiles")
	}
}

func TestResolverUsesNativeGraphicsMetricsForImplicitProfiles(t *testing.T) {
	resolver := NewResolver(stubGeoResolver{
		geo: GeoData{
			CountryCode: "ES",
			Locale:      "es-ES",
			TimezoneID:  "Europe/Madrid",
		},
	})

	profile, err := resolver.Resolve(model.SessionOptions{}, "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36", "session-native", NativeMetrics{
		ScreenWidth:      1920,
		ScreenHeight:     1080,
		AvailWidth:       1920,
		AvailHeight:      1040,
		DevicePixelRatio: 1.25,
		ColorDepth:       30,
		PixelDepth:       30,
		WebGLVendor:      "Intel Inc.",
		WebGLRenderer:    "Intel(R) Iris(R) Xe Graphics",
	})
	if err != nil {
		t.Fatalf("resolve profile: %v", err)
	}

	if profile.ScreenWidth == 1920 && profile.ScreenHeight == 1080 {
		t.Fatalf("expected session hardware template to stay isolated from native screen metrics")
	}
	if profile.WebGLRenderer == "Intel(R) Iris(R) Xe Graphics" {
		t.Fatalf("expected webgl renderer to stay on the session preset, got %q", profile.WebGLRenderer)
	}
	if profile.DeviceScaleFactor == 1.25 {
		t.Fatalf("expected device scale factor to come from session template, got %v", profile.DeviceScaleFactor)
	}
	if profile.ColorDepth != 30 || profile.PixelDepth != 30 {
		t.Fatalf("expected native color depth overrides, got %d / %d", profile.ColorDepth, profile.PixelDepth)
	}
}
