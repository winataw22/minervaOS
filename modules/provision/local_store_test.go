package provision

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStore(t *testing.T) {
	root, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	defer os.RemoveAll(root)

	type fields struct {
		root string
	}
	type args struct {
		r *Reservation
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
				r: &Reservation{
					ID:       "r-1",
					Created:  time.Now().Add(-time.Minute).Round(time.Second),
					Duration: time.Second * 10,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &FSStore{
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
			assert.EqualValues(t, tt.args.r, actual)

			_, err = s.Get("foo")
			require.Error(t, err)

			expired, err := s.GetExpired()
			require.NoError(t, err)
			assert.Equal(t, len(expired), 1)
			assert.Equal(t, tt.args.r, expired[0])

			err = s.Remove(actual.ID)
			assert.NoError(t, err)

			_, err = s.Get(tt.args.r.ID)
			require.Error(t, err)
		})
	}
}
