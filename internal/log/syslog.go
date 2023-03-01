// SPDX-License-Identifier: Apache-2.0

package log

import (
	"fmt"
	"log"
	"log/syslog"
	"sync"
)

const (
	syslogFlags = log.Lshortfile
)

var once sync.Once

// UseSysLog initializes logging to syslog
func UseSysLog() error {
	var err error
	once.Do(func() {
		dl, e := syslog.NewLogger(syslog.LOG_DEBUG, syslogFlags)
		if e != nil {
			err = fmt.Errorf("failed to use syslog: %w", e)
			return
		}

		il, e := syslog.NewLogger(syslog.LOG_INFO, syslogFlags)
		if e != nil {
			err = fmt.Errorf("failed to use syslog: %w", e)
			return
		}

		el, e := syslog.NewLogger(syslog.LOG_ERR, syslogFlags)
		if e != nil {
			err = fmt.Errorf("failed to use syslog: %w", e)
			return
		}
		logDebug = dl
		logInfo = il
		logErr = el
	})
	return err
}
