package container

import "testing"

func Test_cruToLimit(t *testing.T) {
	type args struct {
		cru      uint
		totalCPU int
	}
	tests := []struct {
		name       string
		args       args
		wantQuota  int64
		wantPeriod uint64
	}{
		{
			name: "1-1",
			args: args{
				cru:      1,
				totalCPU: 1,
			},
			wantPeriod: 1000000,
			wantQuota:  1000000,
		},
		{
			name: "1-4",
			args: args{
				cru:      1,
				totalCPU: 4,
			},
			wantPeriod: 250000,
			wantQuota:  1000000,
		},
		{
			name: "1-6",
			args: args{
				cru:      1,
				totalCPU: 6,
			},
			wantPeriod: 166667,
			wantQuota:  1000000,
		},
		{
			name: "2-4",
			args: args{
				cru:      2,
				totalCPU: 4,
			},
			wantPeriod: 500000,
			wantQuota:  1000000,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuota, gotPeriod := cruToLimit(tt.args.cru, tt.args.totalCPU)
			if gotQuota != tt.wantQuota {
				t.Errorf("cruToLimit() gotQuota = %v, want %v", gotQuota, tt.wantQuota)
			}
			if gotPeriod != tt.wantPeriod {
				t.Errorf("cruToLimit() gotPeriod = %v, want %v", gotPeriod, tt.wantPeriod)
			}
		})
	}
}
