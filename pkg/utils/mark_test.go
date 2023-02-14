package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMark(t *testing.T) {
	mark := NewMark()

	mark.Signal()
	mark.Signal()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	require.NoError(t, mark.Done(ctx))
}
