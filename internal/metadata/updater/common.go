package updater

import "errors"

var (
	// ErrUpdateMetadataFailed is returned when failed to update metadata
	ErrUpdateMetadataFailed = errors.New("failed to update status")
)
