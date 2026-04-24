package shadow

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/txdywy/inice/internal/model"
	sshutil "github.com/txdywy/inice/internal/ssh"
)

// XrayCore implements ShadowCore for xray-core (fallback).
type XrayCore struct {
	ts string
}

// NewXrayCore creates a new XrayCore.
func NewXrayCore() *XrayCore {
	return &XrayCore{ts: strconv.FormatInt(time.Now().Unix(), 10)}
}

func (c *XrayCore) Type() CoreType { return CoreXray }

func (c *XrayCore) ConfigPath() string {
	return fmt.Sprintf("/tmp/inice-xray-%s.json", c.ts)
}

func (c *XrayCore) LogPath() string {
	return fmt.Sprintf("/tmp/inice-xray-%s.log", c.ts)
}

// Xray JSON structures
type xrConfig struct {
	Log       xrLog        `json:"log"`
	Inbounds  []xrInbound  `json:"inbounds"`
	Outbounds []xrOutbound `json:"outbounds"`
	Routing   xrRouting    `json:"routing"`
}

type xrLog struct {
	LogLevel string `json:"loglevel"`
}

type xrInbound struct {
	Port     int           `json:"port"`
	Protocol string        `json:"protocol"`
	Listen   string        `json:"listen"`
	Settings xrInSettings  `json:"settings"`
	Tag      string        `json:"tag"`
	Sniffing xrSniffing    `json:"sniffing"`
}

type xrInSettings struct {
	Auth string `json:"auth,omitempty"`
}

type xrSniffing struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type xrOutbound struct {
	Protocol       string           `json:"protocol"`
	Tag            string           `json:"tag"`
	Settings       json.RawMessage  `json:"settings,omitempty"`
	StreamSettings *xrStreamSettings `json:"streamSettings,omitempty"`
	SendThrough    string           `json:"sendThrough,omitempty"`
}

type xrStreamSettings struct {
	Network      string       `json:"network,omitempty"`
	Security     string       `json:"security,omitempty"`
	WSSettings   *xrWSSettings `json:"wsSettings,omitempty"`
	TLSSettings  *xrTLS        `json:"tlsSettings,omitempty"`
	GRPCSettings *xrGRPC      `json:"grpcSettings,omitempty"`
}

type xrWSSettings struct {
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type xrTLS struct {
	ServerName    string `json:"serverName,omitempty"`
	AllowInsecure bool   `json:"allowInsecure,omitempty"`
}

type xrGRPC struct {
	ServiceName string `json:"serviceName,omitempty"`
}

type xrRouting struct {
	Rules []xrRule `json:"rules"`
}

type xrRule struct {
	InboundTag  []string `json:"inboundTag,omitempty"`
	OutboundTag string   `json:"outboundTag,omitempty"`
	Type        string   `json:"type"`
}

func (c *XrayCore) GenerateConfig(nodes []model.ProxyNode) ([]byte, error) {
	cfg := xrConfig{
		Log: xrLog{LogLevel: "warning"},
	}

	inboundTags := make([]string, 0)

	for i, node := range nodes {
		if node.SOCKS5Port == 0 {
			return nil, fmt.Errorf("node %s has no SOCKS5 port assigned", node.Name)
		}
		tag := sanitizeXrayTag(node.Name)
		inTag := "socks-" + tag

		cfg.Inbounds = append(cfg.Inbounds, xrInbound{
			Port:     node.SOCKS5Port,
			Protocol: "socks",
			Listen:   "127.0.0.1",
			Tag:      inTag,
			Settings: xrInSettings{},
			Sniffing: xrSniffing{
				Enabled:      true,
				DestOverride: []string{"http", "tls"},
			},
		})
		inboundTags = append(inboundTags, inTag)

		out, err := c.buildOutbound(node, tag)
		if err != nil {
			return nil, fmt.Errorf("node %s (index %d): %w", node.Name, i, err)
		}
		cfg.Outbounds = append(cfg.Outbounds, out)

		cfg.Routing.Rules = append(cfg.Routing.Rules, xrRule{
			InboundTag:  []string{inTag},
			OutboundTag: tag,
			Type:        "field",
		})
	}

	// Direct outbound
	cfg.Outbounds = append(cfg.Outbounds, xrOutbound{
		Protocol: "freedom",
		Tag:      "direct",
	})

	return json.MarshalIndent(cfg, "", "  ")
}

func (c *XrayCore) buildOutbound(node model.ProxyNode, tag string) (xrOutbound, error) {
	out := xrOutbound{Tag: tag}
	var settings json.RawMessage
	var err error

	switch node.Protocol {
	case model.ProtoVMess:
		out.Protocol = "vmess"
		settings, err = xrVMessSettings(node)
	case model.ProtoVLESS:
		out.Protocol = "vless"
		settings, err = xrVLESSettings(node)
	case model.ProtoTrojan:
		out.Protocol = "trojan"
		settings, err = xrTrojanSettings(node)
	case model.ProtoShadowsocks:
		out.Protocol = "shadowsocks"
		settings, err = xrShadowsocksSettings(node)
	case model.ProtoHTTP:
		out.Protocol = "http"
		settings, err = xrHTTPSettings(node)
	case model.ProtoSOCKS:
		out.Protocol = "socks"
		settings, err = xrSOCKSSettings(node)
	default:
		return out, fmt.Errorf("unsupported protocol %s for xray-core", node.Protocol)
	}

	if err != nil {
		return out, err
	}
	out.Settings = settings

	// Stream settings (transport + TLS)
	if node.Transport != "" && node.Transport != "tcp" {
		ss := &xrStreamSettings{Network: node.Transport}
		switch node.Transport {
		case "ws":
			ss.WSSettings = &xrWSSettings{Path: node.WSPath}
			if node.WSHost != "" {
				ss.WSSettings.Headers = map[string]string{"Host": node.WSHost}
			}
		case "grpc":
			ss.GRPCSettings = &xrGRPC{ServiceName: node.GRPCServiceName}
		}
		out.StreamSettings = ss
	}

	if node.TLS {
		if out.StreamSettings == nil {
			out.StreamSettings = &xrStreamSettings{}
		}
		out.StreamSettings.Security = "tls"
		out.StreamSettings.TLSSettings = &xrTLS{
			ServerName: node.SNI,
		}
	}

	return out, nil
}

func xrVMessSettings(n model.ProxyNode) (json.RawMessage, error) {
	s := map[string]interface{}{
		"vnext": []map[string]interface{}{
			{
				"address": n.Address,
				"port":    n.Port,
				"users": []map[string]interface{}{
					{
						"id":       n.UUID,
						"security": n.Security,
					},
				},
			},
		},
	}
	return json.Marshal(s)
}

func xrVLESSettings(n model.ProxyNode) (json.RawMessage, error) {
	s := map[string]interface{}{
		"vnext": []map[string]interface{}{
			{
				"address": n.Address,
				"port":    n.Port,
				"users": []map[string]interface{}{
					{
						"id":         n.UUID,
						"encryption": "none",
						"flow":       n.Flow,
					},
				},
			},
		},
	}
	return json.Marshal(s)
}

func xrTrojanSettings(n model.ProxyNode) (json.RawMessage, error) {
	s := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"address":  n.Address,
				"port":     n.Port,
				"password": n.Password,
			},
		},
	}
	return json.Marshal(s)
}

