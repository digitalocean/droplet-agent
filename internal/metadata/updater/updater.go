// SPDX-License-Identifier: Apache-2.0

package updater

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/digitalocean/droplet-agent/internal/config"
	"github.com/digitalocean/droplet-agent/internal/metadata"
)

// NewAgentInfoUpdater creates a new agent info updater
func NewAgentInfoUpdater() AgentInfoUpdater {
	return &agentInfoUpdaterImpl{client: &http.Client{}}
}

type agentInfoUpdaterImpl struct {
	client httpClient
}

type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func (u *agentInfoUpdaterImpl) Update(md *metadata.Metadata) error {
	metadataURL := fmt.Sprintf("%s/v1.json", metadata.BaseURL)

	body, err := json.Marshal(md)
	if err != nil {
		return fmt.Errorf("%w:%v", ErrUpdateMetadataFailed, err)
	}

	req, err := http.NewRequest(http.MethodPatch, metadataURL, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("%w:%v", ErrUpdateMetadataFailed, err)
	}

	req.Header.Set("User-Agent", config.UserAgent)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp, err := u.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w:%v", ErrUpdateMetadataFailed, err)
	}
	defer func() {
		if resp.Body != nil {
			resp.Body.Close()
		}
	}()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	if !success {
		return fmt.Errorf("%w: metadata returned status code: %d", ErrUpdateMetadataFailed, resp.StatusCode)
	}
	return nil
}
