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
	"bytes"
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/osv-scalibr/veles"
)

const (
	validAPIKey = "abcdefghijklmnopqrstuvwxyz123456"
)

func TestDetector(t *testing.T) {
	engine, err := veles.NewDetectionEngine([]veles.Detector{NewDetector()})
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name  string
		input string
		want  []veles.Secret
	}{
		{
			name:  "MISTRAL_API_KEY env var",
			input: "MISTRAL_API_KEY=" + validAPIKey,
			want: []veles.Secret{
				APIKey{Key: validAPIKey},
			},
		},
		{
			name:  "MISTRAL_TOKEN env var with quotes",
			input: `MISTRAL_TOKEN="` + validAPIKey + `"`,
			want: []veles.Secret{
				APIKey{Key: validAPIKey},
			},
		},
		{
			name:  "JSON format",
			input: `{"mistral_api_key": "` + validAPIKey + `"}`,
			want: []veles.Secret{
				APIKey{Key: validAPIKey},
			},
		},
		{
			name:  "YAML format",
			input: `mistral_token: ` + validAPIKey,
			want: []veles.Secret{
				APIKey{Key: validAPIKey},
			},
		},
		{
			name:  "YAML format with quotes",
			input: `mistral_api_token: "` + validAPIKey + `"`,
			want: []veles.Secret{
				APIKey{Key: validAPIKey},
			},
		},
		{
			name:  "Case insensitive env var",
			input: `Mistral_Api_Key=` + validAPIKey,
			want: []veles.Secret{
				APIKey{Key: validAPIKey},
			},
		},
		{
			name:  "Random 32-char string without context",
			input: validAPIKey,
			want:  nil,
		},
		{
			name:  "Wrong key name",
			input: `OTHER_API_KEY=` + validAPIKey,
			want:  nil,
		},
		{
			name:  "Key too short",
			input: `MISTRAL_API_KEY=` + validAPIKey[:31],
			want:  nil,
		},
		{
			name:  "Key too long",
			input: `MISTRAL_API_KEY=` + validAPIKey + "a",
			want:  nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := engine.Detect(context.Background(), bytes.NewReader([]byte(tc.input)))
			if err != nil {
				t.Fatalf("Detect(%q) returned unexpected error: %v", tc.input, err)
			}
			if diff := cmp.Diff(tc.want, got, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Detect(%q) returned unexpected diff (-want +got):\n%s", tc.input, diff)
			}
		})
	}
}
