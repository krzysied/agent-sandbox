// Copyright 2026 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package client

import (
	"fmt"
	"net/http"
)

// NoRetryTransport wraps an existing http.RoundTripper to intercept and
// modify specific HTTP responses before client-go processes them.
type NoRetryTransport struct {
	Transport http.RoundTripper
}

// RoundTrip executes a single HTTP transaction, returning a Response for the provided Request.
func (t *NoRetryTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 1. Execute the standard request using the underlying transport
	resp, err := t.Transport.RoundTrip(req)

	// 2. Check if the API server rejected the request due to rate limiting (HTTP 429)
	if err == nil && resp != nil && resp.StatusCode == http.StatusTooManyRequests {

		// 3. CRITICAL: Close the response body to prevent resource/memory leaks,
		// because we are discarding the actual HTTP response object.
		resp.Body.Close()

		// 4. Return a hard error instead of the HTTP response.
		// client-go's inline retry logic needs a valid HTTP response to check the
		// StatusCode and read the "Retry-After" header. By returning an error here,
		// client-go immediately aborts and passes this error straight back up the chain.
		return nil, fmt.Errorf("intercepted HTTP 429 Too Many Requests to prevent client-go inline retries")
	}

	// 5. For all other responses (200 OK, 404 Not Found, 409 Conflict, etc.),
	// pass them through completely unmodified.
	return resp, err
}
