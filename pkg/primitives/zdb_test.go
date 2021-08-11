package primitives

// import (
// 	"net"
// 	"testing"
// )

// func Test_isYgg(t *testing.T) {
// 	type args struct {
// 		ip net.IP
// 	}
// 	tests := []struct {
// 		name string
// 		args args
// 		want bool
// 	}{
// 		{
// 			name: "301:8d90:378f:a93:6364:f8c8:1c6b:865d",
// 			args: args{
// 				ip: net.ParseIP("301:8d90:378f:a93:6364:f8c8:1c6b:865d"),
// 			},
// 			want: true,
// 		},
// 		{
// 			name: "2a02:2788:864:1314:9eb6:d0ff:fe97:764b",
// 			args: args{
// 				ip: net.ParseIP("2a02:2788:864:1314:9eb6:d0ff:fe97:764b"),
// 			},
// 			want: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			if got := isYgg(tt.args.ip); got != tt.want {
// 				t.Errorf("isYgg() = %v, want %v", got, tt.want)
// 			}
// 		})
// 	}
// }
