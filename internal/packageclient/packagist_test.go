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

package packageclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPackagistGetSourceRepository(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wantErr error
		name    string
		pkg     string
		status  int
		body    string
		want    string
	}{
		{
			name:   "tagged releases",
			pkg:    "composer/composer",
			status: http.StatusOK,
			body: `{"packages":{"composer/composer":[
				{"version":"2.9.0","source":{"url":"https://github.com/composer/composer.git","type":"git"}},
				{"version":"2.8.0"}
			]},"minified":"composer/2.0"}`,
			want: "https://github.com/composer/composer.git",
		},
		{
			name:    "not found",
			pkg:     "foo/bar",
			status:  http.StatusNotFound,
			wantErr: ErrPackageNotFoundOnPackagist,
		},
		{
			name:    "no tagged releases",
			pkg:     "foo/bar",
			status:  http.StatusOK,
			body:    `{"packages":{"foo/bar":[]},"minified":"composer/2.0"}`,
			wantErr: ErrPackageHasNoTaggedReleases,
		},
		{
			name:    "server error",
			pkg:     "foo/bar",
			status:  http.StatusBadGateway,
			wantErr: ErrPackagistAPI,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/p2/"+tt.pkg+".json" {
					t.Errorf("unexpected request path %q", r.URL.Path)
				}
				w.WriteHeader(tt.status)
				w.Write([]byte(tt.body)) //nolint:errcheck
			}))
			defer server.Close()

			client := packagistClient{client: server.Client(), baseURL: server.URL + "/p2/"}
			got, err := client.GetSourceRepository(context.Background(), tt.pkg)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetSourceRepository() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("GetSourceRepository() = %q, want %q", got, tt.want)
			}
		})
	}
}
