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
	"regexp"

	"github.com/google/osv-scalibr/veles"
)

// maxTokenLength is the maximum size of a Mistral API key.
const maxTokenLength = 32

// envVarRe matches environment variable style assignments.
// Examples: MISTRAL_API_KEY=token, MISTRAL_TOKEN="token"
var envVarRe = regexp.MustCompile(
	`(?i)(MISTRAL_API_KEY|MISTRAL_KEY|MISTRAL_TOKEN|MISTRAL_API_TOKEN)\s*=\s*['"]?([a-zA-Z0-9]{32})\b['"]?`,
)

// jsonRe matches JSON key-value pairs.
// Examples: "mistral_api_key": "token", "mistral_token": "token"
var jsonRe = regexp.MustCompile(
	`(?i)"(mistral_api_key|mistral_key|mistral_token|mistral_api_token)"\s*:\s*"([a-zA-Z0-9]{32})\b"`,
)

// yamlRe matches YAML key-value pairs.
// Examples: mistral_api_key: token, mistral_token: "token"
var yamlRe = regexp.MustCompile(
	`(?i)(mistral_api_key|mistral_key|mistral_token|mistral_api_token)\s*:\s*['"]?([a-zA-Z0-9]{32})\b['"]?`,
)

// Detector finds instances of Mistral API keys inside a chunk of text.
type Detector struct{}

// NewDetector returns a new Detector that matches Mistral API keys.
func NewDetector() veles.Detector {
	return &Detector{}
}

// MaxSecretLen returns the maximum length of the token.
func (d *Detector) MaxSecretLen() uint32 {
	return maxTokenLength
}

// Detect finds candidate tokens that match the regex patterns.
func (d *Detector) Detect(data []byte) ([]veles.Secret, []int) {
	var secrets []veles.Secret
	var positions []int
	seenTokens := make(map[string]bool)

	// Define all regex patterns to check
	patterns := []*regexp.Regexp{
		envVarRe,
		jsonRe,
		yamlRe,
	}

	// Check each pattern
	for _, pattern := range patterns {
		matches := pattern.FindAllSubmatchIndex(data, -1)
		for _, match := range matches {
			// match[4] and match[5] contain the start and end indices of the second capture group (the token)
			if len(match) >= 6 {
				tokenStart := match[4]
				tokenEnd := match[5]
				token := string(data[tokenStart:tokenEnd])

				if !seenTokens[token] {
					secrets = append(secrets, APIKey{Key: token})
					positions = append(positions, tokenStart)
					seenTokens[token] = true
				}
			}
		}
	}
	return secrets, positions
}
