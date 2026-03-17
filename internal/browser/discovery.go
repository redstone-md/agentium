package browser

import "os"

func resolveChromeBinary(explicitPath string) string {
	if explicitPath != "" {
		return explicitPath
	}

	for _, candidate := range preferredChromeBinaries() {
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

func preferredChromeBinaries() []string {
	return []string{
		`C:\Program Files\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`,
		`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`,
		`/usr/bin/google-chrome`,
		`/usr/bin/chromium`,
		`/usr/bin/chromium-browser`,
		`/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`,
		`/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge`,
	}
}
