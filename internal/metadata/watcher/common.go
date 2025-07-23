// SPDX-License-Identifier: Apache-2.0

package watcher

import "errors"

const (
	webAddr                    = ":303"
	maxFetchPerSecond          = 5
	maxShutdownWaitTimeSeconds = 5
)

// Possible Errors
var (
	ErrFetchMetadataFailed  = errors.New("failed to fetch rmetadata")
	ErrNoRegisteredActioner = errors.New("no registered actioners")
)

// Conf contains configurations for a watcher
type Conf struct {
	SSHPort uint16
}
