// SPDX-License-Identifier: Apache-2.0

package log

import (
	"sync"
)

var muteOnce sync.Once

// Mute mutes all logs
func Mute() {
	muteOnce.Do(func() {
		logDebug = &muteLogger{}
		logInfo = &muteLogger{}
		logErr = &muteLogger{}
	})
}

type muteLogger struct{}

func (*muteLogger) Output(calldepth int, s string) error {
	return nil
}
