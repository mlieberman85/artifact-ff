//
// Copyright 2023 The GUAC Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/guacsec/guac/internal/client"
	"github.com/guacsec/guac/internal/client/githubclient"
	"github.com/guacsec/guac/pkg/collectsub/datasource"
	"github.com/guacsec/guac/pkg/handler/processor"
	"github.com/guacsec/guac/pkg/logging"
)

const (
	GithubCollector = "GithubCollector"
)

type tagOrLatest interface {
	IsTagOrLatest()
}

type tag struct {
	Value string
}

type latest struct{}

func (t *tag) IsTagOrLatest()    {}
func (l *latest) IsTagOrLatest() {}

// Compile time checks
var _ tagOrLatest = &tag{}
var _ tagOrLatest = &latest{}

type githubCollector struct {
	poll              bool
	interval          time.Duration
	client            githubclient.GithubClient
	repoToReleaseTags map[client.Repo][]tagOrLatest
	assetSuffixes     []string
	collectDataSource datasource.CollectSource
}

type Config struct {
	Poll              bool
	Interval          time.Duration
	Client            githubclient.GithubClient
	RepoToReleaseTags map[client.Repo][]tagOrLatest
	AssetSuffixes     []string
	CollectDataSource datasource.CollectSource
}

func NewGithubCollector(ctx context.Context, c Config) (*githubCollector, error) {
	return &githubCollector{
		poll:              c.Poll,
		interval:          c.Interval,
		client:            c.Client,
		repoToReleaseTags: c.RepoToReleaseTags,
		assetSuffixes:     c.AssetSuffixes,
		collectDataSource: c.CollectDataSource,
	}, nil
}

// RetrieveArtifacts get the artifacts from the collector source based on polling or one time
func (g *githubCollector) RetrieveArtifacts(ctx context.Context, docChannel chan<- *processor.Document) error {
	err := g.populateRepoToReleaseTags(ctx)
	if err != nil {
		return err
	}
	if g.poll {
		for repo, tags := range g.repoToReleaseTags {
			// releases are mutable in Github, but it doesn't make sense in most situations to poll a release
			if len(tags) > 0 {
				return errors.New("image tag should not specified when using polling")
			} else {
				g.fetchAssets(ctx, repo.Owner, repo.Repo, tags, docChannel)
			}
			time.Sleep(g.interval)
		}
	} else {
		for repo, tags := range g.repoToReleaseTags {
			g.fetchAssets(ctx, repo.Owner, repo.Repo, tags, docChannel)
		}
	}

	return nil
}

func (g *githubCollector) Type() string {
	return GithubCollector
}

func (g *githubCollector) populateRepoToReleaseTags(ctx context.Context) error {
	logger := logging.FromContext(ctx)
	ds, err := g.collectDataSource.GetDataSources(ctx)
	if err != nil {
		return fmt.Errorf("unable to retrieve datasource: %w", err)
	}

	for _, grds := range ds.GithubReleaseDataSources {
		r, t, err := parseGithubReleaseDataSource(grds)
		if err != nil {
			logger.Warnf("unable to parse github datasource: %w", err)
			continue
		}
		g.repoToReleaseTags[*r] = append(g.repoToReleaseTags[*r], t)
	}

	for _, gds := range ds.GitDataSources {
		r, t, err := parseGitDataSource(gds)
		if err != nil {
			logger.Warnf("unable to parse git datasource: %w", err)
		}
		g.repoToReleaseTags[*r] = append(g.repoToReleaseTags[*r], t)
	}

	return nil
}

func (g *githubCollector) fetchAssets(ctx context.Context, owner string, repo string, tags []tagOrLatest, docChannel chan<- *processor.Document) {
	logger := logging.FromContext(ctx)
	var releases []client.Release
	for _, gitTag := range tags {
		var release *client.Release
		var err error
		switch t := gitTag.(type) {
		case *latest:
			release, err = g.client.GetLatestRelease(ctx, owner, repo)
		case *tag:
			release, err = g.client.GetReleaseByTag(ctx, owner, repo, t.Value)
		}
		if err != nil {
			logger.Warnf("unable to fetch release: %w", err)
			continue
		}
		releases = append(releases, *release)
	}

	for _, release := range releases {
		g.collectAssetsForRelease(ctx, release, docChannel)
	}
}

func (g *githubCollector) collectAssetsForRelease(ctx context.Context, release client.Release, docChannel chan<- *processor.Document) {
	logger := logging.FromContext(ctx)
	for _, asset := range release.Assets {
		if checkSuffixes(asset.Name, g.assetSuffixes) {
			content, err := g.client.GetReleaseAsset(asset)
			if err != nil {
				logger.Warnf("unable to download asset: %w", err)
				continue
			}
			doc := &processor.Document{
				Blob:   content.Bytes,
				Type:   processor.DocumentUnknown,
				Format: processor.FormatUnknown,
				SourceInformation: processor.SourceInformation{
					Collector: GithubCollector,
					Source:    content.Name,
				},
			}

			docChannel <- doc
		}
	}
}

func checkSuffixes(name string, suffixes []string) bool {
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			return true
		}
	}

	return false
}

func parseGithubReleaseDataSource(source datasource.Source) (*client.Repo, tagOrLatest, error) {
	u, err := url.Parse(source.Value)
	if err != nil {
		return nil, nil, err
	}
	if u.Scheme != "https" {
		return nil, nil, fmt.Errorf("invalid github url scheme: %v", u.Scheme)
	}
	if u.Host != "github.com" {
		return nil, nil, fmt.Errorf("invalid github host: %v", u.Host)
	}
	path := strings.Split(u.Path, "/")
	if len(path) < 3 || len(path) > 5 {
		return nil, nil, fmt.Errorf("invalid github url path: %v invalid number of subpaths: %v", u.Path, len(path))
	}
	if path[2] != "releases" || (path[3] != "tags" && len(path) == 5) {
		return nil, nil, fmt.Errorf("invalid github path: %v", u.Path)
	}
	var tol tagOrLatest
	if len(path) == 5 {
		tol = &tag{
			Value: path[4],
		}
	} else if len(path) == 4 {
		tol = &tag{
			Value: path[3],
		}
	} else {
		tol = &latest{}
	}
	r := &client.Repo{
		Owner: path[0],
		Repo:  path[1],
	}

	return r, tol, nil
}

func parseGitDataSource(source datasource.Source) (*client.Repo, tagOrLatest, error) {
	u, err := url.Parse(source.Value)
	if err != nil {
		return nil, nil, err
	}
	if u.Host != "github.com" {
		return nil, nil, fmt.Errorf("invalid github host: %v", u.Host)
	}

	path := strings.Split(u.Path, "/")
	if len(path) != 2 {
		return nil, nil, fmt.Errorf("invalid github uri path: %v invalid number of subpaths: %v", u.Path, len(path))
	}
	final := strings.Split(path[len(path)-1], "@")
	if len(final) > 2 {
		return nil, nil, fmt.Errorf("invalid tag path, only expected one @: %v", u.Path)
	}
	var tol tagOrLatest
	var r *client.Repo
	if len(final) == 2 {
		tol = &tag{
			Value: final[1],
		}
		r = &client.Repo{
			Owner: path[0],
			Repo:  final[0],
		}
	} else {
		tol = &latest{}
		r = &client.Repo{
			Owner: path[0],
			Repo:  final[1],
		}
	}

	return r, tol, nil
}
