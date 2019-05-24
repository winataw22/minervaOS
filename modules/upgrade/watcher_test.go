package upgrade

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testPublisher struct {
	i        int
	upgrades []Upgrade
	err      error
}

func (p *testPublisher) Check() (Upgrade, error) {
	if p.err != nil {
		return Upgrade{}, p.err
	}

	if p.i >= len(p.upgrades) {
		return Upgrade{}, ErrNoUpgrade
	}

	u := p.upgrades[p.i]
	p.i++
	return u, nil
}

// TestWatcher validate the normal behavior of a Watcher
func TestWatcher(t *testing.T) {
	expected := []Upgrade{
		{
			Flist:         "https://hub.grid.tf/tf-official-apps/zos_upgrade_0.1.0.flist",
			Signature:     "6c8c30810bc909cda71497a90475d79b",
			TransactionID: "cff2bbb909b2596cbf626c351c4969e8f6194e4e5528f4714efb30657a21df3c",
		},
		{
			Flist:         "https://hub.grid.tf/tf-official-apps/zos_upgrade_0.1.1.flist",
			Signature:     "e116a9b084dca2b73aac6caf06ad2eaf",
			TransactionID: "0c4b28cda1164fe4fd47eaa3baca804627bd4c551f4955f610ee2148778f758d",
		},
		{
			Flist:         "https://hub.grid.tf/tf-official-apps/zos_upgrade_0.1.2.flist",
			Signature:     "d14e26231ff91b4ab42670ab94550674",
			TransactionID: "ea63c284be59af68074eae2bedbe7a690c0c53a1b6d011b5fddd18da79fdb545",
		},
	}
	p := &testPublisher{upgrades: expected}
	w := NewWatcher(time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	actual := []Upgrade{}
	cUpgrade := w.Watch(ctx, p)
	for upgrade := range cUpgrade {
		actual = append(actual, upgrade)
	}

	assert.NoError(t, w.Error())
	assert.Equal(t, expected, actual)
}

// TestWatcherError validates any error happening in the wacher goroutine is
// returned by the Error() method
func TestWatcherError(t *testing.T) {
	p := &testPublisher{err: fmt.Errorf("test error")}
	w := NewWatcher(time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	actual := []Upgrade{}
	cUpgrade := w.Watch(ctx, p)
	for upgrade := range cUpgrade {
		actual = append(actual, upgrade)
	}

	assert.Error(t, w.Error())
}
