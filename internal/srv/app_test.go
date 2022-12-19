package srv

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestGetHelmValues(t *testing.T) {
	type testCase struct {
		name        string
		content     *fstest.MapFS
		file        string
		expectError bool
	}

	testCases := []testCase{
		{
			name:        "valid yaml",
			expectError: false,
			file:        "values.yaml",
			content: &fstest.MapFS{
				"values.yaml": {
					Data: []byte("hello: world"),
				},
			},
		},
		{
			name:        "invalid yaml",
			expectError: true,
			file:        "values.yaml",
			content: &fstest.MapFS{
				"values.yaml": {
					Data: []byte("hello there"),
				},
			},
		},
	}

	for _, tcase := range testCases {
		t.Run(tcase.name, func(t *testing.T) {
			srv := Server{
				Logger: setupTestLogger(t, tcase.name),
			}
			content, _ := tcase.content.ReadFile(tcase.file)
			values, err := srv.getHelmValues(content)

			if tcase.expectError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, values)
			}
		})
	}
}
