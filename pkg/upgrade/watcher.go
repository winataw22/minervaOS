package upgrade

import "time"

import "github.com/blang/semver"

import "context"

import "github.com/rs/zerolog/log"

import "path"

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
	flistInfo
}

//EventType of the event
func (f *FListEvent) EventType() EventType {
	return FList
}

//Fqdn return full name of the flist (with repo)
func (f *FListEvent) Fqdn() string {
	return path.Join(f.Repository, f.Name)
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

	client hubClient
}

// Watch an flist change in version
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
			flistInfo: info,
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
				ch <- &FListEvent{
					flistInfo: info,
				}
				w.Current = version
			}
		}
	}()

	return ch, nil
}