func xrShadowsocksSettings(n model.ProxyNode) (json.RawMessage, error) {
	s := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"address":  n.Address,
				"port":     n.Port,
				"password": n.Password,
				"method":   n.SSMethod,
			},
		},
	}
	return json.Marshal(s)
}

func xrHTTPSettings(n model.ProxyNode) (json.RawMessage, error) {
	s := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"address": n.Address,
				"port":    n.Port,
				"users": []map[string]string{
					{"user": n.Username, "pass": n.Password},
				},
			},
		},
	}
	return json.Marshal(s)
}

func xrSOCKSSettings(n model.ProxyNode) (json.RawMessage, error) {
	users := []map[string]interface{}{}
	if n.Username != "" {
		users = append(users, map[string]interface{}{
			"user": n.Username,
			"pass": n.Password,
			"level": 0,
		})
	}
	s := map[string]interface{}{
		"servers": []map[string]interface{}{
			{
				"address": n.Address,
				"port":    n.Port,
				"users":   users,
			},
		},
	}
	return json.Marshal(s)
}

func sanitizeXrayTag(name string) string {
	return sanitizeTag(name) // reuse sing-box's helper
}

// Launch starts xray on the remote.
func (c *XrayCore) Launch(ctx context.Context, client *sshutil.Client, configPath string) (int, error) {
	cmd := fmt.Sprintf(
		"nohup xray run -c %s > %s 2>&1 & echo $!",
		configPath, c.LogPath(),
	)
	stdout, _, err := client.ExecuteWithContext(ctx, cmd)
	if err != nil {
		return 0, fmt.Errorf("launch xray: %w", err)
	}

	pid, err := strconv.Atoi(strings.TrimSpace(stdout))
	if err != nil {
		return 0, fmt.Errorf("parse PID from %q: %w", stdout, err)
	}
	time.Sleep(1 * time.Second)
	return pid, nil
}

// Kill terminates the xray process and cleans up temp files.
func (c *XrayCore) Kill(ctx context.Context, client *sshutil.Client, pid int) error {
	client.ExecuteWithContext(ctx, fmt.Sprintf("kill %d 2>/dev/null", pid))
	client.ExecuteWithContext(ctx, fmt.Sprintf("rm -f %s %s", c.ConfigPath(), c.LogPath()))
	return nil
}
