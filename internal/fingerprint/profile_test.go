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

	profile, err := resolver.Resolve(model.SessionOptions{}, "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36")
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
	if profile.WebGLRenderer != "Intel Iris OpenGL Engine" {
		t.Fatalf("expected macOS renderer, got %q", profile.WebGLRenderer)
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
	}, "")
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
	if profile.WebGLVendor != "Google Inc. (Intel)" {
		t.Fatalf("expected windows webgl vendor, got %q", profile.WebGLVendor)
	}
	if profile.DeviceScaleFactor != 1 {
		t.Fatalf("expected windows device scale factor 1, got %v", profile.DeviceScaleFactor)
	}
}
