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

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/go-github/v82/github"
)

const gitattributesFile = ".gitattributes"

// hasExportIgnoreAttribute reports whether the given .gitattributes content
// sets the export-ignore attribute on any path. Files with this attribute are
// left out of the tarballs GitHub generates, so they are invisible to checks
// unless the repository is fetched with git instead.
// https://git-scm.com/docs/gitattributes#_export_ignore
func hasExportIgnoreAttribute(content []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// The first field is the path pattern, the remaining ones are attributes.
		// "-export-ignore" and "!export-ignore" unset the attribute, so they
		// don't cause files to be excluded.
		fields := strings.Fields(line)
		for _, attr := range fields[1:] {
			if attr == "export-ignore" || attr == "export-ignore=true" {
				return true
			}
		}
	}
	return false
}

// usesExportIgnore reports whether the repository's root .gitattributes file
// on the default branch excludes any files from generated archives.
func usesExportIgnore(ctx context.Context, client *github.Client, owner, repo string) (bool, error) {
	file, _, resp, err := client.Repositories.GetContents(ctx, owner, repo, gitattributesFile, nil)
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("Repositories.GetContents: %w", err)
	}
	if file == nil {
		// .gitattributes is a directory.
		return false, nil
	}
	content, err := file.GetContent()
	if err != nil {
		return false, fmt.Errorf("decode %s: %w", gitattributesFile, err)
	}
	return hasExportIgnoreAttribute([]byte(content)), nil
}
