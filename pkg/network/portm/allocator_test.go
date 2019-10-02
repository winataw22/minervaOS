package portm

import (
	"sync"
	"testing"

	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStore struct {
	reserved map[string]mapset.Set
	last     map[string]int
	sync.Mutex
}

func newTestStore() *testStore {
	return &testStore{
		reserved: make(map[string]mapset.Set),
		last:     make(map[string]int, 0),
	}
}

func (s *testStore) Lock() error {
	s.Mutex.Lock()
	return nil
}
func (s *testStore) Unlock() error {
	s.Mutex.Unlock()
	return nil
}
func (s *testStore) Reserve(ns string, port int) (bool, error) {
	set, ok := s.reserved[ns]
	if !ok {
		set = mapset.NewSet()
		s.reserved[ns] = set
	}

	if set.Contains(port) {
		return false, nil
	}

	s.last[ns] = port
	return set.Add(port), nil
}
func (s *testStore) Release(ns string, port int) error {
	set, ok := s.reserved[ns]
	if !ok {
		return nil
	}

	set.Remove(port)
	return nil
}

func (s *testStore) GetByNS(ns string) ([]int, error) {
	set, ok := s.reserved[ns]
	if !ok {
		return []int{}, nil
	}

	ports := set.ToSlice()
	output := make([]int, len(ports))
	for i, p := range ports {
		output[i] = p.(int)
	}

	return output, nil
}

func (s *testStore) LastReserved(ns string) (int, error) {
	port, ok := s.last[ns]
	if !ok {
		return -1, nil
	}
	return port, nil
}

func (s *testStore) Close() error {
	return nil
}

func TestReserve(t *testing.T) {
	store := newTestStore()
	pRange := PortRange{
		Start: 1000,
		End:   6000,
	}
	ns := "ns"
	alloc := NewAllocator(pRange, store)

	p1, err := alloc.Reserve(ns)
	require.NoError(t, err)
	assert.True(t, store.reserved[ns].Contains(p1))
	assert.True(t, p1 >= pRange.Start)
	assert.True(t, p1 <= pRange.End)

	p2, err := alloc.Reserve(ns)
	require.NoError(t, err)
	assert.True(t, store.reserved[ns].Contains(p2))
	assert.True(t, p1 != p2)
	assert.True(t, p2 >= pRange.Start)
	assert.True(t, p2 <= pRange.End)
}

func TestReserveConcurent(t *testing.T) {
	store := newTestStore()
	pRange := PortRange{
		Start: 1000,
		End:   6000,
	}
	ns := "ns"
	N := 5
	wg := sync.WaitGroup{}
	reserved := make([][]int, N)

	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(reserved [][]int, i int) {
			defer wg.Done()
			alloc := NewAllocator(pRange, store)
			reserved[i] = make([]int, 0, 20)

			for y := 0; y < 20; y++ {
				p, err := alloc.Reserve(ns)
				require.NoError(t, err)
				reserved[i] = append(reserved[i], p)
			}
		}(reserved, i)
	}

	wg.Wait()
	// ensure all reserved ports are unique
	allReserved := make(map[int]struct{})
	for i := 0; i < N; i++ {
		for y := 0; y < 20; y++ {
			_, exists := allReserved[reserved[i][y]]
			assert.False(t, exists, "same port should not have been reserved twice")
			allReserved[reserved[i][y]] = struct{}{}
		}
	}
}

func TestReuseReleased(t *testing.T) {
	store := newTestStore()
	pRange := PortRange{
		Start: 1000,
		End:   6000,
	}
	ns := "ns"
	alloc := NewAllocator(pRange, store)

	for i := 0; i <= 5000; i++ {
		_, err := alloc.Reserve(ns)
		require.NoError(t, err)
	}

	_, err := alloc.Reserve(ns)
	assert.Equal(t, err, ErrNoFreePort)

	err = alloc.Release(ns, 1000)
	require.NoError(t, err)

	port, err := alloc.Reserve(ns)
	require.NoError(t, err)
	assert.Equal(t, 1000, port)
}

func TestRelease(t *testing.T) {
	store := newTestStore()
	store.Reserve("ns", 1000)
	store.Reserve("ns", 1001)

	pRange := PortRange{
		Start: 1000,
		End:   6000,
	}

	alloc := NewAllocator(pRange, store)

	err := alloc.Release("ns", 1000)
	require.NoError(t, err)
	assert.False(t, store.reserved["ns"].Contains(1000))
}

func BenchmarkReserve(b *testing.B) {
	store := newTestStore()
	pRange := PortRange{
		Start: 1000,
		End:   6000,
	}
	alloc := NewAllocator(pRange, store)

	for i := 0; i < b.N; i++ {
		port, err := alloc.Reserve("ns")
		if err == ErrNoFreePort {
			break
		}
		require.NoError(b, err)
		_ = port
	}
}
