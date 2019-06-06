package upgrade

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/blang/semver"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zosv2/modules"
)

type HookType string

const (
	hookPreCopy   HookType = "pre-copy"
	hookPostCopy  HookType = "post-copy"
	hookMigrate   HookType = "migrate"
	hookPostStart HookType = "post-start"
)

type UpgradeModule struct {
	root    string
	version semver.Version
	flister modules.Flister
}

func New(root string, flister modules.Flister) (*UpgradeModule, error) {
	if err := os.MkdirAll(root, 0770); err != nil {
		return nil, err
	}

	var version semver.Version
	versionPath := filepath.Join(root, "version")
	version, err := readVersion(versionPath)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error().Err(err).Msg("read version")
			return nil, err
		}
		log.Info().Msg("no version found, assuming fresh install")
		// the file doesn't exist yet. So we are on a fresh system
		version = semver.MustParse("0.0.1")
		if err := writeVersion(versionPath, version); err != nil {
			log.Error().Err(err).Msg("fail to write version")
			return nil, err
		}
	}

	log.Info().Msgf("current version %s", version.String())
	return &UpgradeModule{
		version: version,
		root:    root,
		flister: flister,
	}, nil
}

// FIXME: not sure about the public interface of this package yet
// most probably will need to run as a daemon too
func (u *UpgradeModule) Run(period time.Duration, p Publisher) error {
	ticker := time.NewTicker(period)

	for range ticker.C {
		ok, latest, err := isNewVersionAvailable(u.version, p)
		if err != nil || !ok {
			continue
		}

		toApply, err := versionsToApply(u.version, latest, p)
		if err != nil {
			continue
		}

		for _, version := range toApply {
			upgrade, err := p.Get(version)
			if err != nil {
				log.Error().
					Err(err).
					Str("version", version.String()).
					Msg("fail to retrieve upgrade from publisher")
				break
			}

			log.Info().
				Str("curent version", u.version.String()).
				Str("new version", version.String()).
				Msg("start upgrade")

			if err := u.applyUpgrade(upgrade); err != nil {
				log.Error().
					Err(err).
					Str("version", version.String()).
					Msg("fail to apply upgrade")
				break
			}

			u.version = version
			if err := writeVersion(filepath.Join(u.root, "version"), version); err != nil {
				log.Error().
					Err(err).
					Str("version", version.String()).
					Msg("fail to write version to disks")
			}
		}
	}

	return nil
}

func isNewVersionAvailable(current semver.Version, p Publisher) (bool, semver.Version, error) {
	latest, err := p.Latest()
	if err != nil {
		log.Error().
			Err(err).
			Msg("fail to get latest version from publisher")
		return false, latest, err
	}

	if current.Equals(latest) {
		log.Info().
			Str("version", current.String()).
			Msg("current and latest version match, nothing to do")
		return false, latest, nil
	}
	if current.GT(latest) {
		log.Warn().
			Str("current version", current.String()).
			Str("latest version", latest.String()).
			Msg("current version is higher then latest reported by publisher")
		return false, latest, nil
	}

	log.Info().
		Str("current version", current.String()).
		Str("new version", latest.String()).
		Msg("new version available")
	return true, latest, nil
}

func versionsToApply(current, latest semver.Version, p Publisher) ([]semver.Version, error) {
	versions, err := p.List()
	if err != nil {
		log.Error().
			Err(err).
			Msg("fail to list available version from publisher")
		return nil, err
	}

	latestFound := false
	toApply := []semver.Version{}
	for _, v := range versions {
		if v.Equals(latest) {
			latestFound = true
		}
		// if the v is a higher version as the current version
		if current.Compare(v) < 0 {
			toApply = append(toApply, v)
		}
	}
	if !latestFound {
		return nil, fmt.Errorf("latest version has not been found in available versions of the publisher")
	}

	return toApply, nil
}

