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

// Package packagist detects PHP packages published on Packagist.
//
// Packagist does not receive build artifacts. It reads composer.json directly
// from the source repository and publishes every git tag as a release, so
// there is no publishing workflow to detect. Instead, the package name from
// composer.json is looked up on Packagist and its source repository compared
// with the repository being analyzed.
package packagist

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/ossf/scorecard/v5/checker"
	"github.com/ossf/scorecard/v5/finding"
	"github.com/ossf/scorecard/v5/internal/packageclient"
)

const composerFile = "composer.json"

// https://getcomposer.org/doc/04-schema.md#name
var composerPackageName = regexp.MustCompile(
	`^[a-z0-9]([_.-]?[a-z0-9]+)*/[a-z0-9](([_.]|-{1,2})?[a-z0-9]+)*$`)

// Packaging checks whether the repository is published as a package on Packagist.
func Packaging(c *checker.CheckRequest, client packageclient.PackagistClient) (checker.PackagingData, error) {
	var data checker.PackagingData
	matchedFiles, err := c.RepoClient.ListFiles(func(path string) (bool, error) {
		return path == composerFile, nil
	})
	if err != nil {
		return data, fmt.Errorf("RepoClient.ListFiles: %w", err)
	}
	if len(matchedFiles) == 0 {
		return data, nil
	}

	fr, err := c.RepoClient.GetFileReader(composerFile)
	if err != nil {
		return data, fmt.Errorf("RepoClient.GetFileReader: %w", err)
	}
	fc, err := io.ReadAll(fr)
	fr.Close()
	if err != nil {
		return data, fmt.Errorf("reading file: %w", err)
	}

	file := &checker.File{
		Path:   composerFile,
		Type:   finding.FileTypeSource,
		Offset: checker.OffsetDefault,
	}
	debug := func(msg string) checker.PackagingData {
		data.Packages = append(data.Packages, checker.Package{Msg: &msg, File: file})
		return data
	}

	var composer struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(fc, &composer); err != nil {
		return debug(fmt.Sprintf("unable to parse %s: %v", composerFile, err)), nil
	}
	if !composerPackageName.MatchString(composer.Name) {
		return debug(fmt.Sprintf("%s does not declare a valid package name", composerFile)), nil
	}

	sourceURL, err := client.GetSourceRepository(c.Ctx, composer.Name)
	switch {
	case errors.Is(err, packageclient.ErrPackageNotFoundOnPackagist),
		errors.Is(err, packageclient.ErrPackageHasNoTaggedReleases):
		return debug(fmt.Sprintf("%s: %v", composer.Name, err)), nil
	case err != nil:
		return debug(fmt.Sprintf("unable to query Packagist for %s: %v", composer.Name, err)), nil
	}

	if normalizeRepoURL(sourceURL) != normalizeRepoURL(c.Repo.URI()) {
		return debug(fmt.Sprintf("Packagist package %s is published from a different repository: %s",
			composer.Name, sourceURL)), nil
	}

	name := "pkg:composer/" + composer.Name
	data.Packages = append(data.Packages, checker.Package{
		Name: &name,
		File: file,
	})
	return data, nil
}

// normalizeRepoURL reduces git URLs like "https://github.com/foo/bar.git",
// "git@github.com:foo/bar.git" and repo URIs like "github.com/foo/bar" to a
// comparable "github.com/foo/bar" form.
func normalizeRepoURL(u string) string {
	u = strings.ToLower(strings.TrimSpace(u))
	if i := strings.Index(u, "://"); i >= 0 {
		u = u[i+len("://"):]
	} else if at := strings.Index(u, "@"); at >= 0 {
		// scp-like syntax: git@github.com:foo/bar.git
		u = strings.Replace(u[at+1:], ":", "/", 1)
	}
	u = strings.TrimSuffix(u, "/")
	u = strings.TrimSuffix(u, ".git")
	return u
}
