// Copyright 2026 OpenSSF Scorecard Authors
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

package githubrepo

import "testing"

func TestHasExportIgnoreAttribute(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "empty",
			content: "",
			want:    false,
		},
		{
			name:    "only text attributes",
			content: "* text=auto eol=lf\n*.php text eol=lf diff=php\n",
			want:    false,
		},
		{
			name:    "export-ignore directory",
			content: "* text=auto\n\n# Exclude non-essential files from dist\n/.github/ export-ignore\n",
			want:    true,
		},
		{
			name:    "export-ignore with other attributes",
			content: "/tests linguist-vendored export-ignore\n",
			want:    true,
		},
		{
			name:    "explicit true",
			content: "/tests export-ignore=true\n",
			want:    true,
		},
		{
			name:    "commented out",
			content: "# /.github/ export-ignore\n",
			want:    false,
		},
		{
			name:    "unset and unspecified",
			content: "/phpstan/rules.neon -export-ignore\n/foo !export-ignore\n",
			want:    false,
		},
		{
			name:    "pattern named export-ignore",
			content: "export-ignore text\n",
			want:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasExportIgnoreAttribute([]byte(tt.content)); got != tt.want {
				t.Errorf("hasExportIgnoreAttribute() = %v, want %v", got, tt.want)
			}
		})
	}
}
