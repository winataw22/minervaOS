package rrd

import (
	"encoding/binary"
	"math"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

// RRD is a round robin database of fixed size which is specified on creation
// RRD stores `counter` values. Which means values that can only go UP.
// then it's easy to compute the increase of this counter over a given window
// The database only keep history based on retention.
type RRD interface {
	// Slot returns the current window (slot) to store values.
	Slot() (Slot, error)
	// Counters, return all stored counters since the given time (since) until now.
	Counters(since time.Time) (map[string]float64, error)
	// Last returns the last reported value for a metric given the metric
	// name
	Last(key string) (value float64, ok bool, err error)
}

type Slot interface {
	// Counter sets (or overrides) the current stored value for this key,
	// with value
	Counter(key string, value float64) error
	// Key return the key of the slot which is the window timestamp
	Key() uint64
}

type rrdBolt struct {
	db        *bolt.DB
	window    uint64
	retention uint64
}

type rrdSlot struct {
	db  *bolt.DB
	key uint64
}

// NewRRDBolt creates a new rrd database that uses bolt as a storage. if window or retention are 0
// the function will panic. If retnetion is smaller then window the function will panic.
// retention and window must be multiple of 1 minute.
func NewRRDBolt(path string, window time.Duration, retention time.Duration) (RRD, error) {
	return newRRDBolt(path, window, retention)
}

func newRRDBolt(path string, window time.Duration, retention time.Duration) (*rrdBolt, error) {
	window = (window / time.Minute) * time.Minute
	retention = (retention / time.Minute) * time.Minute

	if window == 0 {
		panic("invalid window, can't be zero")
	}
	if retention == 0 {
		panic("invalid retention, can't be zero")
	}
	if retention < window {
		panic("retention can't be smaller than window")
	}

	db, err := bolt.Open(path, 0644, bolt.DefaultOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open bolt db")
	}

	return &rrdBolt{
		db:        db,
		window:    uint64(window / time.Second),
		retention: uint64(retention / time.Second),
	}, nil
}

func (r *rrdBolt) retain(now uint64) error {
	retain := now - r.retention
	return r.db.Update(func(tx *bolt.Tx) error {
		cur := tx.Cursor()

		for k, _ := cur.First(); k != nil; k, _ = cur.Next() {
			if len(k) != 8 {
				continue // uknown key size
			}

			if lu64(k) <= retain {
				// delete the bucket
				// are we sure this is safe to do while iterating ?
				_ = tx.DeleteBucket(k)
			}
		}

		return nil
	})
}

func (r *rrdBolt) Slots() ([]uint64, error) {
	var slots []uint64
	err := r.db.View(func(tx *bolt.Tx) error {
		cur := tx.Cursor()
		for k, _ := cur.First(); k != nil; k, _ = cur.Next() {
			slots = append(slots, lu64(k))
		}

		return nil
	})

	return slots, err
}

func (r *rrdBolt) Last(key string) (value float64, ok bool, err error) {
	err = r.db.View(func(tx *bolt.Tx) error {
		cur := tx.Cursor()
		for k, _ := cur.Last(); k != nil; k, _ = cur.Prev() {
			bucket := tx.Bucket(k)
			bytes := bucket.Get([]byte(key))
			if bytes != nil {
				value = lf64(bytes)
				ok = true
				break
			}
		}

		return nil
	})

	return
}

// Counters return increase in counter value since the given
// start time.
func (r *rrdBolt) Counters(since time.Time) (map[string]float64, error) {
	ts := uint64(since.Unix())
	ts = (ts / r.window) * r.window

	// we start from the previous slot so we check from the last value.
	//ts -= r.window
	change := make(map[string]float64)

	err := r.db.View(func(tx *bolt.Tx) error {
		last := make(map[string]float64)
		first := true
		cur := tx.Cursor()
		for k, _ := cur.Seek(u64(ts)); k != nil; k, _ = cur.Next() {
			bucket := tx.Bucket(k)
			values := bucket.Cursor()
			for k, v := values.First(); k != nil; k, v = values.Next() {
				vf := lf64(v)
				key := string(k)

				if !first {
					prev := last[key]
					if prev > vf {
						change[key] += vf
					} else {
						change[key] += vf - prev
					}
				}

				last[key] = vf
			}

			first = false
		}

		return nil
	})

	return change, err
}

func (r *rrdBolt) slotAt(ts uint64) (*rrdSlot, error) {
	ts = (ts / r.window) * r.window

	if err := r.retain(ts); err != nil {
		return nil, errors.Wrap(err, "failed to apply retnetion policy")
	}

	if err := r.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(u64(ts))
		return errors.Wrap(err, "failed to create slot bucket")

	}); err != nil {
		return nil, err
	}

	return &rrdSlot{db: r.db, key: ts}, nil
}

func (r *rrdBolt) Slot() (Slot, error) {
	ts := uint64(time.Now().Unix())
	return r.slotAt(ts)
}

func lu64(v []byte) uint64 {
	return binary.BigEndian.Uint64(v)
}

func u64(u uint64) []byte {
	var v [8]byte
	binary.BigEndian.PutUint64(v[:], u)
	return v[:]
}

func f64(f float64) []byte {
	return u64(math.Float64bits(f))
}

func lf64(v []byte) float64 {
	return math.Float64frombits(lu64(v))
}

func (r *rrdSlot) Counter(key string, value float64) error {
	return r.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket(u64(r.key))
		return bucket.Put([]byte(key), f64(value))
	})
}

func (r *rrdSlot) Key() uint64 {
	return r.key
}
