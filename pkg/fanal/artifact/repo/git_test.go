//go:build unix

package repo

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aquasecurity/trivy/internal/gittest"
	"github.com/aquasecurity/trivy/pkg/cache"
	"github.com/aquasecurity/trivy/pkg/fanal/artifact"
	"github.com/aquasecurity/trivy/pkg/fanal/walker"
	"github.com/aquasecurity/trivy/pkg/uuid"

	_ "github.com/aquasecurity/trivy/pkg/fanal/analyzer/config/all"
	_ "github.com/aquasecurity/trivy/pkg/fanal/analyzer/secret"
)

func setupGitRepository(t *testing.T, repo, dir string) (*httptest.Server, *git.Repository) {
	gs := gittest.NewServer(t, repo, dir)

	worktree := t.TempDir()
	r := gittest.Clone(t, gs, repo, worktree)

	// Branch
	gittest.CreateRemoteBranch(t, r, "valid-branch")

	// Tag
	gittest.SetTag(t, r, "v1.0.0")
	gittest.PushTags(t, r)

	return gs, r
}

func TestNewArtifact(t *testing.T) {
	ts, repo := setupGitRepository(t, "test-repo", "testdata/test-repo")
	defer ts.Close()

	head, err := repo.Head()
	require.NoError(t, err)

	type args struct {
		target     string
		c          cache.ArtifactCache
		noProgress bool
		repoBranch string
		repoTag    string
		repoCommit string
	}
	tests := []struct {
		name      string
		args      args
		assertion assert.ErrorAssertionFunc
	}{
		{
			name: "remote repo",
			args: args{
				target:     ts.URL + "/test-repo.git",
				c:          nil,
				noProgress: false,
			},
			assertion: assert.NoError,
		},
		{
			name: "local repo",
			args: args{
				target:     "testdata",
				c:          nil,
				noProgress: false,
			},
			assertion: assert.NoError,
		},
		{
			name: "no progress",
			args: args{
				target:     ts.URL + "/test-repo.git",
				c:          nil,
				noProgress: true,
			},
			assertion: assert.NoError,
		},
		{
			name: "branch",
			args: args{
				target:     ts.URL + "/test-repo.git",
				c:          nil,
				repoBranch: "valid-branch",
			},
			assertion: assert.NoError,
		},
		{
			name: "tag",
			args: args{
				target:  ts.URL + "/test-repo.git",
				c:       nil,
				repoTag: "v1.0.0",
			},
			assertion: assert.NoError,
		},
		{
			name: "commit",
			args: args{
				target:     ts.URL + "/test-repo.git",
				c:          nil,
				repoCommit: head.String(),
			},
			assertion: assert.NoError,
		},
		{
			name: "sad path",
			args: args{
				target:     ts.URL + "/unknown.git",
				c:          nil,
				noProgress: false,
			},
			assertion: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorContains(t, err, "repository not found")
			},
		},
		{
			name: "invalid url",
			args: args{
				target:     "ht tp://foo.com",
				c:          nil,
				noProgress: false,
			},
			assertion: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorContains(t, err, "url parse error")
			},
		},
		{
			name: "invalid branch",
			args: args{
				target:     ts.URL + "/test-repo.git",
				c:          nil,
				repoBranch: "invalid-branch",
			},
			assertion: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorContains(t, err, `couldn't find remote ref "refs/heads/invalid-branch"`)
			},
		},
		{
			name: "invalid tag",
			args: args{
				target:  ts.URL + "/test-repo.git",
				c:       nil,
				repoTag: "v1.0.9",
			},
			assertion: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorContains(t, err, `couldn't find remote ref "refs/tags/v1.0.9"`)
			},
		},
		{
			name: "invalid commit",
			args: args{
				target:     ts.URL + "/test-repo.git",
				c:          nil,
				repoCommit: "6ac152fe2b87cb5e243414df71790a32912e778e",
			},
			assertion: func(t assert.TestingT, err error, args ...any) bool {
				return assert.ErrorContains(t, err, "git checkout error: object not found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup, err := NewArtifact(tt.args.target, tt.args.c, walker.NewFS(), artifact.Option{
				NoProgress: tt.args.noProgress,
				RepoBranch: tt.args.repoBranch,
				RepoTag:    tt.args.repoTag,
				RepoCommit: tt.args.repoCommit,
			})
			tt.assertion(t, err)
			defer cleanup()
		})
	}
}

func TestArtifact_Inspect(t *testing.T) {
	ts, _ := setupGitRepository(t, "test-repo", "testdata/test-repo")
	defer ts.Close()

	tests := []struct {
		name    string
		rawurl  string
		want    artifact.Reference
		wantErr bool
	}{
		{
			name:   "happy path",
			rawurl: ts.URL + "/test-repo.git",
			want: artifact.Reference{
				Name: ts.URL + "/test-repo.git",
				Type: artifact.TypeRepository,
				ID:   "sha256:6f4672e139d4066fd00391df614cdf42bda5f7a3f005d39e1d8600be86157098",
				BlobIDs: []string{
					"sha256:6f4672e139d4066fd00391df614cdf42bda5f7a3f005d39e1d8600be86157098",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set fake UUID for consistent test results
			uuid.SetFakeUUID(t, "3ff14136-e09f-4df9-80ea-%012d")

			fsCache, err := cache.NewFSCache(t.TempDir())
			require.NoError(t, err)

			art, cleanup, err := NewArtifact(tt.rawurl, fsCache, walker.NewFS(), artifact.Option{})
			require.NoError(t, err)
			defer cleanup()

			ref, err := art.Inspect(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.want, ref)
		})
	}
}

func Test_newURL(t *testing.T) {
	type args struct {
		rawurl string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr string
	}{
		{
			name: "happy path",
			args: args{
				rawurl: "https://github.com/aquasecurity/fanal",
			},
			want: "https://github.com/aquasecurity/fanal",
		},
		{
			name: "happy path: no scheme",
			args: args{
				rawurl: "github.com/aquasecurity/fanal",
			},
			want: "https://github.com/aquasecurity/fanal",
		},
		{
			name: "sad path: invalid url",
			args: args{
				rawurl: "ht tp://foo.com",
			},
			wantErr: "first path segment in URL cannot contain colon",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newURL(tt.args.rawurl)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			} else {
				require.NoError(t, err)
			}

			assert.Equal(t, tt.want, got.String())
		})
	}
}
