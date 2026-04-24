package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/txdywy/inice/internal/model"
	sshutil "github.com/txdywy/inice/internal/ssh"
)

// SingBoxCore implements ShadowCore for sing-box.
type SingBoxCore struct {
	ts string // timestamp for unique file names
}

// NewSingBoxCore creates a new SingBoxCore.
func NewSingBoxCore() *SingBoxCore {
	return &SingBoxCore{ts: strconv.FormatInt(time.Now().Unix(), 10)}
}

func (c *SingBoxCore) Type() CoreType { return CoreSingBox }

func (c *SingBoxCore) ConfigPath() string {
	return fmt.Sprintf("/tmp/inice-singbox-%s.json", c.ts)
}

func (c *SingBoxCore) LogPath() string {
	return fmt.Sprintf("/tmp/inice-singbox-%s.log", c.ts)
}

// Sing-box JSON structures
type sbConfig struct {
	Log      sbLog       `json:"log"`
	Inbounds []sbInbound `json:"inbounds"`
	Outbounds []sbOutbound `json:"outbounds"`
	Route    sbRoute     `json:"route"`
}

type sbLog struct {
	Level  string `json:"level"`
	Output string `json:"output"`
}

type sbInbound struct {
	Type        string `json:"type"`
	Tag         string `json:"tag"`
	Listen      string `json:"listen"`
	ListenPort  int    `json:"listen_port"`
	Sniff       bool   `json:"sniff"`
}

type sbOutbound struct {
	Type           string       `json:"type"`
	Tag            string       `json:"tag"`
	Server         string       `json:"server,omitempty"`
	ServerPort     int          `json:"server_port,omitempty"`
	UUID           string       `json:"uuid,omitempty"`
	Password       string       `json:"password,omitempty"`
	Username       string       `json:"username,omitempty"`
	Security       string       `json:"security,omitempty"`
	Flow           string       `json:"flow,omitempty"`
	Method         string       `json:"method,omitempty"`
	Transport      *sbTransport `json:"transport,omitempty"`
	TLS            *sbTLS       `json:"tls,omitempty"`
	UpMbps         int          `json:"up_mbps,omitempty"`
	DownMbps       int          `json:"down_mbps,omitempty"`
	Obfs           *sbHysteria2Obfs `json:"obfs,omitempty"`
	CongestionControl string    `json:"congestion_control,omitempty"`
	UDPRelayMode   string       `json:"udp_relay_mode,omitempty"`
}

