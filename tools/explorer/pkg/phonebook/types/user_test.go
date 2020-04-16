package types

import (
	"testing"

	"gotest.tools/assert"
)

func TestUser_Validate(t *testing.T) {
	tests := []struct {
		name string
		u    User
		err  string
	}{
		{
			name: "user.3bot",
			u: User{
				Name: "user.3bot",
			},
			err: "",
		},
		{
			name: "User.3bot",
			u: User{
				Name: "User.3bot",
			},
			err: "name should be all lower case",
		},
		{
			name: "lower.email",
			u: User{
				Name: "com",
				Email: "user@example.com",
			},
			err: "",
		},
		{
			name: "upper.email",
			u: User{
				Name: "com",
				Email: "User@example.com",
			},
			err: "email should be all lower case",
		},
		{
			name: "ab",
			u: User{
				Name: "ab",
			},
			err: "name should be at least 3 character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.u.Validate()
			if tt.err != "" {
				assert.Error(t, err, tt.err)
			} else {
				assert.NilError(t, err)
			}
		})
	}
}
