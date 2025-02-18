package google

import (
	"reflect"
	"testing"

	"github.com/GoogleCloudPlatform/docker-credential-gcr/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/aquasecurity/trivy/pkg/fanal/types"
)

func TestCheckOptions(t *testing.T) {
	var tests = map[string]struct {
		domain  string
		opt     types.RegistryOptions
		grc     *GoogleRegistryClient
		wantErr error
	}{
		"InvalidURL": {
			domain:  "alpine:3.9",
			wantErr: types.InvalidURLPattern,
		},
		"InvalidDomain": {
			domain:  "not-gcr.io",
			wantErr: types.InvalidURLPattern,
		},
		"NoOption": {
			domain: "gcr.io",
			grc:    &GoogleRegistryClient{domain: "gcr.io"},
		},
		"CredOption": {
			domain: "gcr.io",
			opt:    types.RegistryOptions{GCPCredPath: "/path/to/file.json"},
			grc: &GoogleRegistryClient{
				domain: "gcr.io",
				Store:  store.NewGCRCredStore("/path/to/file.json"),
			},
		},
	}

	for testname, v := range tests {
		g := &Registry{}
		grc, err := g.CheckOptions(v.domain, v.opt)
		if v.wantErr != nil {
			require.ErrorIsf(t, err, v.wantErr, "[%s]\nexpected error based on %v\nactual : %v", testname, v.wantErr, err)
		} else {
			assert.Truef(t, reflect.DeepEqual(v.grc, grc), "[%s]\nexpected : %v\nactual : %v", testname, v.grc, grc)
		}
	}
}
