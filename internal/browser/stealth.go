package browser

import (
	"fmt"
	"strings"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

type OverrideProfile struct {
	Platform            string
	Languages           []string
	HardwareConcurrency int
	DeviceMemory        int
}

func ApplyStealth(page *rod.Page, userAgent, locale string) error {
	profile := inferOverrideProfile(userAgent, locale)
	languages := "['en-US','en']"
	if len(profile.Languages) > 0 {
		quoted := make([]string, 0, len(profile.Languages))
		for _, language := range profile.Languages {
			quoted = append(quoted, fmt.Sprintf("'%s'", language))
		}
		languages = "[" + strings.Join(quoted, ",") + "]"
	}

	script := fmt.Sprintf(`(() => {
		const define = (target, key, value) => {
			Object.defineProperty(target, key, { get: () => value, configurable: true });
		};
		const overrideNavigatorValue = (key, value) => {
			const proto = Object.getPrototypeOf(navigator);
			const descriptor = Object.getOwnPropertyDescriptor(proto, key);
			if (!descriptor || descriptor.configurable) {
				define(proto, key, value);
				return;
			}
			define(navigator, key, value);
		};
		const removeWebdriver = () => {
			try {
				delete navigator.webdriver;
			} catch (_) {}

			const proto = Object.getPrototypeOf(navigator);
			const descriptor = Object.getOwnPropertyDescriptor(proto, 'webdriver');
			if (descriptor && descriptor.configurable) {
				try {
					delete proto.webdriver;
				} catch (_) {}
			}
		};
		overrideNavigatorValue('platform', %q);
		overrideNavigatorValue('hardwareConcurrency', %d);
		overrideNavigatorValue('deviceMemory', %d);
		overrideNavigatorValue('languages', %s);
		removeWebdriver();
		if (window.RTCPeerConnection) {
			window.RTCPeerConnection = class BlockedPeerConnection {
				constructor() {
					throw new Error('WebRTC disabled by Agentium');
				}
			};
		}
		if (window.webkitRTCPeerConnection) {
			window.webkitRTCPeerConnection = window.RTCPeerConnection;
		}
	})();`, profile.Platform, profile.HardwareConcurrency, profile.DeviceMemory, languages)

	if _, err := (proto.PageAddScriptToEvaluateOnNewDocument{
		Source:         script,
		RunImmediately: true,
	}).Call(page); err != nil {
		return fmt.Errorf("apply overrides: %w", err)
	}

	return nil
}

func inferOverrideProfile(userAgent, locale string) OverrideProfile {
	languages := []string{"en-US", "en"}
	if locale != "" {
		primary := strings.Split(locale, "-")[0]
		languages = []string{locale}
		if primary != locale {
			languages = append(languages, primary)
		}
	}

	profile := OverrideProfile{
		Platform:            "Win32",
		Languages:           languages,
		HardwareConcurrency: 8,
		DeviceMemory:        8,
	}

	switch {
	case strings.Contains(userAgent, "Macintosh"):
		profile.Platform = "MacIntel"
	case strings.Contains(userAgent, "Linux"):
		profile.Platform = "Linux x86_64"
	}

	return profile
}
