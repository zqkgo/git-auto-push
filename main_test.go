package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	assert := assert.New(t)
	assert.Nil(nil)
	f, err := os.CreateTemp(os.TempDir(), "parseConfig*")
	assert.Nil(err)
	fname := f.Name()

	defer func() {
		os.Remove(fname)
		t.Logf("%s removed", fname)
	}()

	s := `
	{
		"interval_ms": 10000,
		"repositories": [
			{
				"path": "/path/to/your/local/repository1",
				"remote": "origin",
				"branch": "main"
			},
			{
				"path": "/path/to/your/local/repository2",
				"remote": "origin",
				"branch": "master"
			}
		]
	}
	`
	_, err = f.WriteString(s)
	assert.Nil(err)
	f.Close()

	c, err := parseConfig(fname)
	assert.Nil(err)
	assert.Equal(2, len(c.Repositories))
	assert.Equal(10000, c.IntervalMs)
}
