package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_Raw(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		expected any
	}{
		{
			name:     "parameter with a string type value",
			key:      "AWS::AccountId",
			expected: "123456789012",
		},
		{
			name:     "a parameter with a list type value",
			key:      "AWS::NotificationARNs",
			expected: []string{"notification::arn::1", "notification::arn::2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, ok := pseudoParameters[tt.key]
			require.False(t, ok, "unexpected parameter key")
			assert.Equal(t, tt.expected, val.getRawValue())
		})
	}
}
