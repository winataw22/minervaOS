package upgrade

import (
	"github.com/pkg/errors"
)

/**
Watcher unifies the upgrade pipeline by making sure we can watch
different types of required updates, while always make sure only
one kind of update is applied at a time.

This to prevent updates to step on each other toes.
*/

// FListRepo type
type FListRepo struct {
	Repo    string
	Current map[string]FListInfo

	client HubClient
}

func (w *FListRepo) list() (map[string]FListInfo, error) {
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

func (w *FListRepo) diff(packages map[string]FListInfo) (toAdd, toDel []FListInfo) {
	for name, pkg := range packages {
		current, ok := w.Current[name]
		if !ok || pkg.Updated != current.Updated {
			toAdd = append(toAdd, pkg)
		}
	}

	for name, pkg := range w.Current {
		_, ok := packages[name]
		if !ok {
			toDel = append(toDel, pkg)
		}
	}

	return
}

// Diff return the remote changes related to current list of packages
func (w *FListRepo) Diff() (all map[string]FListInfo, toAdd, toDell []FListInfo, err error) {
	all, err = w.list()
	if err != nil {
		return all, nil, nil, errors.Wrap(err, "failed to get available packages")
	}

	toAdd, toDell = w.diff(all)
	return
}
