package rrd

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAddSlot(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 1 * time.Minute
	db, err := NewRRDBolt(path, window, 10*time.Minute)
	require.NoError(err)

	now := time.Now()
	slot, err := db.Slot()
	require.NoError(err)

	w := uint64(window / time.Second) // window in seconds
	require.Equal((uint64(now.Unix())/w)*w, slot.Key())

	err = slot.Counter("test-1", 120)
	require.NoError(err)
}

func TestCountersTwoValues(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 1 * time.Minute
	db, err := newRRDBolt(path, window, 10*time.Minute)
	require.NoError(err)

	now := time.Now()
	slot1, err := db.slotAt(uint64(now.Add(-time.Minute).Unix()))
	require.NoError(err)

	slotNow, err := db.slotAt(uint64(now.Unix()))
	require.NoError(err)

	err = slot1.Counter("test-1", 100)
	require.NoError(err)

	err = slotNow.Counter("test-1", 120)
	require.NoError(err)

	counters, err := db.Counters(now.Add(-5 * time.Minute))
	require.NoError(err)
	require.Len(counters, 1)

	require.EqualValues(20, counters["test-1"])
}

func TestCountersSeries(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 1 * time.Minute
	// note retention is only 10 minutes
	db, err := newRRDBolt(path, window, 10*time.Minute)
	require.NoError(err)

	over := 20
	now := time.Now()
	first := now.Add(-time.Duration(over) * time.Minute)

	// we insert values over the last 20 minutes
	for i := 0; i <= over; i++ {
		slot, err := db.slotAt(uint64(first.Add(time.Duration(i) * time.Minute).Unix()))
		require.NoError(err)

		require.NoError(slot.Counter("test-1", float64(i)))
	}

	// i get all the values over the last 10 minutes
	counters, err := db.Counters(now.Add(-10 * time.Minute))
	require.NoError(err)
	require.Len(counters, 1)
	require.EqualValues(10, counters["test-1"])
}

func TestCountersRandomIncrese(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 1 * time.Minute
	// note retention is only 10 minutes
	db, err := newRRDBolt(path, window, 10*time.Minute)
	require.NoError(err)

	over := 5
	now := time.Now()
	//slot, _ := db.slotAt(uint64(now.Add(-5 * time.Microsecond).Unix()))

	first := now.Add(-time.Duration(over) * time.Minute)

	// we insert values over the last 20 minutes
	var expected int64
	for i := 0; i <= over; i++ {
		slot, err := db.slotAt(uint64(first.Add(time.Duration(i) * time.Minute).Unix()))
		require.NoError(err)
		v := rand.Int63n(10)
		if i != 0 {
			expected += v
		}
		require.NoError(slot.Counter("test-1", float64(expected)))
	}

	// i get all the values over the last 10 minutes
	counters, err := db.Counters(now.Add(-10 * time.Minute))
	require.NoError(err)
	require.Len(counters, 1)

	// we go up by one for each slot. so query last 10 blocks should return 10
	require.EqualValues(expected, counters["test-1"])
}

func TestCountersGap(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 1 * time.Minute
	db, err := newRRDBolt(path, window, 10*time.Minute)
	require.NoError(err)

	now := time.Now()
	slot1, err := db.slotAt(uint64(now.Add(3 - time.Minute).Unix()))
	require.NoError(err)

	slotNow, err := db.slotAt(uint64(now.Unix()))
	require.NoError(err)

	err = slot1.Counter("test-1", 100)
	require.NoError(err)

	err = slotNow.Counter("test-1", 120)
	require.NoError(err)

	counters, err := db.Counters(now.Add(-5 * time.Minute))
	require.NoError(err)
	require.Len(counters, 1)

	require.EqualValues(20, counters["test-1"])
}

func TestCountersRetention(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 1 * time.Minute
	// note retention is only 10 minutes
	db, err := newRRDBolt(path, window, 10*time.Minute)
	require.NoError(err)

	over := 20
	now := time.Now()
	first := now.Add(-time.Duration(over) * time.Minute)

	// we insert values over the last 20 minutes
	for i := 0; i <= over; i++ {
		slot, err := db.slotAt(uint64(first.Add(time.Duration(i) * time.Minute).Unix()))
		require.NoError(err)

		require.NoError(slot.Counter("test-1", float64(i)))
	}

	slots, err := db.Slots()
	require.NoError(err)

	require.Len(slots, 10)
	require.EqualValues((now.Add(-9*time.Minute).Unix()/60)*60, slots[0])
}

func TestCountersLast(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 1 * time.Minute
	db, err := newRRDBolt(path, window, 10*time.Minute)
	require.NoError(err)

	_, ok, err := db.Last("test-1")
	require.NoError(err)
	require.False(ok)

	now := time.Now()
	slot1, err := db.slotAt(uint64(now.Add(-5 * time.Minute).Unix()))
	require.NoError(err)

	slotNow, err := db.slotAt(uint64(now.Add(-2 * time.Minute).Unix()))
	require.NoError(err)

	err = slot1.Counter("test-1", 100)
	require.NoError(err)

	err = slotNow.Counter("test-1", 120)
	require.NoError(err)

	v, ok, err := db.Last("test-1")
	require.NoError(err)
	require.True(ok)
	require.EqualValues(120, v)

}

func TestCountersMultipleReports(t *testing.T) {
	require := require.New(t)
	path := filepath.Join(os.TempDir(), fmt.Sprint(rand.Int63()))
	defer os.RemoveAll(path)

	window := 5 * time.Minute
	db, err := newRRDBolt(path, window, 24*time.Hour)
	require.NoError(err)
	now := time.Now()

	lastReportTime := now.Unix()
	slot, err := db.slotAt(uint64(now.Add(-5 * time.Minute).Unix()))
	require.NoError(slot.Counter("test-0", 0))
	require.NoError(err)

	total := 0.0
	for i := 0; i <= 24; i++ {
		slot, err := db.slotAt(uint64(now.Add(time.Duration(i) * 5 * time.Minute).Unix()))
		require.NoError(err)
		if i%6 == 0 && i != 0 {
			counters, err := db.Counters(time.Unix(lastReportTime, 0))
			require.NoError(err)
			require.Len(counters, 1)
			lastReportTime = int64(slot.Key()) // now.Add(time.Duration(i) * 5 * time.Minute).Unix()
			require.EqualValues(6, counters["test-0"])
			total += counters["test-0"]
		}
		err = slot.Counter("test-0", float64(i)+1)
		require.NoError(err)

	}

	require.EqualValues(24, total)
}
