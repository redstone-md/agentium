package fingerprint

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
	"unicode"
)

type GeoData struct {
	CountryCode string
	Locale      string
	TimezoneID  string
	Latitude    float64
	Longitude   float64
	Accuracy    float64
}

func (g GeoData) HasLocation() bool {
	return g.Latitude != 0 || g.Longitude != 0
}

type GeoResolver struct {
	client   *http.Client
	endpoint string
	mu       sync.Mutex
	cache    map[string]GeoData
}

func NewGeoResolver(endpoint string, timeout time.Duration) *GeoResolver {
	if endpoint == "" {
		endpoint = "https://ipwho.is/"
	}
	if timeout <= 0 {
		timeout = 8 * time.Second
	}

	return &GeoResolver{
		client: &http.Client{
			Timeout: timeout,
		},
		endpoint: endpoint,
		cache:    make(map[string]GeoData),
	}
}

func (r *GeoResolver) Resolve(proxyURL string) (GeoData, error) {
	cacheKey := proxyURL
	if cacheKey == "" {
		cacheKey = "direct"
	}

	r.mu.Lock()
	if cached, ok := r.cache[cacheKey]; ok {
		r.mu.Unlock()
		return cached, nil
	}
	r.mu.Unlock()

	client := r.client
	if proxyURL != "" {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		parsed, err := url.Parse(proxyURL)
		if err != nil {
			return GeoData{}, fmt.Errorf("parse proxy url: %w", err)
		}
		transport.Proxy = http.ProxyURL(parsed)
		client = &http.Client{
			Timeout:   r.client.Timeout,
			Transport: transport,
		}
	}

	request, err := http.NewRequest(http.MethodGet, r.endpoint, nil)
	if err != nil {
		return GeoData{}, fmt.Errorf("create geo lookup request: %w", err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Agentium/1.1")

	response, err := client.Do(request)
	if err != nil {
		return GeoData{}, fmt.Errorf("geo lookup request failed: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return GeoData{}, fmt.Errorf("geo lookup returned status %d", response.StatusCode)
	}

	var payload struct {
		Success     *bool   `json:"success"`
		Message     string  `json:"message"`
		CountryCode string  `json:"country_code"`
		Latitude    float64 `json:"latitude"`
		Longitude   float64 `json:"longitude"`
		Timezone    struct {
			ID string `json:"id"`
		} `json:"timezone"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return GeoData{}, fmt.Errorf("decode geo lookup response: %w", err)
	}

	if payload.Success != nil && !*payload.Success {
		return GeoData{}, fmt.Errorf("geo lookup failed: %s", strings.TrimSpace(payload.Message))
	}

	geo := GeoData{
		CountryCode: strings.ToUpper(strings.TrimSpace(payload.CountryCode)),
		Locale:      localeFromCountryCode(payload.CountryCode),
		TimezoneID:  strings.TrimSpace(payload.Timezone.ID),
		Latitude:    payload.Latitude,
		Longitude:   payload.Longitude,
		Accuracy:    approximateAccuracy(payload.Latitude, payload.Longitude),
	}

	r.mu.Lock()
	r.cache[cacheKey] = geo
	r.mu.Unlock()

	return geo, nil
}

func approximateAccuracy(latitude, longitude float64) float64 {
	if latitude == 0 && longitude == 0 {
		return 0
	}
	return 20
}

func localeFromCountryCode(countryCode string) string {
	switch strings.ToUpper(strings.TrimSpace(countryCode)) {
	case "US":
		return "en-US"
	case "GB":
		return "en-GB"
	case "CA":
		return "en-CA"
	case "AU":
		return "en-AU"
	case "NZ":
		return "en-NZ"
	case "IE":
		return "en-IE"
	case "DE":
		return "de-DE"
	case "AT":
		return "de-AT"
	case "CH":
		return "de-CH"
	case "FR":
		return "fr-FR"
	case "IT":
		return "it-IT"
	case "ES":
		return "es-ES"
	case "MX":
		return "es-MX"
	case "AR":
		return "es-AR"
	case "PT":
		return "pt-PT"
	case "BR":
		return "pt-BR"
	case "NL":
		return "nl-NL"
	case "BE":
		return "nl-BE"
	case "PL":
		return "pl-PL"
	case "CZ":
		return "cs-CZ"
	case "SK":
		return "sk-SK"
	case "UA":
		return "uk-UA"
	case "RU":
		return "ru-RU"
	case "TR":
		return "tr-TR"
	case "RO":
		return "ro-RO"
	case "HU":
		return "hu-HU"
	case "SE":
		return "sv-SE"
	case "NO":
		return "nb-NO"
	case "DK":
		return "da-DK"
	case "FI":
		return "fi-FI"
	case "GR":
		return "el-GR"
	case "IL":
		return "he-IL"
	case "SA":
		return "ar-SA"
	case "AE":
		return "ar-AE"
	case "EG":
		return "ar-EG"
	case "JP":
		return "ja-JP"
	case "KR":
		return "ko-KR"
	case "CN":
		return "zh-CN"
	case "TW":
		return "zh-TW"
	case "HK":
		return "zh-HK"
	case "IN":
		return "en-IN"
	case "ID":
		return "id-ID"
	case "TH":
		return "th-TH"
	case "VN":
		return "vi-VN"
	case "MY":
		return "en-MY"
	case "SG":
		return "en-SG"
	case "PH":
		return "en-PH"
	case "ZA":
		return "en-ZA"
	default:
		if code := strings.ToUpper(strings.TrimSpace(countryCode)); len(code) == 2 && isASCII(code) {
			return "en-" + code
		}
		return "en-US"
	}
}

func isASCII(value string) bool {
	for _, char := range value {
		if char > unicode.MaxASCII || !unicode.IsLetter(char) {
			return false
		}
	}
	return true
}
