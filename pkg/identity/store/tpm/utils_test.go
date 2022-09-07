package tpm

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPersistedHandlers(t *testing.T) {
	t.Skip("manual test needs specific setup")
	handlers, err := PersistedHandlers(context.Background())
	assert.NoError(t, err)
	fmt.Println(handlers)
}

func TestPCRs(t *testing.T) {
	t.Skip("manual test needs specific setup")
	pcrs, err := PCRs(context.Background())
	assert.NoError(t, err)
	fmt.Println(pcrs)
}

func TestPCRSelector(t *testing.T) {
	selector := PCRSelector{
		SHA1: {1, 2, 3},
	}

	assert.Equal(t, "sha1:1,2,3", selector.String())

	selector = PCRSelector{
		SHA1:   {1, 2, 3},
		SHA256: {3, 4},
	}

	assert.Equal(t, "sha1:1,2,3+sha256:3,4", selector.String())
}

func TestPCRPolicy(t *testing.T) {
	t.Skip("manual test")
	selector := PCRSelector{
		SHA1: {0, 1, 2},
	}

	hash, err := CreatePCRPolicy(context.Background(), selector)
	assert.NoError(t, err)
	assert.Equal(t, HexString("f3473b1bc51f0c9179f4c6f377b5b7d20bde98793e1a052c7bc3b736b821d1fe"), hash)
}
