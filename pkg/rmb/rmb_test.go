package rmb

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouter(t *testing.T) {
	require := require.New(t)

	router := newSubRouter()

	require.NotNil(router)
	router.WithHandler("test.handler.do", func(ctx context.Context, payload []byte) (interface{}, error) {
		return nil, nil
	})

	sub := router.Subroute("test")

	sub.WithHandler("handler.do2", func(ctx context.Context, payload []byte) (interface{}, error) {
		return nil, nil
	})

	l := make([]string, 0)
	router.getTopics("", &l)

	require.Len(l, 2)
	require.Equal([]string{"test.handler.do", "test.handler.do2"}, l)
}

func TestRouter1(t *testing.T) {
	require := require.New(t)

	router := newSubRouter()
	require.NotNil(router)

	subRouter := router.Subroute("msgbus").Subroute("zos").Subroute("statistics")
	subRouter.WithHandler("get", func(ctx context.Context, payload []byte) (interface{}, error) {
		return nil, nil
	})

	router.Subroute("msgbus").Subroute("zos").Subroute("statistics")

	l := make([]string, 0)
	router.getTopics("", &l)

	require.Len(l, 1)
	require.Equal([]string{"msgbus.zos.statistics.get"}, l)

	_, err := router.call(context.Background(), "msgbus.zos.statistics.get", nil)
	require.NoError(err)
}

type nameKey struct{}
type ageKey struct{}

func TestMiddleware(t *testing.T) {
	require := require.New(t)

	router := newSubRouter()
	router.Use(func(ctx context.Context, payload []byte) (context.Context, error) {
		return context.WithValue(ctx, nameKey{}, "test middleware"), nil
	})

	router.WithHandler("test.handle.do", func(ctx context.Context, payload []byte) (interface{}, error) {
		value := ctx.Value(nameKey{})
		require.Equal("test middleware", value)

		notVisibleHere := ctx.Value(ageKey{})
		require.Nil(notVisibleHere)
		return nil, nil
	})

	sub := router.Subroute("test")
	sub.Use(func(ctx context.Context, payload []byte) (context.Context, error) {
		return context.WithValue(ctx, ageKey{}, 150), nil
	})

	sub.WithHandler("handle.do2", func(ctx context.Context, payload []byte) (interface{}, error) {
		value := ctx.Value(nameKey{})
		require.Equal("test middleware", value)

		age := ctx.Value(ageKey{})
		require.EqualValues(150, age)
		return nil, nil
	})

	_, err := router.call(context.Background(), "test.handle.do", nil)
	require.NoError(err)
	_, err = router.call(context.Background(), "test.handle.do2", nil)
	require.NoError(err)
}