func (u *UpgradeModule) applyUpgrade(upgrade Upgrade) error {

	log.Info().Str("flist", upgrade.Flist).Msg(("start applying upgrade"))

	path, err := u.flister.Mount(upgrade.Flist, upgrade.Storage)
	if err != nil {
		return err
	}
	defer func() {
		if err := u.flister.Umount(path); err != nil {
			log.Error().Err(err).Msgf("fail to umount flist at %s: %v", path, err)
		}
	}()

	// tx := beginTransaction()

	if err := executeHook(filepath.Join(path, string(hookPreCopy))); err != nil {
		log.Error().Err(err).Msg("fail to execute pre-copy script")
		// u.rollback()
	}

	// copy file from path to /
	// TODO: what upgrade that fails mid way ?
	if err := mergeFs(path, "/"); err != nil {
		return err
	}
	log.Info().Str("flist", upgrade.Flist).Msg(("upgrade applied"))

	if err := executeHook(filepath.Join(path, string(hookPostCopy))); err != nil {
		log.Error().Err(err).Msg("fail to execute post-copy script")
		// u.rollback()
	}

	// zinit.Stop() stop services

	if err := executeHook(filepath.Join(path, string(hookMigrate))); err != nil {
		log.Error().Err(err).Msg("fail to execute migrate script")
		// u.rollback()
	}

	// zinit.Start() restart stopped/new services

	if err := executeHook(filepath.Join(path, string(hookPostStart))); err != nil {
		log.Error().Err(err).Msg("fail to execute post-start script")
		// u.rollback()
	}

	// tx.Commit()
	return nil
	// TODO:
	// identify which module has been updated
	// if present call migration
	// restart the required module
}

func executeHook(path string) error {
	name := filepath.Base(path)

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			log.Info().Str("script", path).Msgf("%s upgrade hook script not found, skipping", name)
			return nil
		}
		return err
	}

	if !IsExecutable(info.Mode().Perm()) {
		return fmt.Errorf("%s exists but is not executable", name)
	}

	cmd := exec.Command(path)
	err = cmd.Run()
	if err != nil {
		return err
	}
	log.Info().Str("script", path).Msgf("%s upgrade hook script executed", name)

	return nil
}

func mergeFs(upgradeRoot, fsRoot string) error {

	skippingFiles := []string{
		fmt.Sprintf("/%s", string(hookPreCopy)),
		fmt.Sprintf("/%s", string(hookPostCopy)),
		fmt.Sprintf("/%s", string(hookMigrate)),
		fmt.Sprintf("/%s", string(hookPostStart)),
	}

	return filepath.Walk(upgradeRoot, func(path string, info os.FileInfo, err error) error {
		// trim flist mountpoint from flist path
		destPath := ""
		if path == upgradeRoot {
			destPath = "/"
		} else {
			destPath = path[len(upgradeRoot):]
			if destPath[0] != filepath.Separator {
				destPath = fmt.Sprintf("/%s", path)
			}
		}

		// don't copy hook scripts
		if isIn(destPath, skippingFiles) {
			return nil
		}

		// change root
		p, err := changeRoot(fsRoot, destPath)
		if err != nil {
			return err
		}
		// create directories
		if info.IsDir() {
			if err := os.MkdirAll(p, info.Mode().Perm()); err != nil {
				return err
			}
			return nil
		}

		// upgrade flist should only container directory and regular files
		if !info.Mode().IsRegular() {
			log.Printf("skip %s: not a regular file", path)
			return nil
		}

		// copy the file to final destination
		return copyFile(p, path, info.Mode().Perm())
	})
}

func isIn(target string, list []string) bool {
	for _, x := range list {
		if target == x {
			return true
		}
	}
	return false
}

func copyFile(dst, src string, perm os.FileMode) error {
	fSrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fSrc.Close()

	fDst, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_SYNC, perm)
	if err != nil {
		return err
	}
	defer fDst.Close()

	_, err = io.Copy(fDst, fSrc)
	return err
}

var errNotAbsolute = errors.New("path is not absolute")

// changeRoot changes the root of path by base
// both base and path needs to be absolute
func changeRoot(base, path string) (string, error) {
	if !filepath.IsAbs(base) {
		return "", errNotAbsolute
	}
	if !filepath.IsAbs(path) {
		return "", errNotAbsolute
	}

	ss := strings.SplitN(path, string(filepath.Separator), 2)
	if len(ss) > 1 {
		return filepath.Join(base, ss[1]), nil
	}
	return base, nil
}
