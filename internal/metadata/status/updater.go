// SPDX-License-Identifier: Apache-2.0

package status

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/digitalocean/dotty-agent/internal/metadata"
)

//Possible Errors
var (
	ErrUpdateMetadataFailed = errors.New("failed to update status")
)

// Updater updates the metadata for the DOTTY agent status of the droplet
type Updater interface {
	Update(status metadata.AgentStatus) error
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewStatusUpdater creates a new status updater using the passed in http client
func NewStatusUpdater() Updater {
	return &statusUpdaterImpl{
		http: &http.Client{},
	}
}

type statusUpdaterImpl struct {
	http httpClient
}

func (m *statusUpdaterImpl) Update(status metadata.AgentStatus) error {
	metadataURL := fmt.Sprintf("%s/v1.json", metadata.BaseURL)

	body, err := json.Marshal(&metadata.Metadata{DOTTYStatus: status})
	if err != nil {
		return fmt.Errorf("%w:%v", ErrUpdateMetadataFailed, err)
	}

	req, err := http.NewRequest(http.MethodPatch, metadataURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("%w:%v", ErrUpdateMetadataFailed, err)
	}

	req.Header.Set("User-Agent", "DoTTY/1.0.1")
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := m.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w:%v", ErrUpdateMetadataFailed, err)
	}

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		return fmt.Errorf("%w: metadata returned status code: %d", ErrUpdateMetadataFailed, resp.StatusCode)
	}
	return nil
}
