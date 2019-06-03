package upgrade

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)



func TestMergeFs(t *testing.T) {
	err := mergeFs("/tmp/test/flist", "/tmp/test/root")
	assert.NoError(t, err)
}

func TestChangeBase(t *testing.T) {

	for _, test := range []struct {
		name   string
		base   string
		path   string
		result string
		err    error
	}{
		{
			name:   "valid",
			base:   "/mnt/containers/1",
			path:   "/etc/modules/module.conf",
			result: "/mnt/containers/1/etc/modules/module.conf",
			err:    nil,
		},
		{
			name: "base not absolute",
			base: "containers/1",
			err:  errNotAbsolute,
		},
		{
			name: "path not absolute",
			base: "",
			path: "etc/modules/module.conf",
			err:  errNotAbsolute,
		},
	} {
		t.Run(test.name, func(t *testing.T) {

			result, err := changeRoot(test.base, test.path)
			if test.err != nil {
				assert.Error(t, err)
				assert.Equal(t, test.err, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.result, result)
			}
		})
	}
}
