package browser

import (
	"encoding/json"
	"fmt"

	"agentium/internal/fingerprint"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func ApplyStealth(page *rod.Page, profile fingerprint.Profile) error {
	payload, err := json.Marshal(map[string]any{
		"userAgent":           profile.UserAgent,
		"platform":            profile.Platform,
		"vendor":              profile.Vendor,
		"languages":           profile.Languages,
		"language":            firstLanguage(profile.Languages),
		"hardwareConcurrency": profile.HardwareConcurrency,
		"deviceMemory":        profile.DeviceMemory,
		"maxTouchPoints":      profile.MaxTouchPoints,
		"userAgentData":       buildUserAgentData(profile),
	})
	if err != nil {
		return fmt.Errorf("marshal stealth profile: %w", err)
	}

	script := fmt.Sprintf(`(() => {
		const profile = %s;
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
		const createUserAgentData = (data) => {
			const lowEntropy = {
				brands: data.brands || [],
				mobile: !!data.mobile,
				platform: data.platform || ''
			};
			const highEntropy = {
				architecture: data.architecture || '',
				bitness: data.bitness || '',
				brands: data.brands || [],
				fullVersionList: data.fullVersionList || [],
				mobile: !!data.mobile,
				model: data.model || '',
				platform: data.platform || '',
				platformVersion: data.platformVersion || '',
				uaFullVersion: data.uaFullVersion || ''
			};
			return {
				...lowEntropy,
				getHighEntropyValues: async (hints) => {
					const result = {};
					for (const hint of hints) {
						if (Object.prototype.hasOwnProperty.call(highEntropy, hint)) {
							result[hint] = highEntropy[hint];
						}
					}
					return result;
				},
				toJSON: () => ({ ...lowEntropy })
			};
		};
		const patchWebRTC = () => {
			const NativePeerConnection = window.RTCPeerConnection || window.webkitRTCPeerConnection;
			if (!NativePeerConnection) {
				return;
			}
			const sanitizeSDP = (sdp) => {
				if (!sdp) {
					return sdp;
				}
				return sdp
					.split('\n')
					.filter((line) => !line.includes(' typ host '))
					.join('\n');
			};
			const wrapDescription = (description) => {
				if (!description || !description.sdp) {
					return description;
				}
				return {
					type: description.type,
					sdp: sanitizeSDP(description.sdp)
				};
			};
			const WrappedPeerConnection = function(config, constraints) {
				const safeConfig = Object.assign({}, config || {});
				safeConfig.iceServers = Array.isArray(safeConfig.iceServers) ? safeConfig.iceServers : [];
				safeConfig.iceTransportPolicy = safeConfig.iceTransportPolicy || 'relay';
				const peer = new NativePeerConnection(safeConfig, constraints);
				const nativeAddEventListener = peer.addEventListener.bind(peer);
				peer.addEventListener = (type, listener, options) => {
					if (type !== 'icecandidate') {
						return nativeAddEventListener(type, listener, options);
					}
					return nativeAddEventListener(type, (event) => {
						if (event && event.candidate && event.candidate.candidate && event.candidate.candidate.includes(' typ host ')) {
							listener.call(peer, new Event('icecandidate'));
							return;
						}
						listener.call(peer, event);
					}, options);
				};
				const nativeSetLocalDescription = peer.setLocalDescription.bind(peer);
				peer.setLocalDescription = async (description) => nativeSetLocalDescription(wrapDescription(description));
				const nativeCreateOffer = peer.createOffer.bind(peer);
				peer.createOffer = async (...args) => wrapDescription(await nativeCreateOffer(...args));
				const nativeCreateAnswer = peer.createAnswer.bind(peer);
				peer.createAnswer = async (...args) => wrapDescription(await nativeCreateAnswer(...args));
				return peer;
			};
			WrappedPeerConnection.prototype = NativePeerConnection.prototype;
			window.RTCPeerConnection = WrappedPeerConnection;
			window.webkitRTCPeerConnection = WrappedPeerConnection;
		};
		overrideNavigatorValue('platform', profile.platform);
		overrideNavigatorValue('userAgent', profile.userAgent);
		overrideNavigatorValue('vendor', profile.vendor);
		overrideNavigatorValue('language', profile.language);
		overrideNavigatorValue('languages', profile.languages);
		overrideNavigatorValue('hardwareConcurrency', profile.hardwareConcurrency);
		overrideNavigatorValue('deviceMemory', profile.deviceMemory);
		overrideNavigatorValue('maxTouchPoints', profile.maxTouchPoints);
		overrideNavigatorValue('userAgentData', createUserAgentData(profile.userAgentData));
		removeWebdriver();
		patchWebRTC();
	})();`, payload)

	if _, err := (proto.PageAddScriptToEvaluateOnNewDocument{
		Source:         script,
		RunImmediately: true,
	}).Call(page); err != nil {
		return fmt.Errorf("apply overrides: %w", err)
	}

	return nil
}

func buildUserAgentData(profile fingerprint.Profile) map[string]any {
	if profile.UserAgentMetadata == nil {
		return map[string]any{}
	}

	brands := make([]map[string]string, 0, len(profile.UserAgentMetadata.Brands))
	for _, brand := range profile.UserAgentMetadata.Brands {
		if brand == nil {
			continue
		}
		brands = append(brands, map[string]string{
			"brand":   brand.Brand,
			"version": brand.Version,
		})
	}

	fullVersionList := make([]map[string]string, 0, len(profile.UserAgentMetadata.FullVersionList))
	for _, brand := range profile.UserAgentMetadata.FullVersionList {
		if brand == nil {
			continue
		}
		fullVersionList = append(fullVersionList, map[string]string{
			"brand":   brand.Brand,
			"version": brand.Version,
		})
	}

	return map[string]any{
		"brands":          brands,
		"mobile":          profile.UserAgentMetadata.Mobile,
		"platform":        profile.UserAgentMetadata.Platform,
		"architecture":    profile.UserAgentMetadata.Architecture,
		"bitness":         profile.UserAgentMetadata.Bitness,
		"model":           profile.UserAgentMetadata.Model,
		"platformVersion": profile.UserAgentMetadata.PlatformVersion,
		"uaFullVersion":   profile.UserAgentMetadata.FullVersion,
		"fullVersionList": fullVersionList,
	}
}

func firstLanguage(languages []string) string {
	if len(languages) == 0 {
		return "en-US"
	}
	return languages[0]
}
