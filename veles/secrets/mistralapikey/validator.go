// Copyright 2025 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mistralapikey

import (
	"net/http"
	"time"

	sv "github.com/google/osv-scalibr/veles/secrets/common/simplevalidate"
)

const (
	// Mistral API base URL.
	mistralAPIBaseURL = "https://api.mistral.ai"
	// Timeout for API validation requests.
	validationTimeout = 10 * time.Second
	// ModelsEndpoint is the Mistral models API endpoint.
	ModelsEndpoint = "/v1/models"
)

// NewValidator creates a new Validator that validates API keys by
// making a test request to the Mistral API /v1/models endpoint.
func NewValidator() *sv.Validator[APIKey] {
	return &sv.Validator[APIKey]{
		Endpoint:   mistralAPIBaseURL + ModelsEndpoint,
		HTTPMethod: http.MethodGet,
		HTTPHeaders: func(k APIKey) map[string]string {
			return map[string]string{"Authorization": "Bearer " + k.Key}
		},
		ValidResponseCodes:   []int{http.StatusOK, http.StatusTooManyRequests},
		InvalidResponseCodes: []int{http.StatusUnauthorized},
		HTTPC: &http.Client{
			Timeout: validationTimeout,
		},
	}
}
