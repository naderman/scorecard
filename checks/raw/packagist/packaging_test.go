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

package packagist

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/ossf/scorecard/v5/checker"
	mockrepo "github.com/ossf/scorecard/v5/clients/mockclients"
	"github.com/ossf/scorecard/v5/internal/packageclient"
)

type fakePackagist struct {
	err       error
	sourceURL string
	queried   string
}

func (f *fakePackagist) GetSourceRepository(_ context.Context, name string) (string, error) {
	f.queried = name
	return f.sourceURL, f.err
}

func TestPackaging(t *testing.T) {
	t.Parallel()
	//nolint:govet
	tests := []struct {
		name         string
		composer     string // empty means no composer.json
		sourceURL    string
		packagistErr error
		wantQuery    string
		wantPackage  string // empty means only debug messages are expected
		wantDebug    bool
	}{
		{
			name: "no composer.json",
		},
		{
			name:        "published from this repository",
			composer:    `{"name": "composer/composer", "type": "library"}`,
			sourceURL:   "https://github.com/composer/composer.git",
			wantQuery:   "composer/composer",
			wantPackage: "pkg:composer/composer/composer",
		},
		{
			name:        "published from this repository with scp-like url",
			composer:    `{"name": "composer/composer"}`,
			sourceURL:   "git@github.com:Composer/Composer.git",
			wantQuery:   "composer/composer",
			wantPackage: "pkg:composer/composer/composer",
		},
		{
			name:      "published from a different repository",
			composer:  `{"name": "composer/composer"}`,
			sourceURL: "https://github.com/someone/fork.git",
			wantQuery: "composer/composer",
			wantDebug: true,
		},
		{
			name:         "not on packagist",
			composer:     `{"name": "acme/app", "type": "project"}`,
			packagistErr: packageclient.ErrPackageNotFoundOnPackagist,
			wantQuery:    "acme/app",
			wantDebug:    true,
		},
		{
			name:         "packagist unavailable",
			composer:     `{"name": "acme/app"}`,
			packagistErr: errors.New("connection refused"),
			wantQuery:    "acme/app",
			wantDebug:    true,
		},
		{
			name:      "no package name",
			composer:  `{"require": {"php": "^8.1"}}`,
			wantDebug: true,
		},
		{
			name:      "invalid package name",
			composer:  `{"name": "../../foo"}`,
			wantDebug: true,
		},
		{
			name:      "invalid json",
			composer:  `{"name": `,
			wantDebug: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			mockRepoClient := mockrepo.NewMockRepoClient(ctrl)
			mockRepo := mockrepo.NewMockRepo(ctrl)

			var files []string
			if tt.composer != "" {
				files = []string{"composer.json"}
			}
			mockRepoClient.EXPECT().ListFiles(gomock.Any()).Return(files, nil)
			mockRepoClient.EXPECT().GetFileReader("composer.json").DoAndReturn(func(string) (io.ReadCloser, error) {
				return io.NopCloser(strings.NewReader(tt.composer)), nil
			}).AnyTimes()
			mockRepo.EXPECT().URI().Return("github.com/composer/composer").AnyTimes()

			client := &fakePackagist{sourceURL: tt.sourceURL, err: tt.packagistErr}
			req := checker.CheckRequest{
				Ctx:        context.Background(),
				RepoClient: mockRepoClient,
				Repo:       mockRepo,
			}
			data, err := Packaging(&req, client)
			if err != nil {
				t.Fatalf("Packaging() error = %v", err)
			}
			if client.queried != tt.wantQuery {
				t.Errorf("queried Packagist for %q, want %q", client.queried, tt.wantQuery)
			}

			var gotPackage string
			var gotDebug bool
			for _, p := range data.Packages {
				if p.Msg != nil {
					gotDebug = true
					continue
				}
				if p.Name != nil {
					gotPackage = *p.Name
				}
			}
			if gotPackage != tt.wantPackage {
				t.Errorf("package = %q, want %q", gotPackage, tt.wantPackage)
			}
			if gotDebug != tt.wantDebug {
				t.Errorf("debug message present = %v, want %v", gotDebug, tt.wantDebug)
			}
		})
	}
}
