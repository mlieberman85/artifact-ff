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

package githubclient

import (
	"context"
	"net/http"
	"reflect"
	"testing"

	"github.com/google/go-github/github"
	"github.com/guacsec/guac/internal/client"
)

type MockGithubClient struct {
	token string
}

var (
	mockTag          = "mockTag"
	mockCommit       = "mockCommit"
	mockReleaseAsset = client.ReleaseAsset{
		Name: "mockReleaseAsset",
		URL:  "https://github.com/mock/releaseAsset.jsonl",
	}
	mockReleaseAssetContent = client.ReleaseAssetContent{
		Name:  "releaseAsset.jsonl",
		Bytes: []byte{},
	}
)

// GetLatestRelease fetches the latest release for a repo
func (m *MockGithubClient) GetLatestRelease(ctx context.Context, owner string, repo string) (*client.Release, error) {
	r := &client.Release{
		Tag:    mockTag,
		Commit: mockCommit,
		Assets: []client.ReleaseAsset{mockReleaseAsset},
	}

	return r, nil
}

// GetCommitSHA1 fetches the commit SHA in a repo based on a tag, branch head, or other ref.
// NOTE: Github release 2022-11-28 and similar server returns a commitish for a release.
// The release commitish can be a commit, branch name, or a tag.
// We need to resolve it to a commit.
func (m *MockGithubClient) GetCommitSHA1(ctx context.Context, owner string, repo string, ref string) (string, error) {
	return mockCommit, nil
}

// GetReleaseByTagSlices fetches metadata regarding releases for a given tag. If the tag is the empty string,
// it should just return the latest.
func (m *MockGithubClient) GetReleaseByTag(ctx context.Context, owner string, repo string, tag string) (*client.Release, error) {
	r := &client.Release{
		Tag:    mockTag,
		Commit: mockCommit,
		Assets: []client.ReleaseAsset{mockReleaseAsset},
	}

	return r, nil
}

// GetReleaseAsset fetches the content of a release asset, e.g. artifacts, metadata documents, etc.
func (m *MockGithubClient) GetReleaseAsset(asset client.ReleaseAsset) (*client.ReleaseAssetContent, error) {
}

func TestNewGithubClient(t *testing.T) {
	type args struct {
		ctx   context.Context
		token string
	}
	tests := []struct {
		name string
		args args
		want *githubClient
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewGithubClient(tt.args.ctx, tt.args.token); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewGithubClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_githubClient_GetLatestRelease(t *testing.T) {
	type fields struct {
		ghClient   *github.Client
		httpClient *http.Client
	}
	type args struct {
		ctx   context.Context
		owner string
		repo  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *client.Release
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gc := &githubClient{
				ghClient:   tt.fields.ghClient,
				httpClient: tt.fields.httpClient,
			}
			got, err := gc.GetLatestRelease(tt.args.ctx, tt.args.owner, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("githubClient.GetLatestRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("githubClient.GetLatestRelease() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_githubClient_GetCommitSHA1(t *testing.T) {
	type fields struct {
		ghClient   *github.Client
		httpClient *http.Client
	}
	type args struct {
		ctx   context.Context
		owner string
		repo  string
		ref   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gc := &githubClient{
				ghClient:   tt.fields.ghClient,
				httpClient: tt.fields.httpClient,
			}
			got, err := gc.GetCommitSHA1(tt.args.ctx, tt.args.owner, tt.args.repo, tt.args.ref)
			if (err != nil) != tt.wantErr {
				t.Errorf("githubClient.GetCommitSHA1() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("githubClient.GetCommitSHA1() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_githubClient_GetReleaseByTag(t *testing.T) {
	type fields struct {
		ghClient   *github.Client
		httpClient *http.Client
	}
	type args struct {
		ctx   context.Context
		owner string
		repo  string
		tag   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *client.Release
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gc := &githubClient{
				ghClient:   tt.fields.ghClient,
				httpClient: tt.fields.httpClient,
			}
			got, err := gc.GetReleaseByTag(tt.args.ctx, tt.args.owner, tt.args.repo, tt.args.tag)
			if (err != nil) != tt.wantErr {
				t.Errorf("githubClient.GetReleaseByTag() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("githubClient.GetReleaseByTag() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_githubClient_GetReleaseAsset(t *testing.T) {
	type fields struct {
		ghClient   *github.Client
		httpClient *http.Client
	}
	type args struct {
		asset client.ReleaseAsset
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *client.ReleaseAssetContent
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gc := &githubClient{
				ghClient:   tt.fields.ghClient,
				httpClient: tt.fields.httpClient,
			}
			got, err := gc.GetReleaseAsset(tt.args.asset)
			if (err != nil) != tt.wantErr {
				t.Errorf("githubClient.GetReleaseAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("githubClient.GetReleaseAsset() = %v, want %v", got, tt.want)
			}
		})
	}
}
