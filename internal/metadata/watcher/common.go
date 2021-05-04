package watcher

import "errors"

const (
	webAddr                    = ":303"
	maxFetchPerSecond          = 5
	maxShutdownWaitTimeSeconds = 5
)

//Possible Errors
var (
	ErrFetchMetadataFailed  = errors.New("failed to fetch rmetadata")
	ErrNoRegisteredActioner = errors.New("no registered actioners")
)
