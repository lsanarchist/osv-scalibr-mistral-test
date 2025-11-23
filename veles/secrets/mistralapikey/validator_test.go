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

package mistralapikey_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/osv-scalibr/veles"
	"github.com/google/osv-scalibr/veles/secrets/mistralapikey"
)

const (
	validAPIKey = "abcdefghijklmnopqrstuvwxyz123456"
)

// mockMistralServer creates a mock Mistral API server for testing keys
func mockMistralServer(t *testing.T, expectedKey string, statusCode int) *httptest.Server {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,
		r *http.Request) {
		modelsEndpoint := mistralapikey.ModelsEndpoint
		// Check if it's a GET request to the models endpoint
		if r.Method != http.MethodGet || r.URL.Path != modelsEndpoint {
			t.Errorf("unexpected request: %s %s, expected: GET %s",
				r.Method, r.URL.Path, modelsEndpoint)
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// Check Authorization header (Bearer token format)
		expectedAuth := "Bearer " + expectedKey
		if r.Header.Get("Authorization") != expectedAuth {
			t.Errorf("expected Authorization: %s, got: %s",
				expectedAuth, r.Header.Get("Authorization"))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		w.WriteHeader(statusCode)
	}))

	return server
}

func TestValidator(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		want       veles.ValidationStatus
	}{
		{
			name:       "valid_key",
			statusCode: http.StatusOK,
			want:       veles.ValidationValid,
		},
		{
			name:       "invalid_key",
			statusCode: http.StatusUnauthorized,
			want:       veles.ValidationInvalid,
		},
		{
			name:       "server_error",
			statusCode: http.StatusInternalServerError,
			want:       veles.ValidationFailed,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := mockMistralServer(t, validAPIKey, tc.statusCode)
			defer server.Close()

			validator := mistralapikey.NewValidator()
			validator.Endpoint = server.URL + mistralapikey.ModelsEndpoint

			status, err := validator.Validate(context.Background(), mistralapikey.APIKey{Key: validAPIKey})
			if err != nil && tc.want != veles.ValidationFailed {
				t.Fatalf("Validate returned unexpected error: %v", err)
			}

			if status != tc.want {
				t.Errorf("Validate returned status %v, want %v", status, tc.want)
			}
		})
	}
}
