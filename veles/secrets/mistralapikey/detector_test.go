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
	}{{
		name:  "valid_key",
		input: validAPIKey,
		want: []veles.Secret{
			APIKey{Key: validAPIKey},
		},
	}, {
		name:  "valid_key_in_config",
		input: "MISTRAL_API_KEY=" + validAPIKey,
		want: []veles.Secret{
			APIKey{Key: validAPIKey},
		},
	}, {
		name:  "invalid_key_too_short",
		input: "abcdefghijklmnopqrstuvwxyz12345",
		want:  nil,
	}, {
		name:  "invalid_key_too_long",
		input: "abcdefghijklmnopqrstuvwxyz1234567",
		want:  nil,
	}}

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
