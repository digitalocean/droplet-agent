// SPDX-License-Identifier: Apache-2.0

package mockutils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/golang/mock/gomock"
)

var _ gomock.Matcher = &HTTPRequestMatcher{}

// HTTPRequestMatcher is the object implementing the gomock Matcher interface to verify an http request
type HTTPRequestMatcher struct {
	ExpectedRequest *http.Request
}

// Matches returns true if the expected req matches the current http request
func (m *HTTPRequestMatcher) Matches(x interface{}) bool {
	actual, ok := x.(*http.Request)
	if !ok {
		return false
	}

	if m.ExpectedRequest.Method != actual.Method && m.ExpectedRequest.URL != actual.URL {
		return false
	}

	actualHeader := actual.Header
	for expectedKey, expectedValues := range m.ExpectedRequest.Header {
		if len(expectedValues) == 0 {
			return false
		}
		ev := expectedValues[0]

		actualValues, ok := actualHeader[expectedKey]
		if len(actualValues) == 0 {
			return false
		}
		av := actualValues[0]

		if ev != av || !ok {
			return false
		}
	}

	actualBodyReader := actual.Body
	expectedBodyReader := m.ExpectedRequest.Body

	actualBody, err := ioutil.ReadAll(actualBodyReader)
	if err != nil {
		return false
	}

	expectedBody, err := ioutil.ReadAll(expectedBodyReader)
	if err != nil {
		return false
	}

	return bytes.Equal(actualBody, expectedBody)
}

// String returns the expected http response from this matcher
func (m *HTTPRequestMatcher) String() string {
	return fmt.Sprintf("%v", m.ExpectedRequest)
}
