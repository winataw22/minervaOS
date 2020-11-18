package upgrade

import (
	"context"
	"time"

	"github.com/blang/semver"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

/**
Watcher unifies the upgrade pipeline by making sure we can watch
different types of required updates, while always make sure only
one kind of update is applied at a time.

This to prevent updates to step on each other toes.
*/

//EventType of the watcher
type EventType string

const (
	//FList event type
	FList EventType = "flist"
	//Repo event type
	Repo EventType = "repo"
)

//Event interface
type Event interface {
	EventType() EventType
}

//Watcher interface
type Watcher interface {
	Watch(ctx context.Context) (<-chan Event, error)
}

//FListEvent struct
type FListEvent struct {
	FullFListInfo
}

//EventType of the event
func (f *FListEvent) EventType() EventType {
	return FList
}

//TryVersion will try to parse the version from flist
//other wise return empty version 0.0.0
func (f *FListEvent) TryVersion() semver.Version {
	version, _ := f.Version()
	return version
}

//FListSemverWatcher watches a single FList for changes in it's semver
//the semver to change without the flist name itself changes, means
//that this flist is mostly a symlink
type FListSemverWatcher struct {
	FList    string
	Duration time.Duration
	Current  semver.Version

	client HubClient
}

var _ Watcher = &FListSemverWatcher{}

// Watch an flist change in version
// The Event returned by the channel is of concrete type FListEvent
func (w *FListSemverWatcher) Watch(ctx context.Context) (<-chan Event, error) {
	if w.Duration == time.Duration(0) {
		//default delay of 5min
		w.Duration = 600 * time.Second
	}

	info, err := w.client.Info(w.FList)
	if err != nil {
		return nil, err
	}

	version, err := info.Version()
	if err != nil {
		return nil, err
	}

	ch := make(chan Event, 1)

	if version.GT(w.Current) {
		ch <- &FListEvent{
			FullFListInfo: info,
		}
		w.Current = version
	}

	ticker := time.NewTicker(w.Duration)

	go func() {
		defer ticker.Stop()
		defer close(ch)

		for {
			// wait for next tick
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}

			log.Debug().Str("flist", w.FList).Msg("check updates")

			info, err := w.client.Info(w.FList)
			if err != nil {
				log.Error().Err(err).Str("flist", w.FList).Msg("failed to get flist info")
				continue
			}
			version, err := info.Version()
			if err != nil {
				log.Error().Err(err).Str("flist", w.FList).Msg("failed to get flist version")
				continue
			}

			if version.GT(w.Current) {
				select {
				case ch <- &FListEvent{
					FullFListInfo: info,
				}:
				case <-ctx.Done():
					return
				}

				w.Current = version
			}
		}
	}()

	return ch, nil
}

//RepoEvent is returned by the repo watcher
type RepoEvent struct {
	Repo  string
	ToAdd []FListInfo
	ToDel []FListInfo
}

//EventType returns event type
func (e *RepoEvent) EventType() EventType {
	return Repo
}

//FListRepoWatcher type
type FListRepoWatcher struct {
	Repo     string
	Current  map[string]FListInfo
	Duration time.Duration

	client HubClient
}

func (w *FListRepoWatcher) list() (map[string]FListInfo, error) {
	packages, err := w.client.List(w.Repo)
	if err != nil {
		return nil, err
	}

	result := make(map[string]FListInfo)
	for _, pkg := range packages {
		//flist := FListInfo{pkg}
		result[pkg.Fqdn()] = pkg
	}

	return result, nil
}

func (w *FListRepoWatcher) diff(packages map[string]FListInfo) (toAdd, toDel []FListInfo) {
	for name, pkg := range packages {
		current, ok := w.Current[name]
		if !ok || pkg.Updated != current.Updated {
			toAdd = append(toAdd, pkg)
		}
	}

	for name := range w.Current {
		current, ok := packages[name]
		if !ok {
			toDel = append(toDel, current)
		}
	}

	return
}

// Diff return the remote changes related to current list of packages
func (w *FListRepoWatcher) Diff() (all map[string]FListInfo, toAdd, toDell []FListInfo, err error) {
	all, err = w.list()
	if err != nil {
		return all, nil, nil, errors.Wrap(err, "failed to get available packages")
	}

	toAdd, toDell = w.diff(all)
	return
}

// Watch watches a full repo for changes. Event is always of concrete type RepoEvent
func (w *FListRepoWatcher) Watch(ctx context.Context) (<-chan Event, error) {
	if w.Duration == time.Duration(0) {
		//default delay of 5min
		w.Duration = 600 * time.Second
	}

	packages, toAdd, toDel, err := w.Diff()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get available packages")
	}

	ch := make(chan Event, 1)

	if len(toAdd) > 0 || len(toDel) > 0 {
		ch <- &RepoEvent{
			Repo:  w.Repo,
			ToAdd: toAdd,
			ToDel: toDel,
		}

		w.Current = packages
	}

	ticker := time.NewTicker(w.Duration)

	go func() {
		defer close(ch)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}

			packages, toAdd, toDel, err := w.Diff()
			if err != nil {
				log.Error().Err(err).Str("repo", w.Repo).Msg("failed to list repo flists")
				continue
			}

			if len(toAdd) > 0 || len(toDel) > 0 {
				select {
				case ch <- &RepoEvent{
					Repo:  w.Repo,
					ToAdd: toAdd,
					ToDel: toDel,
				}:
				case <-ctx.Done():
					return
				}

				w.Current = packages
			}
		}

	}()

	return ch, nil
}
