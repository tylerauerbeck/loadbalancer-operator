package srv

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func TestGetHelmValues(t *testing.T) {
	fs := fstest.MapFS{
		"values.yaml": {
			Data: []byte("hello: world"),
		},
	}

	content, _ := fs.ReadFile("vales.yaml")

	values, err := getHelmValues(content)
	assert.NotNil(t, values)
	assert.Nil(t, err)

	fs = fstest.MapFS{
		"values.yaml": {
			Data: []byte("hello there"),
		},
	}
	content, _ = fs.ReadFile("values.yaml")

	_, err = getHelmValues(content)
	assert.NotNil(t, err)
}
