package network

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertMac(t *testing.T) {
	id := convertMac("fe:44:e1:67:a8:d2")
	assert.Equal(t, "fe44e167a8d2", id)
}
