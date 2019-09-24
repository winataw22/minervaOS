package main

import (
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTestHub(baseURL, user, token string) (*Hub, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}

	jar.SetCookies(base, []*http.Cookie{
		&http.Cookie{Name: "caddyoauth", Value: token},
		&http.Cookie{Name: "active-user", Value: user},
	})

	return &Hub{client: http.Client{Jar: jar}, base: base}, nil
}

func TestHubRename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		require.Equal(t, req.URL.String(), "/me/source/rename/destination")

		token, err := req.Cookie("caddyoauth")
		require.NoError(t, err)
		require.Equal(t, "my jwt token", token.Value)

		user, err := req.Cookie("active-user")
		require.NoError(t, err)
		require.Equal(t, "test-user", user.Value)

		// Send response to be tested
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server.Close()

	hub, err := newTestHub(server.URL, "test-user", "my jwt token")
	require.NoError(t, err)

	err = hub.Rename("source", "destination")

	require.NoError(t, err)
}

func TestHubLink(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		// Test request parameters
		require.Equal(t, req.URL.String(), "/me/source/link/destination")

		token, err := req.Cookie("caddyoauth")
		require.NoError(t, err)
		require.Equal(t, "my jwt token", token.Value)

		user, err := req.Cookie("active-user")
		require.NoError(t, err)
		require.Equal(t, "test-user", user.Value)

		// Send response to be tested
		rw.Write([]byte(`OK`))
	}))
	// Close the server when test finishes
	defer server.Close()

	hub, err := newTestHub(server.URL, "test-user", "my jwt token")
	require.NoError(t, err)

	err = hub.Link("source", "destination")

	require.NoError(t, err)
}
