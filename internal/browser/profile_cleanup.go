package browser

import (
	"errors"
	"os"
	"time"
)

const (
	profileCleanupAttempts = 20
	profileCleanupDelay    = 250 * time.Millisecond
)

func removeProfileDir(profileDir string) error {
	var lastErr error

	for attempt := 0; attempt < profileCleanupAttempts; attempt++ {
		if err := os.RemoveAll(profileDir); err == nil || errors.Is(err, os.ErrNotExist) {
			return nil
		} else {
			lastErr = err
		}

		time.Sleep(profileCleanupDelay)
	}

	return lastErr
}
