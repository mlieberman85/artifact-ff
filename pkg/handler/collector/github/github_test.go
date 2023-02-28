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
	"reflect"
	"testing"
	"time"

	"github.com/guacsec/guac/internal/client"
	"github.com/guacsec/guac/internal/client/githubclient"
	"github.com/guacsec/guac/pkg/collectsub/datasource"
	"github.com/guacsec/guac/pkg/handler/processor"
)

func Test_tag_IsTagOrLatest(t *testing.T) {
	type fields struct {
		Value string
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &tag{
				Value: tt.fields.Value,
			}
			tr.IsTagOrLatest()
		})
	}
}

func Test_latest_IsTagOrLatest(t *testing.T) {
	tests := []struct {
		name string
		l    *latest
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l := &latest{}
			l.IsTagOrLatest()
		})
	}
}

func TestNewGithubCollector(t *testing.T) {
	type args struct {
		ctx context.Context
		c   Config
	}
	tests := []struct {
		name    string
		args    args
		want    *githubCollector
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGithubCollector(tt.args.ctx, tt.args.c)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewGithubCollector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGithubCollector() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_githubCollector_RetrieveArtifacts(t *testing.T) {
	type fields struct {
		poll              bool
		interval          time.Duration
		client            githubclient.GithubClient
		repoToReleaseTags map[client.Repo][]tagOrLatest
		assetSuffixes     []string
		collectDataSource datasource.CollectSource
	}
	type args struct {
		ctx        context.Context
		docChannel chan<- *processor.Document
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &githubCollector{
				poll:              tt.fields.poll,
				interval:          tt.fields.interval,
				client:            tt.fields.client,
				repoToReleaseTags: tt.fields.repoToReleaseTags,
				assetSuffixes:     tt.fields.assetSuffixes,
				collectDataSource: tt.fields.collectDataSource,
			}
			if err := g.RetrieveArtifacts(tt.args.ctx, tt.args.docChannel); (err != nil) != tt.wantErr {
				t.Errorf("githubCollector.RetrieveArtifacts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_githubCollector_Type(t *testing.T) {
	type fields struct {
		poll              bool
		interval          time.Duration
		client            githubclient.GithubClient
		repoToReleaseTags map[client.Repo][]tagOrLatest
		assetSuffixes     []string
		collectDataSource datasource.CollectSource
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &githubCollector{
				poll:              tt.fields.poll,
				interval:          tt.fields.interval,
				client:            tt.fields.client,
				repoToReleaseTags: tt.fields.repoToReleaseTags,
				assetSuffixes:     tt.fields.assetSuffixes,
				collectDataSource: tt.fields.collectDataSource,
			}
			if got := g.Type(); got != tt.want {
				t.Errorf("githubCollector.Type() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_githubCollector_populateRepoToReleaseTags(t *testing.T) {
	type fields struct {
		poll              bool
		interval          time.Duration
		client            githubclient.GithubClient
		repoToReleaseTags map[client.Repo][]tagOrLatest
		assetSuffixes     []string
		collectDataSource datasource.CollectSource
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &githubCollector{
				poll:              tt.fields.poll,
				interval:          tt.fields.interval,
				client:            tt.fields.client,
				repoToReleaseTags: tt.fields.repoToReleaseTags,
				assetSuffixes:     tt.fields.assetSuffixes,
				collectDataSource: tt.fields.collectDataSource,
			}
			if err := g.populateRepoToReleaseTags(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("githubCollector.populateRepoToReleaseTags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_githubCollector_fetchAssets(t *testing.T) {
	type fields struct {
		poll              bool
		interval          time.Duration
		client            githubclient.GithubClient
		repoToReleaseTags map[client.Repo][]tagOrLatest
		assetSuffixes     []string
		collectDataSource datasource.CollectSource
	}
	type args struct {
		ctx        context.Context
		owner      string
		repo       string
		tags       []tagOrLatest
		docChannel chan<- *processor.Document
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &githubCollector{
				poll:              tt.fields.poll,
				interval:          tt.fields.interval,
				client:            tt.fields.client,
				repoToReleaseTags: tt.fields.repoToReleaseTags,
				assetSuffixes:     tt.fields.assetSuffixes,
				collectDataSource: tt.fields.collectDataSource,
			}
			g.fetchAssets(tt.args.ctx, tt.args.owner, tt.args.repo, tt.args.tags, tt.args.docChannel)
		})
	}
}

func Test_githubCollector_collectAssetsForRelease(t *testing.T) {
	type fields struct {
		poll              bool
		interval          time.Duration
		client            githubclient.GithubClient
		repoToReleaseTags map[client.Repo][]tagOrLatest
		assetSuffixes     []string
		collectDataSource datasource.CollectSource
	}
	type args struct {
		ctx        context.Context
		release    client.Release
		docChannel chan<- *processor.Document
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &githubCollector{
				poll:              tt.fields.poll,
				interval:          tt.fields.interval,
				client:            tt.fields.client,
				repoToReleaseTags: tt.fields.repoToReleaseTags,
				assetSuffixes:     tt.fields.assetSuffixes,
				collectDataSource: tt.fields.collectDataSource,
			}
			g.collectAssetsForRelease(tt.args.ctx, tt.args.release, tt.args.docChannel)
		})
	}
}

func Test_checkSuffixes(t *testing.T) {
	type args struct {
		name     string
		suffixes []string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := checkSuffixes(tt.args.name, tt.args.suffixes); got != tt.want {
				t.Errorf("checkSuffixes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parseGithubReleaseDataSource(t *testing.T) {
	type args struct {
		source datasource.Source
	}
	tests := []struct {
		name    string
		args    args
		want    *client.Repo
		want1   tagOrLatest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseGithubReleaseDataSource(tt.args.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGithubReleaseDataSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseGithubReleaseDataSource() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("parseGithubReleaseDataSource() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_parseGitDataSource(t *testing.T) {
	type args struct {
		source datasource.Source
	}
	tests := []struct {
		name    string
		args    args
		want    *client.Repo
		want1   tagOrLatest
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := parseGitDataSource(tt.args.source)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitDataSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseGitDataSource() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("parseGitDataSource() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
