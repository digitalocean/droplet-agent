// SPDX-License-Identifier: Apache-2.0

package log

import (
	"fmt"
	"log"
	"os"
)

const (
	logFlags = log.LUTC | log.Ldate | log.Ltime | log.Lshortfile
)

var (
	logDebug  logger = log.New(os.Stdout, "DEBUG:", logFlags)
	logInfo   logger = log.New(os.Stdout, "INFO:", logFlags)
	logErr    logger = log.New(os.Stderr, "ERROR:", logFlags)
	debugMode        = false
)

type logger interface {
	Output(calldepth int, s string) error
}

// EnableDebug enables logging debug messages
func EnableDebug() {
	debugMode = true
}

// Debug prints a debug message. If syslog is enabled then LOG_NOTICE is used
func Debug(format string, params ...interface{}) {
	if !debugMode {
		return
	}
	if err := logDebug.Output(2, fmt.Sprintf(format, params...)); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR writing debug log output: %+v", err)
	}
}

// Info prints a message. If syslog is enabled then LOG_NOTICE is used
func Info(format string, params ...interface{}) {
	if err := logInfo.Output(2, fmt.Sprintf(format, params...)); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR writing info log output: %+v", err)
	}
}

// Error prints an error message. If syslog is enabled then LOG_ERR is used
func Error(format string, params ...interface{}) {
	if err := logErr.Output(2, fmt.Sprintf(format, params...)); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR writing error log output: %+v", err)
	}
}

// Fatal logs Error and exits 1
func Fatal(format string, params ...interface{}) {
	Error(format, params...)
	os.Exit(1)
}
