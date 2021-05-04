package watcher

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/digitalocean/droplet-agent/internal/metadata"
)

type metadataFetcher interface {
	fetchMetadata() (*metadata.Metadata, error)
}

func newMetadataFetcher() metadataFetcher {
	return &metadataFetcherImpl{}
}

type metadataFetcherImpl struct {
}

func (m *metadataFetcherImpl) fetchMetadata() (*metadata.Metadata, error) {
	metadataURL := fmt.Sprintf("%s/v1.json", metadata.BaseURL)
	metaResp, err := http.Get(metadataURL) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("%w:%v", ErrFetchMetadataFailed, err)
	}
	defer func() {
		_ = metaResp.Body.Close()
	}()

	metadataRaw, err := ioutil.ReadAll(metaResp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w:%v", ErrFetchMetadataFailed, err)
	}
	ret := &metadata.Metadata{}
	if err := json.Unmarshal(metadataRaw, ret); err != nil {
		return nil, fmt.Errorf("%w:%v", ErrFetchMetadataFailed, err)
	}
	return ret, nil
}
