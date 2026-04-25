package model

import "time"

// NodeType represents the PassWall2 node type (determines which core to use).
type NodeType string

const (
	NodeTypeXray        NodeType = "Xray"
	NodeTypeSingBox     NodeType = "sing-box"
	NodeTypeShadowsocks NodeType = "Shadowsocks"
	NodeTypeHysteria2   NodeType = "hysteria2"
	NodeTypeTUIC        NodeType = "TUIC"
	NodeTypeNaiveProxy  NodeType = "naiveproxy"
)

// Protocol is the wire protocol used by a proxy node.
type Protocol string

const (
	ProtoVMess       Protocol = "vmess"
	ProtoVLESS       Protocol = "vless"
	ProtoTrojan      Protocol = "trojan"
	ProtoShadowsocks Protocol = "shadowsocks"
	ProtoHTTP        Protocol = "http"
	ProtoSOCKS       Protocol = "socks"
	ProtoHysteria2   Protocol = "hysteria2"
	ProtoTUIC        Protocol = "tuic"
	ProtoNaive       Protocol = "naive"
	ProtoWireGuard   Protocol = "wireguard"
	ProtoSSH         Protocol = "ssh"
	ProtoAnyTLS      Protocol = "anytls"
)

// ProxyNode is the normalized representation of a PassWall2 proxy node,
// regardless of whether it's configured as Xray-type or sing-box-type in UCI.
type ProxyNode struct {
	Name       string
	Type       NodeType
	Protocol   Protocol
	Address    string
	Port       int
	UUID       string // vmess/vless
	Password   string // trojan/shadowsocks/hysteria2
	Username   string // http/socks/naive
	Security   string // vmess encryption
	Flow       string // vless flow control (xtls-rprx-vision)
	TLS        bool
	Reality    bool
	RealityPublicKey string
	RealityShortID   string
	Transport  string // tcp/ws/grpc/httpupgrade/quic
	WSHost     string
	WSPath     string
	SNI        string
	ALPN       []string
	Fingerprint string
	Insecure    bool
	// Sing-box specific
	Hysteria2UpMbps     int
	Hysteria2DownMbps   int
	Hysteria2ObfsType   string
	Hysteria2ObfsPass   string
	Hysteria2AuthPass   string
	// TUIC specific
	TUICCongestionControl string
	TUICUDPMode           string
	// Shadowsocks
	SSMethod string
	// gRPC
	GRPCServiceName string
	// AnyTLS specific
	AnyTLSIdleCheckInterval string
	AnyTLSIdleTimeout       string
	AnyTLSMinIdleSession    int
	// Assigned during shadow config generation
	SOCKS5Port int
}

// LatencyClass represents the classification of latency measurements.
type LatencyClass string

const (
	LatencyExcellent LatencyClass = "excellent"
	LatencyGood      LatencyClass = "good"
	LatencyModerate  LatencyClass = "moderate"
	LatencyPoor      LatencyClass = "poor"
)

// ClassifyLatency returns the latency class for a given duration.
func ClassifyLatency(d time.Duration) LatencyClass {
	switch {
	case d < 90*time.Millisecond:
		return LatencyExcellent
	case d < 150*time.Millisecond:
		return LatencyGood
	case d < 250*time.Millisecond:
		return LatencyModerate
	default:
		return LatencyPoor
	}
}

// LatencyResult holds the results of latency probing.
type LatencyResult struct {
	Min    time.Duration
	Max    time.Duration
	Avg    time.Duration
	Median time.Duration
	Loss   float64 // 0.0 - 1.0
	Class  LatencyClass
}

// IPInfo holds exit IP information from a proxy node.
type IPInfo struct {
	IP      string
	Country string
	City    string
	ISP     string
	ASN     string
	Colo    string // Cloudflare colo code
	Hosting bool   // true if datacenter IP
	Source  string // which provider returned this info
}

// DNSLeakResult holds DNS leak detection results.
type DNSLeakResult struct {
	ProxyResolvedIPs  []string
	DirectResolvedIPs []string
	LeakDetected      bool
	LeakCount         int
}

// StreamingResult holds streaming/geo-unlock test results.
type StreamingResult struct {
	Google   string // "YES", "NO", "MAYBE", "ERROR"
	GitHub   string
	Netflix  string
	ChatGPT  string
	YouTube  string
	Disney   string
	Bilibili string
}

// TestResult aggregates all probe results for a single proxy node.
type TestResult struct {
	Node          ProxyNode
	Latency       LatencyResult
	ExitIP        IPInfo
	DNSLeak       DNSLeakResult
	Streaming     StreamingResult
	UDPOK         bool
	UDPError      string
	Errors        []string // non-fatal error messages
	TotalDuration time.Duration
}

// TestConfig holds testing parameters.
type TestConfig struct {
	Concurrency       int
	Timeout           time.Duration
	WarmupProbes      int
	MeasurementProbes int
}

// DefaultTestConfig returns sensible defaults.
func DefaultTestConfig() TestConfig {
	return TestConfig{
		Concurrency:       10,
		Timeout:           5 * time.Second,
		WarmupProbes:      3,
		MeasurementProbes: 5,
	}
}

// UserConfig is the ~/.inice.yaml configuration.
type UserConfig struct {
	Router struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		KeyFile  string `mapstructure:"key_file"`
	} `mapstructure:"router"`
	Shadow struct {
		PreferredCore string `mapstructure:"preferred_core"` // auto, sing-box, xray
		BasePort      int    `mapstructure:"base_port"`
	} `mapstructure:"shadow"`
	Testing struct {
		Concurrency       int    `mapstructure:"concurrency"`
		Timeout           string `mapstructure:"timeout"`
		WarmupProbes      int    `mapstructure:"warmup_probes"`
		MeasurementProbes int    `mapstructure:"measurement_probes"`
	} `mapstructure:"testing"`
	Output struct {
		Mode   string `mapstructure:"mode"`   // static, tui
		Format string `mapstructure:"format"` // table, json, csv
	} `mapstructure:"output"`
}

// DefaultUserConfig returns sensible defaults for UserConfig.
func DefaultUserConfig() UserConfig {
	cfg := UserConfig{}
	cfg.Router.Port = 22
	cfg.Router.User = "root"
	cfg.Shadow.PreferredCore = "auto"
	cfg.Shadow.BasePort = 20000
	cfg.Testing.Concurrency = 10
	cfg.Testing.Timeout = "10s"
	cfg.Testing.WarmupProbes = 3
	cfg.Testing.MeasurementProbes = 5
	cfg.Output.Mode = "static"
	cfg.Output.Format = "table"
	return cfg
}
