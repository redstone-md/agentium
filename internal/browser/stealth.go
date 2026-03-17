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
		"userAgent":               profile.UserAgent,
		"platform":                profile.Platform,
		"vendor":                  profile.Vendor,
		"webglVendor":             profile.WebGLVendor,
		"webglRenderer":           profile.WebGLRenderer,
		"preserveNativeNavigator": profile.PreserveNativeNavigator,
		"preserveNativeScreen":    profile.PreserveNativeScreen,
		"preserveNativeGraphics":  profile.PreserveNativeGraphics,
		"preserveNativeMedia":     profile.PreserveNativeMedia,
		"fingerprintSeed":         profile.FingerprintSeed,
		"canvasNoiseR":            profile.CanvasNoiseR,
		"canvasNoiseG":            profile.CanvasNoiseG,
		"canvasNoiseB":            profile.CanvasNoiseB,
		"audioNoise":              profile.AudioNoise,
		"viewportWidth":           profile.ViewportWidth,
		"viewportHeight":          profile.ViewportHeight,
		"outerWidth":              profile.OuterWidth,
		"outerHeight":             profile.OuterHeight,
		"windowScreenX":           profile.WindowScreenX,
		"windowScreenY":           profile.WindowScreenY,
		"screenWidth":             profile.ScreenWidth,
		"screenHeight":            profile.ScreenHeight,
		"availWidth":              profile.AvailWidth,
		"availHeight":             profile.AvailHeight,
		"deviceScaleFactor":       profile.DeviceScaleFactor,
		"colorDepth":              profile.ColorDepth,
		"pixelDepth":              profile.PixelDepth,
		"languages":               profile.Languages,
		"language":                firstLanguage(profile.Languages),
		"hardwareConcurrency":     profile.HardwareConcurrency,
		"deviceMemory":            profile.DeviceMemory,
		"maxTouchPoints":          profile.MaxTouchPoints,
		"speechVoices":            profile.SpeechVoices,
		"chromeKeys":              profile.ChromeKeys,
		"userAgentData":           buildUserAgentData(profile),
	})
	if err != nil {
		return fmt.Errorf("marshal stealth profile: %w", err)
	}

	script := fmt.Sprintf(`(() => {
		const profile = %s;
		const appVersion = profile.userAgent.startsWith('Mozilla/') ? profile.userAgent.slice('Mozilla/'.length) : profile.userAgent;
		const markNativeSource = (fn, source) => {
			return fn;
		};
		const markNative = (fn, name) => {
			return fn;
		};
		const createNativeGetter = (key, value) => {
			return function() {
				return value;
			};
		};
		const defineValue = (target, key, value) => {
			Object.defineProperty(target, key, { value, configurable: true, writable: true });
		};
		const define = (target, key, value) => {
			Object.defineProperty(target, key, { get: createNativeGetter(key, value), configurable: true });
		};
		const overrideNavigatorValue = (navigatorObject, key, value) => {
			if (!navigatorObject) {
				return;
			}
			const proto = Object.getPrototypeOf(navigatorObject);
			const descriptor = Object.getOwnPropertyDescriptor(proto, key);
			if (!descriptor || descriptor.configurable) {
				define(proto, key, value);
				return;
			}
			define(navigatorObject, key, value);
		};
		const removeWebdriver = (navigatorObject) => {
			if (!navigatorObject) {
				return;
			}
			try {
				delete navigatorObject.webdriver;
			} catch (_) {}

			const proto = Object.getPrototypeOf(navigatorObject);
			const descriptor = Object.getOwnPropertyDescriptor(proto, 'webdriver');
			if (descriptor && descriptor.configurable) {
				try {
					delete proto.webdriver;
				} catch (_) {}
				if (!('webdriver' in navigatorObject) && !('webdriver' in proto)) {
					return;
				}
			}
			define(navigatorObject, 'webdriver', false);
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
		const navigatorValues = () => ({
			appVersion,
			deviceMemory: profile.deviceMemory,
			hardwareConcurrency: profile.hardwareConcurrency,
			language: profile.language,
			languages: profile.languages,
			maxTouchPoints: profile.maxTouchPoints,
			platform: profile.platform,
			userAgent: profile.userAgent,
			userAgentData: createUserAgentData(profile.userAgentData),
			vendor: profile.vendor,
		});
		const applyNavigatorProfile = (navigatorObject) => {
			if (profile.preserveNativeNavigator) {
				removeWebdriver(navigatorObject);
				return;
			}
			for (const [key, value] of Object.entries(navigatorValues())) {
				overrideNavigatorValue(navigatorObject, key, value);
			}
			removeWebdriver(navigatorObject);
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
		const patchChromeObject = () => {
			const ensureChromeRoot = () => {
				if (window.chrome && typeof window.chrome === 'object') {
					return window.chrome;
				}
				const chromeObject = {};
				try {
					Object.defineProperty(window, 'chrome', {
						value: chromeObject,
						writable: true,
						enumerable: true,
						configurable: false
					});
					return chromeObject;
				} catch (_) {
					return window.chrome || chromeObject;
				}
			};
			const chromeObject = ensureChromeRoot();
			if (!chromeObject || typeof chromeObject !== 'object') {
				return;
			}
			if (!('app' in chromeObject)) {
				try {
					const installState = {
						DISABLED: 'disabled',
						INSTALLED: 'installed',
						NOT_INSTALLED: 'not_installed'
					};
					const runningState = {
						CANNOT_RUN: 'cannot_run',
						READY_TO_RUN: 'ready_to_run',
						RUNNING: 'running'
					};
					defineValue(chromeObject, 'app', {
						InstallState: installState,
						RunningState: runningState,
						get isInstalled() {
							return false;
						},
						getDetails() {
							if (arguments.length) {
								throw new TypeError('Error in invocation of app.getDetails()');
							}
							return null;
						},
						getIsInstalled() {
							if (arguments.length) {
								throw new TypeError('Error in invocation of app.getIsInstalled()');
							}
							return false;
						},
						runningState() {
							if (arguments.length) {
								throw new TypeError('Error in invocation of app.runningState()');
							}
							return 'cannot_run';
						}
					});
				} catch (_) {}
			}
			if (profile.preserveNativeNavigator) {
				return;
			}
			const allowedKeys = new Set((profile.chromeKeys || []).length ? profile.chromeKeys : ['loadTimes', 'csi', 'app']);
			for (const key of Object.getOwnPropertyNames(chromeObject)) {
				if (allowedKeys.has(key)) {
					continue;
				}
				try {
					delete chromeObject[key];
				} catch (_) {}
			}
			for (const key of ['loadTimes', 'csi', 'app']) {
				if (allowedKeys.has(key) && !(key in chromeObject)) {
					try {
						Object.defineProperty(chromeObject, key, {
							value: typeof chromeObject[key] === 'function' ? chromeObject[key] : (key === 'app' ? {} : markNative(function() {}, key)),
							configurable: true
						});
					} catch (_) {}
				}
			}
		};
		const patchWindowMetrics = () => {
			const desiredOuterWidth = profile.outerWidth || profile.viewportWidth || window.innerWidth || 0;
			const desiredOuterHeight = profile.outerHeight || profile.viewportHeight || window.innerHeight || 0;
			const desiredScreenX = typeof profile.windowScreenX === 'number' ? profile.windowScreenX : (window.screenX || window.screenLeft || 0);
			const desiredScreenY = typeof profile.windowScreenY === 'number' ? profile.windowScreenY : (window.screenY || window.screenTop || 0);
			if (desiredOuterWidth > 0) {
				define(window, 'outerWidth', desiredOuterWidth);
			}
			if (desiredOuterHeight > 0) {
				define(window, 'outerHeight', desiredOuterHeight);
			}
			define(window, 'screenX', desiredScreenX);
			define(window, 'screenY', desiredScreenY);
			define(window, 'screenLeft', desiredScreenX);
			define(window, 'screenTop', desiredScreenY);
		};
		const patchWebGL = () => {
			if (profile.preserveNativeGraphics) {
				return;
			}
			const patchContext = (ctor) => {
				if (!ctor || !ctor.prototype || typeof ctor.prototype.getParameter !== 'function') {
					return;
				}
				const nativeGetParameter = ctor.prototype.getParameter;
				Object.defineProperty(ctor.prototype, 'getParameter', {
					value: markNative(function getParameter(parameter) {
						if (parameter === 37445) {
							return profile.webglVendor;
						}
						if (parameter === 37446) {
							return profile.webglRenderer;
						}
						return nativeGetParameter.apply(this, arguments);
					}, 'getParameter'),
					configurable: true
				});
				const nativeReadPixels = ctor.prototype.readPixels;
				if (typeof nativeReadPixels === 'function') {
					Object.defineProperty(ctor.prototype, 'readPixels', {
						value: markNative(function readPixels() {
							const output = nativeReadPixels.apply(this, arguments);
							const pixels = arguments[6];
							if (pixels && typeof pixels.length === 'number' && pixels.length > 4) {
								const pixelIndex = Math.min(pixels.length - 4, Math.max(0, (profile.fingerprintSeed %% 16) * 4));
								pixels[pixelIndex] = Math.max(0, Math.min(255, pixels[pixelIndex] + profile.canvasNoiseR));
								pixels[pixelIndex + 1] = Math.max(0, Math.min(255, pixels[pixelIndex + 1] + profile.canvasNoiseG));
								pixels[pixelIndex + 2] = Math.max(0, Math.min(255, pixels[pixelIndex + 2] + profile.canvasNoiseB));
							}
							return output;
						}, 'readPixels'),
						configurable: true
					});
				}
			};
			patchContext(window.WebGLRenderingContext);
			patchContext(window.WebGL2RenderingContext);
		};
		const patchScreen = () => {
			if (profile.preserveNativeScreen) {
				return;
			}
			const overrideScreenValue = (key, value) => {
				try {
					Object.defineProperty(window.screen, key, { get: createNativeGetter(key, value), configurable: true });
				} catch (_) {}
				try {
					const screenProto = Object.getPrototypeOf(window.screen);
					Object.defineProperty(screenProto, key, { get: createNativeGetter(key, value), configurable: true });
				} catch (_) {}
			};
			overrideScreenValue('width', profile.screenWidth);
			overrideScreenValue('height', profile.screenHeight);
			overrideScreenValue('availWidth', profile.availWidth);
			overrideScreenValue('availHeight', profile.availHeight);
			overrideScreenValue('colorDepth', profile.colorDepth);
			overrideScreenValue('pixelDepth', profile.pixelDepth);
		};
		const patchIframes = () => {
			if (!document || typeof document.createElement !== 'function') {
				return;
			}
			const patchedIframes = new WeakSet();
			const addContentWindowProxy = (iframe) => {
				if (!iframe || patchedIframes.has(iframe)) {
					return;
				}
				if (iframe.contentWindow) {
					patchedIframes.add(iframe);
					return;
				}
				try {
					const proxy = new Proxy(window, {
						get(target, key) {
							if (key === 'self') {
								return proxy;
							}
							if (key === 'frameElement') {
								return iframe;
							}
							if (key === '0') {
								return undefined;
							}
							return Reflect.get(target, key);
						}
					});
					Object.defineProperty(iframe, 'contentWindow', {
						get() {
							return proxy;
						},
						set(value) {
							return value;
						},
						enumerable: true,
						configurable: false
					});
				} catch (_) {}
				patchedIframes.add(iframe);
			};
			const nativeCreateElement = document.createElement;
			const wrappedCreateElement = new Proxy(nativeCreateElement, {
				apply(target, thisArg, args) {
					const element = Reflect.apply(target, thisArg, args);
					const tagName = String((args && args[0]) || '').toLowerCase();
					if (tagName !== 'iframe' || !element) {
						return element;
					}
					const iframe = element;
					addContentWindowProxy(iframe);
					const nativeSetAttribute = iframe.setAttribute ? iframe.setAttribute.bind(iframe) : null;
					const originalSrcdoc = iframe.srcdoc;
					try {
						Object.defineProperty(iframe, 'srcdoc', {
							configurable: true,
							get() {
								return originalSrcdoc;
							},
							set(value) {
								addContentWindowProxy(iframe);
								try {
									Object.defineProperty(iframe, 'srcdoc', {
										configurable: false,
										writable: true,
										value: originalSrcdoc
									});
								} catch (_) {}
								iframe.srcdoc = value;
							}
						});
					} catch (_) {}
					if (nativeSetAttribute) {
						iframe.setAttribute = new Proxy(nativeSetAttribute, {
							apply(targetSetAttribute, thisArgSetAttribute, setArgs) {
								if (setArgs && String(setArgs[0] || '').toLowerCase() === 'srcdoc') {
									addContentWindowProxy(iframe);
								}
								return Reflect.apply(targetSetAttribute, thisArgSetAttribute, setArgs);
							}
						});
					}
					return iframe;
				}
			});
			try {
				Object.defineProperty(document, 'createElement', {
					value: wrappedCreateElement,
					configurable: true
				});
			} catch (_) {}
		};
		const patchCanvas = () => {
			if (profile.preserveNativeMedia) {
				return;
			}
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
						value: markNative(function getImageData() {
							return applyNoiseToImageData(nativeGetImageData.apply(this, arguments));
						}, 'getImageData'),
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
						value: markNative(function toDataURL() {
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
						}, 'toDataURL'),
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
			if (profile.preserveNativeMedia) {
				return;
			}
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
					value: markNative(function getChannelData() {
						return patchChannel(nativeGetChannelData.apply(this, arguments));
					}, 'getChannelData'),
					configurable: true
				});
			}
			const nativeCopyFromChannel = window.AudioBuffer && window.AudioBuffer.prototype.copyFromChannel;
			if (typeof nativeCopyFromChannel === 'function') {
				Object.defineProperty(window.AudioBuffer.prototype, 'copyFromChannel', {
					value: markNative(function copyFromChannel(destination) {
						const output = nativeCopyFromChannel.apply(this, arguments);
						if (destination && destination.length) {
							patchChannel(destination);
						}
						return output;
					}, 'copyFromChannel'),
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
					value: markNative(function startRendering() {
						const rendered = nativeStartRendering.apply(this, arguments);
						if (rendered && typeof rendered.then === 'function') {
							return rendered.then((buffer) => patchRenderedBuffer(buffer));
						}
						return patchRenderedBuffer(rendered);
					}, 'startRendering'),
					configurable: true
				});
			};
			patchOfflineContext(window.OfflineAudioContext);
			patchOfflineContext(window.webkitOfflineAudioContext);
		};
		const patchSpeechSynthesis = () => {
			if (profile.preserveNativeNavigator || profile.preserveNativeMedia) {
				return;
			}
			if (!window.speechSynthesis || typeof window.speechSynthesis.getVoices !== 'function') {
				return;
			}
			const nativeGetVoices = window.speechSynthesis.getVoices.bind(window.speechSynthesis);
			const languagePrefixes = (profile.languages || [])
				.map((language) => String(language || '').toLowerCase())
				.filter(Boolean)
				.map((language) => language.split('-')[0]);
			const cloneVoices = () => (profile.speechVoices || []).map((voice) => ({
				default: !!voice.default,
				lang: voice.lang || profile.language,
				localService: voice.localService !== false,
				name: voice.name || '',
				voiceURI: voice.voiceURI || voice.name || ''
			}));
			const needsVoiceFallback = (voices) => {
				if (!Array.isArray(voices) || voices.length === 0) {
					return true;
				}
				return !voices.some((voice) => {
					const lang = String(voice && voice.lang || '').toLowerCase();
					if (!lang) {
						return false;
					}
					if ((profile.languages || []).some((language) => language.toLowerCase() === lang)) {
						return true;
					}
					return languagePrefixes.some((prefix) => prefix && lang.startsWith(prefix));
				});
			};
			let syntheticVoices = null;
			const wrappedGetVoices = markNative(function getVoices() {
				const nativeVoices = nativeGetVoices();
				if (!needsVoiceFallback(nativeVoices)) {
					return nativeVoices;
				}
				if (!syntheticVoices) {
					syntheticVoices = cloneVoices();
				}
				return syntheticVoices;
			}, 'getVoices');
			Object.defineProperty(window.speechSynthesis, 'getVoices', {
				value: wrappedGetVoices,
				configurable: true
			});
			try {
				queueMicrotask(() => {
					const handler = window.speechSynthesis.onvoiceschanged;
					if (typeof handler === 'function') {
						handler.call(window.speechSynthesis, new Event('voiceschanged'));
					}
				});
			} catch (_) {}
		};
		applyNavigatorProfile(navigator);
		patchChromeObject();
		patchWindowMetrics();
		patchWebRTC();
		patchWebGL();
		patchScreen();
		patchCanvas();
		patchAudio();
		patchSpeechSynthesis();
		patchIframes();
	})();`, payload)

	if _, err := (proto.PageAddScriptToEvaluateOnNewDocument{
		Source:         script,
		RunImmediately: true,
	}).Call(page); err != nil {
		return fmt.Errorf("apply overrides: %w", err)
	}

	return nil
}
