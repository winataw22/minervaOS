package gateway

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/stubs"
)

const (
	flist = "https://hub.grid.tf/tf-autobuilder/traefik-2.9.9.flist"
)

// ensureTraefikBin makes sure traefik flist is mounted.
// TODO: we need to "update" traefik and restart the service
// if new version is available!
func ensureTraefikBin(ctx context.Context, cl zbus.Client) (string, error) {
	const bin = "traefik"
	flistd := stubs.NewFlisterStub(cl)
	hash, err := flistd.FlistHash(ctx, flist)
	if err != nil {
		return "", errors.Wrap(err, "failed to get traefik flist hash")
	}
	mnt, err := flistd.Mount(ctx, fmt.Sprintf("%s:%s", bin, hash), flist, pkg.ReadOnlyMountOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to mount traefik flist")
	}

	return filepath.Join(mnt, bin), nil
}
