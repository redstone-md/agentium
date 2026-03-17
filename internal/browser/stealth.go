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
		"webglVendor":         profile.WebGLVendor,
		"webglRenderer":       profile.WebGLRenderer,
		"fingerprintSeed":     profile.FingerprintSeed,
		"canvasNoiseR":        profile.CanvasNoiseR,
		"canvasNoiseG":        profile.CanvasNoiseG,
		"canvasNoiseB":        profile.CanvasNoiseB,
		"audioNoise":          profile.AudioNoise,
		"viewportWidth":       profile.ViewportWidth,
		"viewportHeight":      profile.ViewportHeight,
		"screenWidth":         profile.ScreenWidth,
		"screenHeight":        profile.ScreenHeight,
		"availWidth":          profile.AvailWidth,
		"availHeight":         profile.AvailHeight,
		"deviceScaleFactor":   profile.DeviceScaleFactor,
		"colorDepth":          profile.ColorDepth,
		"pixelDepth":          profile.PixelDepth,
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
		const patchWebGL = () => {
			const patchContext = (ctor) => {
				if (!ctor || !ctor.prototype || typeof ctor.prototype.getParameter !== 'function') {
					return;
				}
				const nativeGetParameter = ctor.prototype.getParameter;
				Object.defineProperty(ctor.prototype, 'getParameter', {
					value: function(parameter) {
						if (parameter === 37445) {
							return profile.webglVendor;
						}
						if (parameter === 37446) {
							return profile.webglRenderer;
						}
						return nativeGetParameter.apply(this, arguments);
					},
					configurable: true
				});
				const nativeReadPixels = ctor.prototype.readPixels;
				if (typeof nativeReadPixels === 'function') {
					Object.defineProperty(ctor.prototype, 'readPixels', {
						value: function() {
							const output = nativeReadPixels.apply(this, arguments);
							const pixels = arguments[6];
							if (pixels && typeof pixels.length === 'number' && pixels.length > 4) {
								const pixelIndex = Math.min(pixels.length - 4, Math.max(0, (profile.fingerprintSeed %% 16) * 4));
								pixels[pixelIndex] = Math.max(0, Math.min(255, pixels[pixelIndex] + profile.canvasNoiseR));
								pixels[pixelIndex + 1] = Math.max(0, Math.min(255, pixels[pixelIndex + 1] + profile.canvasNoiseG));
								pixels[pixelIndex + 2] = Math.max(0, Math.min(255, pixels[pixelIndex + 2] + profile.canvasNoiseB));
							}
							return output;
						},
						configurable: true
					});
				}
			};
			patchContext(window.WebGLRenderingContext);
			patchContext(window.WebGL2RenderingContext);
		};
		const patchScreen = () => {
			const overrideScreenValue = (key, value) => {
				try {
					Object.defineProperty(window.screen, key, { get: () => value, configurable: true });
				} catch (_) {}
				try {
					const screenProto = Object.getPrototypeOf(window.screen);
					Object.defineProperty(screenProto, key, { get: () => value, configurable: true });
				} catch (_) {}
			};
			overrideScreenValue('width', profile.screenWidth);
			overrideScreenValue('height', profile.screenHeight);
			overrideScreenValue('availWidth', profile.availWidth);
			overrideScreenValue('availHeight', profile.availHeight);
			overrideScreenValue('colorDepth', profile.colorDepth);
			overrideScreenValue('pixelDepth', profile.pixelDepth);
			define(window, 'innerWidth', profile.viewportWidth || profile.screenWidth);
			define(window, 'innerHeight', profile.viewportHeight || profile.availHeight);
			define(window, 'outerWidth', profile.viewportWidth || profile.screenWidth);
			define(window, 'outerHeight', profile.viewportHeight || profile.availHeight);
			define(window, 'devicePixelRatio', profile.deviceScaleFactor);
		};
		const patchCanvas = () => {
			const patchedImageData = new WeakSet();
			const applyNoiseToImageData = (imageData) => {
				if (!imageData || !imageData.data || imageData.data.length < 4 || patchedImageData.has(imageData)) {
					return imageData;
				}
				const index = (profile.canvasNoiseR + profile.canvasNoiseG + profile.canvasNoiseB) %% 16;
				const offset = Math.abs(index) * 4;
				const maxIndex = imageData.data.length - 4;
				const pixelIndex = Math.min(offset, maxIndex);
				imageData.data[pixelIndex] = Math.max(0, Math.min(255, imageData.data[pixelIndex] + profile.canvasNoiseR));
				imageData.data[pixelIndex + 1] = Math.max(0, Math.min(255, imageData.data[pixelIndex + 1] + profile.canvasNoiseG));
				imageData.data[pixelIndex + 2] = Math.max(0, Math.min(255, imageData.data[pixelIndex + 2] + profile.canvasNoiseB));
				patchedImageData.add(imageData);
				return imageData;
			};
			const patchContext = (ctor) => {
				if (!ctor || !ctor.prototype) {
					return;
				}
				const nativeGetImageData = ctor.prototype.getImageData;
				if (typeof nativeGetImageData === 'function') {
					Object.defineProperty(ctor.prototype, 'getImageData', {
						value: function() {
							return applyNoiseToImageData(nativeGetImageData.apply(this, arguments));
						},
						configurable: true
					});
				}
			};
			patchContext(window.CanvasRenderingContext2D);
			if (window.OffscreenCanvasRenderingContext2D) {
				patchContext(window.OffscreenCanvasRenderingContext2D);
			}
			const patchCanvasExport = (ctor) => {
				if (!ctor || !ctor.prototype) {
					return;
				}
				const nativeToDataURL = ctor.prototype.toDataURL;
				if (typeof nativeToDataURL === 'function') {
					Object.defineProperty(ctor.prototype, 'toDataURL', {
						value: function() {
							try {
								const width = Math.max(1, this.width || 1);
								const height = Math.max(1, this.height || 1);
								const clone = document.createElement('canvas');
								clone.width = width;
								clone.height = height;
								const cloneCtx = clone.getContext('2d');
								if (cloneCtx && typeof cloneCtx.drawImage === 'function' && typeof cloneCtx.getImageData === 'function' && typeof cloneCtx.putImageData === 'function') {
									cloneCtx.drawImage(this, 0, 0);
									const imageData = cloneCtx.getImageData(0, 0, Math.min(width, 32), Math.min(height, 32));
									cloneCtx.putImageData(applyNoiseToImageData(imageData), 0, 0);
									return nativeToDataURL.apply(clone, arguments);
								}
							} catch (_) {}
							return nativeToDataURL.apply(this, arguments);
						},
						configurable: true
					});
				}
			};
			patchCanvasExport(window.HTMLCanvasElement);
			if (window.OffscreenCanvas) {
				patchCanvasExport(window.OffscreenCanvas);
			}
		};
		const patchAudio = () => {
			const patchedChannels = new WeakSet();
			const patchChannel = (channel) => {
				if (!channel || !channel.length || patchedChannels.has(channel)) {
					return channel;
				}
				const index = Math.min(channel.length - 1, Math.max(0, profile.fingerprintSeed %% 32));
				channel[index] = channel[index] + profile.audioNoise;
				patchedChannels.add(channel);
				return channel;
			};
			const nativeGetChannelData = window.AudioBuffer && window.AudioBuffer.prototype.getChannelData;
			if (typeof nativeGetChannelData === 'function') {
				Object.defineProperty(window.AudioBuffer.prototype, 'getChannelData', {
					value: function() {
						return patchChannel(nativeGetChannelData.apply(this, arguments));
					},
					configurable: true
				});
			}
			const nativeCopyFromChannel = window.AudioBuffer && window.AudioBuffer.prototype.copyFromChannel;
			if (typeof nativeCopyFromChannel === 'function') {
				Object.defineProperty(window.AudioBuffer.prototype, 'copyFromChannel', {
					value: function(destination) {
						const output = nativeCopyFromChannel.apply(this, arguments);
						if (destination && destination.length) {
							patchChannel(destination);
						}
						return output;
					},
					configurable: true
				});
			}
			const patchRenderedBuffer = (buffer) => {
				if (!buffer || typeof buffer.getChannelData !== 'function' || typeof buffer.numberOfChannels !== 'number') {
					return buffer;
				}
				for (let channel = 0; channel < buffer.numberOfChannels; channel += 1) {
					try {
						patchChannel(buffer.getChannelData(channel));
					} catch (_) {}
				}
				return buffer;
			};
			const patchOfflineContext = (ctor) => {
				if (!ctor || !ctor.prototype || typeof ctor.prototype.startRendering !== 'function') {
					return;
				}
				const nativeStartRendering = ctor.prototype.startRendering;
				Object.defineProperty(ctor.prototype, 'startRendering', {
					value: function() {
						const rendered = nativeStartRendering.apply(this, arguments);
						if (rendered && typeof rendered.then === 'function') {
							return rendered.then((buffer) => patchRenderedBuffer(buffer));
						}
						return patchRenderedBuffer(rendered);
					},
					configurable: true
				});
			};
			patchOfflineContext(window.OfflineAudioContext);
			patchOfflineContext(window.webkitOfflineAudioContext);
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
		patchWebGL();
		patchScreen();
		patchCanvas();
		patchAudio();
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
