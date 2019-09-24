package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"path/filepath"
)

const (
	baseHubURL = "https://hub.grid.tf/api/flist"
)

// Hub API
type Hub struct {
	base   *url.URL
	client http.Client
}

// NewHub creates a new hub client
func NewHub(token string) (*Hub, error) {
	base, err := url.Parse(baseHubURL)
	if err != nil {
		return nil, err
	}

	user, err := JWTUser(token)
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

func (h *Hub) join(p ...string) string {
	b := *h.base
	b.Path = filepath.Join(b.Path, filepath.Join(p...))

	return b.String()
}

// Rename an flist from src name to dst
func (h *Hub) Rename(src, dst string) error {
	response, err := h.client.Get(h.join("me", src, "rename", dst))
	if err != nil {
		return err
	}

	defer response.Body.Close()
	defer ioutil.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("rename failed with error: %s", response.Status)
	}

	return nil
}

// Link a source flist to name ln
func (h *Hub) Link(src, ln string) error {
	response, err := h.client.Get(h.join("me", src, "link", ln))
	if err != nil {
		return err
	}

	defer response.Body.Close()
	defer ioutil.ReadAll(response.Body)

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("rename failed with error: %s", response.Status)
	}

	return nil
}
