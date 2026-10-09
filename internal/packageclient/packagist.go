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
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// PackagistClient looks up PHP packages published on Packagist.
type PackagistClient interface {
	// GetSourceRepository returns the source repository URL of the most
	// recent tagged release of the given Composer package.
	GetSourceRepository(ctx context.Context, name string) (string, error)
}

// packagistMetadataURL serves the static metadata of tagged releases
// https://packagist.org/apidoc#get-package-metadata-v2
const packagistMetadataURL = "https://repo.packagist.org/p2/"

var (
	ErrPackagistAPI                = errors.New("packagist")
	ErrPackageNotFoundOnPackagist  = errors.New("package not found on Packagist")
	ErrPackageHasNoTaggedReleases  = errors.New("package has no tagged releases on Packagist")
	errPackagistMissingSourceField = errors.New("packagist metadata has no source repository")
)

type packagistClient struct {
	client  *http.Client
	baseURL string
}

func CreatePackagistClient() PackagistClient {
	return packagistClient{
		client:  &http.Client{},
		baseURL: packagistMetadataURL,
	}
}

type packagistMetadata struct {
	Packages map[string][]struct {
		Source struct {
			URL string `json:"url"`
		} `json:"source"`
	} `json:"packages"`
}

// GetSourceRepository implements PackagistClient. The name must be a
// valid Composer package name, which is lowercase "vendor/project".
func (p packagistClient) GetSourceRepository(ctx context.Context, name string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+name+".json", nil)
	if err != nil {
		return "", fmt.Errorf("http.NewRequestWithContext: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("packagist GetSourceRepository: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return "", ErrPackageNotFoundOnPackagist
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %s", ErrPackagistAPI, resp.Status)
	}

	var res packagistMetadata
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", fmt.Errorf("packagist json decode: %w", err)
	}

	// Versions are ordered newest first. The metadata is minified, but the
	// first version always contains all fields.
	versions := res.Packages[name]
	if len(versions) == 0 {
		return "", ErrPackageHasNoTaggedReleases
	}
	if versions[0].Source.URL == "" {
		return "", errPackagistMissingSourceField
	}
	return versions[0].Source.URL, nil
}