type sbTransport struct {
	Type    string         `json:"type"`
	Path    string         `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	ServiceName string     `json:"service_name,omitempty"`
	Host    string         `json:"host,omitempty"`
}

type sbTLS struct {
	Enabled    bool         `json:"enabled"`
	ServerName string       `json:"server_name,omitempty"`
	Insecure   bool         `json:"insecure,omitempty"`
	ALPN       []string     `json:"alpn,omitempty"`
	Reality    *sbReality   `json:"reality,omitempty"`
	UTLS       *sbUTLS      `json:"utls,omitempty"`
}

type sbReality struct {
	Enabled    bool   `json:"enabled"`
	PublicKey  string `json:"public_key,omitempty"`
	ShortID    string `json:"short_id,omitempty"`
}

type sbUTLS struct {
	Enabled     bool   `json:"enabled"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

type sbHysteria2Obfs struct {
	Type     string `json:"type"`
	Password string `json:"password,omitempty"`
}

type sbRoute struct {
	Rules []sbRule `json:"rules"`
	Final string   `json:"final"`
}

type sbRule struct {
	Inbound  string `json:"inbound,omitempty"`
	Outbound string `json:"outbound"`
}

func (c *SingBoxCore) GenerateConfig(nodes []model.ProxyNode) ([]byte, error) {
	cfg := sbConfig{
		Log: sbLog{
			Level:  "warn",
			Output: c.LogPath(),
		},
		Route: sbRoute{Final: "direct"},
	}

	for i, node := range nodes {
		if node.SOCKS5Port == 0 {
			return nil, fmt.Errorf("node %s has no SOCKS5 port assigned", node.Name)
		}

		tag := sanitizeTag(node.Name)
		inTag := "socks-" + tag

		// Inbound: SOCKS5
		cfg.Inbounds = append(cfg.Inbounds, sbInbound{
			Type:       "socks",
			Tag:        inTag,
			Listen:     "127.0.0.1",
			ListenPort: node.SOCKS5Port,
			Sniff:      true,
		})

		// Outbound
		out, err := c.buildOutbound(node, tag)
		if err != nil {
			return nil, fmt.Errorf("node %s (index %d): %w", node.Name, i, err)
		}
		cfg.Outbounds = append(cfg.Outbounds, out)

		// Route rule
		cfg.Route.Rules = append(cfg.Route.Rules, sbRule{
			Inbound:  inTag,
			Outbound: tag,
		})
	}

	// Direct outbound for fallback
	cfg.Outbounds = append(cfg.Outbounds, sbOutbound{
		Type: "direct",
		Tag:  "direct",
	})

	return json.MarshalIndent(cfg, "", "  ")
}

func (c *SingBoxCore) buildOutbound(node model.ProxyNode, tag string) (sbOutbound, error) {
	out := sbOutbound{Tag: tag}

	switch node.Protocol {
	case model.ProtoVMess:
		out.Type = "vmess"
		out.Server = node.Address
		out.ServerPort = node.Port
		out.UUID = node.UUID
		out.Security = node.Security
	case model.ProtoVLESS:
		out.Type = "vless"
		out.Server = node.Address
		out.ServerPort = node.Port
		out.UUID = node.UUID
		out.Flow = node.Flow
	case model.ProtoTrojan:
		out.Type = "trojan"
		out.Server = node.Address
		out.ServerPort = node.Port
		out.Password = node.Password
	case model.ProtoShadowsocks:
		out.Type = "shadowsocks"
		out.Server = node.Address
		out.ServerPort = node.Port
		out.Password = node.Password
		out.Method = node.SSMethod
	case model.ProtoHTTP, model.ProtoSOCKS:
		out.Type = string(node.Protocol) // "http" or "socks"
		out.Server = node.Address
		out.ServerPort = node.Port
		if node.Username != "" {
			out.Username = node.Username
			out.Password = node.Password
		}
	case model.ProtoHysteria2:
		out.Type = "hysteria2"
		out.Server = node.Address
		out.ServerPort = node.Port
		if node.Hysteria2AuthPass != "" {
			out.Password = node.Hysteria2AuthPass
		} else {
			out.Password = node.Password
		}
		if node.Hysteria2UpMbps > 0 {
			out.UpMbps = node.Hysteria2UpMbps
		}
		if node.Hysteria2DownMbps > 0 {
			out.DownMbps = node.Hysteria2DownMbps
		}
		if node.Hysteria2ObfsType != "" && node.Hysteria2ObfsType != "disable" {
			out.Obfs = &sbHysteria2Obfs{
				Type:     node.Hysteria2ObfsType,
				Password: node.Hysteria2ObfsPass,
			}
		}
	case model.ProtoTUIC:
		out.Type = "tuic"
		out.Server = node.Address
		out.ServerPort = node.Port
		out.UUID = node.UUID
		out.Password = node.Password
		if node.TUICCongestionControl != "" {
			out.CongestionControl = node.TUICCongestionControl
		}
		if node.TUICUDPMode != "" {
			out.UDPRelayMode = node.TUICUDPMode
		}
	case model.ProtoNaive:
		out.Type = "naive"
		out.Server = node.Address
		out.ServerPort = node.Port
		out.Username = node.Username
		out.Password = node.Password
	case model.ProtoWireGuard:
		out.Type = "wireguard"
		out.Server = node.Address
		out.ServerPort = node.Port
	default:
		return out, fmt.Errorf("unsupported protocol %s for sing-box", node.Protocol)
	}

	// Transport
	if node.Transport != "" && node.Transport != "tcp" {
		t := &sbTransport{Type: node.Transport}
		switch node.Transport {
		case "ws":
			if node.WSHost != "" {
				t.Headers = map[string]string{"Host": node.WSHost}
			}
			t.Path = node.WSPath
		case "grpc":
			t.ServiceName = node.GRPCServiceName
		case "httpupgrade":
			t.Host = node.WSHost
			t.Path = node.WSPath
		}
		out.Transport = t
	}

	// TLS
	if node.TLS || node.Reality {
		tls := &sbTLS{Enabled: true}
		if node.SNI != "" {
			tls.ServerName = node.SNI
		}
		if len(node.ALPN) > 0 {
			tls.ALPN = node.ALPN
		}
		if node.Reality {
			tls.Reality = &sbReality{Enabled: true}
		}
		if node.Fingerprint != "" {
			tls.UTLS = &sbUTLS{
				Enabled:     true,
				Fingerprint: node.Fingerprint,
			}
		}
		out.TLS = tls
	}

	return out, nil
}

var tagCleanRE = regexp.MustCompile(`[^a-zA-Z0-9_-]`)

func sanitizeTag(name string) string {
	return tagCleanRE.ReplaceAllString(strings.ReplaceAll(name, " ", "_"), "")
}

// Launch starts sing-box on the remote.
func (c *SingBoxCore) Launch(ctx context.Context, client *sshutil.Client, configPath string) (int, error) {
	cmd := fmt.Sprintf(
		"nohup sing-box run -c %s > %s 2>&1 & echo $!",
		configPath, c.LogPath(),
	)
	stdout, _, err := client.ExecuteWithContext(ctx, cmd)
	if err != nil {
		return 0, fmt.Errorf("launch sing-box: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(stdout))
	if err != nil {
		return 0, fmt.Errorf("parse PID from %q: %w", stdout, err)
	}

	// Give sing-box a moment to start
	time.Sleep(1 * time.Second)

	return pid, nil
}

// Kill terminates the sing-box process and cleans up temp files.
func (c *SingBoxCore) Kill(ctx context.Context, client *sshutil.Client, pid int) error {
	// Kill the process
	client.ExecuteWithContext(ctx, fmt.Sprintf("kill %d 2>/dev/null", pid))
	// Clean up temp files
	client.ExecuteWithContext(ctx, fmt.Sprintf("rm -f %s %s", c.ConfigPath(), c.LogPath()))
	return nil
}
