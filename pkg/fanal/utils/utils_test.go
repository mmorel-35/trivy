package utils

import (
	"bufio"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGzip(t *testing.T) {
	var tests = []struct {
		in   string
		want bool
	}{
		{filepath.Join("testdata", "test.txt.gz"), true},
		{filepath.Join("testdata", "test.tar.gz"), true},
		{filepath.Join("testdata", "test.txt"), false},
		{filepath.Join("testdata", "test.txt.zst"), false},
		{filepath.Join("testdata", "aqua.png"), false},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			f, err := os.Open(tt.in)
			require.NoErrorf(t, err, "unknown error: %s", err)

			got := IsGzip(bufio.NewReader(f))
			assert.Equalf(t, tt.want, got, "got %t, want %t", got, tt.want)
		})
	}
}
