package fingerprint

import (
	"fmt"

	"github.com/go-rod/rod"
)

type NativeMetrics struct {
	Platform         string   `json:"platform"`
	Language         string   `json:"language"`
	Languages        []string `json:"languages"`
	ScreenWidth      int      `json:"screenWidth"`
	ScreenHeight     int      `json:"screenHeight"`
	AvailWidth       int      `json:"availWidth"`
	AvailHeight      int      `json:"availHeight"`
	DevicePixelRatio float64  `json:"devicePixelRatio"`
	ColorDepth       int      `json:"colorDepth"`
	PixelDepth       int      `json:"pixelDepth"`
	WebGLVendor      string   `json:"webglVendor"`
	WebGLRenderer    string   `json:"webglRenderer"`
}

const probeScript = `
() => {
  const canvas = document.createElement('canvas');
  const gl = canvas.getContext('webgl') || canvas.getContext('experimental-webgl');
  const debugInfo = gl ? gl.getExtension('WEBGL_debug_renderer_info') : null;
  const webglVendor = debugInfo ? gl.getParameter(debugInfo.UNMASKED_VENDOR_WEBGL) : '';
  const webglRenderer = debugInfo ? gl.getParameter(debugInfo.UNMASKED_RENDERER_WEBGL) : '';

  return {
    platform: navigator.platform || '',
    language: navigator.language || '',
    languages: Array.isArray(navigator.languages) ? navigator.languages : [],
    screenWidth: window.screen?.width || 0,
    screenHeight: window.screen?.height || 0,
    availWidth: window.screen?.availWidth || 0,
    availHeight: window.screen?.availHeight || 0,
    devicePixelRatio: window.devicePixelRatio || 1,
    colorDepth: window.screen?.colorDepth || 24,
    pixelDepth: window.screen?.pixelDepth || 24,
    webglVendor,
    webglRenderer
  };
}
`

func Probe(page *rod.Page) (NativeMetrics, error) {
	result, err := page.Eval(probeScript)
	if err != nil {
		return NativeMetrics{}, fmt.Errorf("probe native metrics: %w", err)
	}

	var metrics NativeMetrics
	if err := result.Value.Unmarshal(&metrics); err != nil {
		return NativeMetrics{}, fmt.Errorf("decode native metrics: %w", err)
	}

	return metrics, nil
}
