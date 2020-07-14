package cache

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/threefoldtech/zos/pkg/provision"
)

func TestLocalStore(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)

	type fields struct {
		root string
	}
	type args struct {
		r *provision.Reservation
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "main",
			fields: fields{
				root: root,
			},
			args: args{
				r: &provision.Reservation{
					ID:       "r-1",
					Created:  time.Now().UTC().Add(-time.Minute).Round(time.Second),
					Duration: time.Second * 10,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Fs{
				root: tt.fields.root,
			}

			err = s.Add(tt.args.r)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			actual, err := s.Get(tt.args.r.ID)
			require.NoError(t, err)
			assert.Equal(t, tt.args.r.Duration, actual.Duration)
			assert.Equal(t, tt.args.r.Created, actual.Created)
			assert.Equal(t, tt.args.r.ID, actual.ID)

			_, err = s.Get("foo")
			require.Error(t, err)

			expired, err := s.GetExpired()
			require.NoError(t, err)
			assert.Equal(t, len(expired), 1)
			assert.Equal(t, tt.args.r.Duration, expired[0].Duration)
			assert.Equal(t, tt.args.r.Created, expired[0].Created)
			assert.Equal(t, tt.args.r.ID, expired[0].ID)

			err = s.Remove(actual.ID)
			assert.NoError(t, err)

			_, err = s.Get(tt.args.r.ID)
			require.Error(t, err)
		})
	}
}
