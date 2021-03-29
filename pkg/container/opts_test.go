package container

import "testing"

func Test_cruToLimit(t *testing.T) {
	type args struct {
		cru uint
	}
	tests := []struct {
		name       string
		args       args
		wantQuota  int64
		wantPeriod uint64
	}{
		{
			name: "1",
			args: args{
				cru: 1,
			},
			wantPeriod: 100000,
			wantQuota:  100000,
		},
		{
			name: "2",
			args: args{
				cru: 2,
			},
			wantPeriod: 100000,
			wantQuota:  200000,
		},
		{
			name: "3",
			args: args{
				cru: 3,
			},
			wantPeriod: 100000,
			wantQuota:  300000,
		},
		{
			name: "10",
			args: args{
				cru: 10,
			},
			wantPeriod: 100000,
			wantQuota:  1000000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuota, gotPeriod := cruToLimit(tt.args.cru)
			if gotQuota != tt.wantQuota {
				t.Errorf("cruToLimit() gotQuota = %v, want %v", gotQuota, tt.wantQuota)
			}
			if gotPeriod != tt.wantPeriod {
				t.Errorf("cruToLimit() gotPeriod = %v, want %v", gotPeriod, tt.wantPeriod)
			}
		})
	}
}
