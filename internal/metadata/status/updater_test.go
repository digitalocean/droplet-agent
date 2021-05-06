// SPDX-License-Identifier: Apache-2.0

package status

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/golang/mock/gomock"
)

func Test_statusUpdaterImpl_Update(t *testing.T) {
	tests := []struct {
		name         string
		status       metadata.AgentStatus
		expectations func(client *MockhttpClient)
		wantErr      bool
	}{
		{
			"successful response",
			metadata.RunningStatus,
			func(client *MockhttpClient) {
				reqMatcher := &HTTPRequestMatcher{
					ExpectedRequest: newRequest(t, []byte("{\"dotty_status\":\"running\"}")),
				}

				client.EXPECT().Do(reqMatcher).Return(&http.Response{StatusCode: 202}, nil)
			},
			false,
		},
		{
			"unsuccessful response code",
			metadata.RunningStatus,
			func(client *MockhttpClient) {
				reqMatcher := &HTTPRequestMatcher{
					ExpectedRequest: newRequest(t, []byte("{\"dotty_status\":\"running\"}")),
				}

				client.EXPECT().Do(reqMatcher).Return(&http.Response{StatusCode: 404}, nil)
			},
			true,
		},
		{
			"error from http client",
			metadata.RunningStatus,
			func(client *MockhttpClient) {
				reqMatcher := &HTTPRequestMatcher{
					ExpectedRequest: newRequest(t, []byte("{\"dotty_status\":\"running\"}")),
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
			m := &statusUpdaterImpl{
				http: client,
			}
			if err := m.Update(tt.status); (err != nil) != tt.wantErr {
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
