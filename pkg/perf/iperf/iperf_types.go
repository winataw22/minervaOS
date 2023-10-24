package iperf

type iperfCommandOutput struct {
	Start     Start      `json:"start"`
	Intervals []Interval `json:"intervals"`
	End       End        `json:"end"`
	Error     string     `json:"error"`
}

type End struct {
	Streams               []EndStream           `json:"streams"`
	SumSent               Sum                   `json:"sum_sent"`
	SumReceived           Sum                   `json:"sum_received"`
	CPUUtilizationPercent CPUUtilizationPercent `json:"cpu_utilization_percent"`
	SenderTCPCongestion   string                `json:"sender_tcp_congestion"`
	ReceiverTCPCongestion string                `json:"receiver_tcp_congestion"`
}

type CPUUtilizationPercent struct {
	HostTotal    float64 `json:"host_total"`
	HostUser     float64 `json:"host_user"`
	HostSystem   float64 `json:"host_system"`
	RemoteTotal  float64 `json:"remote_total"`
	RemoteUser   float64 `json:"remote_user"`
	RemoteSystem float64 `json:"remote_system"`
}

type EndStream struct {
	Sender   Sum    `json:"sender"`
	Receiver Sum    `json:"receiver"`
	UDP      UDPSum `json:"udp"`
}

type UDPSum struct {
	Socket        int64   `json:"socket"`
	Start         float64 `json:"start"`
	End           float64 `json:"end"`
	Seconds       float64 `json:"seconds"`
	Bytes         int64   `json:"bytes"`
	BitsPerSecond float64 `json:"bits_per_second"`
	JitterMS      float64 `json:"jitter_ms"`
	LostPackets   int64   `json:"lost_packets"`
	Packets       int64   `json:"packets"`
	LostPercent   float64 `json:"lost_percent"`
	OutOfOrder    int64   `json:"out_of_order"`
	Sender        bool    `json:"sender"`
}

type Sum struct {
	Socket        int64   `json:"socket"`
	Start         float64 `json:"start"`
	End           float64 `json:"end"`
	Seconds       float64 `json:"seconds"`
	Bytes         int64   `json:"bytes"`
	BitsPerSecond float64 `json:"bits_per_second"`
	Retransmits   int64   `json:"retransmits"`
	MaxSndCwnd    int64   `json:"max_snd_cwnd"`
	MaxSndWnd     int64   `json:"max_snd_wnd"`
	MaxRtt        int64   `json:"max_rtt"`
	MinRtt        int64   `json:"min_rtt"`
	MeanRtt       int64   `json:"mean_rtt"`
	Sender        bool    `json:"sender"`
}

type Interval struct {
	Streams []IntervalStream `json:"streams"`
	Sum     Sum              `json:"sum"`
}

type IntervalStream struct {
	Socket        int64   `json:"socket"`
	Start         float64 `json:"start"`
	End           float64 `json:"end"`
	Seconds       float64 `json:"seconds"`
	Bytes         int64   `json:"bytes"`
	BitsPerSecond float64 `json:"bits_per_second"`
	Retransmits   int64   `json:"retransmits"`
	SndCwnd       int64   `json:"snd_cwnd"`
	SndWnd        int64   `json:"snd_wnd"`
	Rtt           int64   `json:"rtt"`
	Rttvar        int64   `json:"rttvar"`
	Pmtu          int64   `json:"pmtu"`
	Omitted       bool    `json:"omitted"`
	Sender        bool    `json:"sender"`
}

type Start struct {
	Connected     []Connected  `json:"connected"`
	Version       string       `json:"version"`
	SystemInfo    string       `json:"system_info"`
	Timestamp     Timestamp    `json:"timestamp"`
	ConnectingTo  ConnectingTo `json:"connecting_to"`
	Cookie        string       `json:"cookie"`
	TCPMssDefault int64        `json:"tcp_mss_default"`
	TargetBitrate int64        `json:"target_bitrate"`
	FqRate        int64        `json:"fq_rate"`
	SockBufsize   int64        `json:"sock_bufsize"`
	SndbufActual  int64        `json:"sndbuf_actual"`
	RcvbufActual  int64        `json:"rcvbuf_actual"`
	TestStart     TestStart    `json:"test_start"`
}

type Connected struct {
	Socket     int64  `json:"socket"`
	LocalHost  string `json:"local_host"`
	LocalPort  int64  `json:"local_port"`
	RemoteHost string `json:"remote_host"`
	RemotePort int64  `json:"remote_port"`
}

type ConnectingTo struct {
	Host string `json:"host"`
	Port int64  `json:"port"`
}

type TestStart struct {
	Protocol      string `json:"protocol"`
	NumStreams    int64  `json:"num_streams"`
	Blksize       int64  `json:"blksize"`
	Omit          int64  `json:"omit"`
	Duration      int64  `json:"duration"`
	Bytes         int64  `json:"bytes"`
	Blocks        int64  `json:"blocks"`
	Reverse       int64  `json:"reverse"`
	Tos           int64  `json:"tos"`
	TargetBitrate int64  `json:"target_bitrate"`
	Bidir         int64  `json:"bidir"`
	Fqrate        int64  `json:"fqrate"`
}

type Timestamp struct {
	Time     string `json:"time"`
	Timesecs int64  `json:"timesecs"`
}
