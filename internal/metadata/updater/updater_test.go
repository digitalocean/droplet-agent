// SPDX-License-Identifier: Apache-2.0

package updater

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/mockutils"
	"go.uber.org/mock/gomock"
)

func Test_agentInfoUpdaterImpl_Update(t *testing.T) {
	info := &metadata.Metadata{
		DOTTYStatus: metadata.RunningStatus,
		SSHInfo: &metadata.SSHInfo{
			Port:     256,
			HostKeys: nil,
		},
	}
	tests := []struct {
		name         string
		expectations func(client *MockhttpClient)
		wantErr      bool
	}{
		{
			"successful response",
			func(client *MockhttpClient) {
				reqMatcher := &mockutils.HTTPRequestMatcher{
					ExpectedRequest: newRequest(t, []byte("{\"dotty_status\":\"running\",\"ssh_info\":{\"port\":256}}")),
				}

				client.EXPECT().Do(reqMatcher).Return(&http.Response{StatusCode: 202}, nil)
			},
			false,
		},
		{
			"unsuccessful response code",
			func(client *MockhttpClient) {
				reqMatcher := &mockutils.HTTPRequestMatcher{
					ExpectedRequest: newRequest(t, []byte("{\"dotty_status\":\"running\",\"ssh_info\":{\"port\":256}}")),
				}

				client.EXPECT().Do(reqMatcher).Return(&http.Response{StatusCode: 404}, nil)
			},
			true,
		},
		{
			"error from http client",
			func(client *MockhttpClient) {
				reqMatcher := &mockutils.HTTPRequestMatcher{
					ExpectedRequest: newRequest(t, []byte("{\"dotty_status\":\"running\",\"ssh_info\":{\"port\":256}}")),
				}
				client.EXPECT().Do(reqMatcher).Return(nil, errors.New("something went wrong"))
			},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			client := NewMockhttpClient(ctrl)
			tt.expectations(client)
			m := &agentInfoUpdaterImpl{
				client: client,
			}
			if err := m.Update(info); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newRequest(t *testing.T, body []byte) *http.Request {
	req, err := http.NewRequest(
		http.MethodPatch,
		fmt.Sprintf("%s/v1.json", metadata.BaseURL),
		bytes.NewBuffer(body),
	)

	if err != nil {
		t.Fatalf("could not create http request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("User-Agent", "Droplet-Agent/1.0.1")

	return req
}
