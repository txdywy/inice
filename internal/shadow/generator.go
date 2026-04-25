package shadow

import (
	"encoding/json"
	"fmt"

	"github.com/txdywy/inice/internal/model"
)

// GenerateSingboxConfig creates a sing-box JSON configuration from proxy nodes.
// It uses the provided portMap (node index -> port) to configure inbounds.
func GenerateSingboxConfig(nodes []model.ProxyNode, portMap map[int]int) ([]byte, error) {
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes to generate config for")
	}

	inbounds := make([]map[string]interface{}, 0, len(nodes))
	outbounds := make([]map[string]interface{}, 0, len(nodes))
	rules := make([]map[string]interface{}, 0, len(nodes))

	for i, node := range nodes {
		port := portMap[i]

		inTag := fmt.Sprintf("socks-in-%d", i)
		outTag := fmt.Sprintf("out-%d", i)

		inbounds = append(inbounds, map[string]interface{}{
			"type":        "socks",
			"tag":         inTag,
			"listen":      "0.0.0.0",
			"listen_port": port,
		})

		outbound, err := buildOutbound(node, outTag)
		if err != nil {
			return nil, fmt.Errorf("node %q (%s): %w", node.Name, node.Protocol, err)
		}
		outbounds = append(outbounds, outbound)

		rules = append(rules, map[string]interface{}{
			"inbound":  []string{inTag},
			"outbound": outTag,
		})
	}

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"level":  "warn",
			"output": "/dev/null",
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"route": map[string]interface{}{
			"rules": rules,
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}
	return data, nil
}

func buildOutbound(node model.ProxyNode, tag string) (map[string]interface{}, error) {
	out := map[string]interface{}{
		"tag":  tag,
	}

	switch node.Protocol {
	case model.ProtoVMess:
		out["type"] = "vmess"
		out["server"] = node.Address
		out["server_port"] = node.Port
		out["uuid"] = node.UUID
		if node.Security != "" {
			out["security"] = node.Security
		} else {
			out["security"] = "auto"
		}
		addTLS(node, out)
		addTransport(node, out)

	case model.ProtoVLESS:
		out["type"] = "vless"
		out["server"] = node.Address
		out["server_port"] = node.Port
		out["uuid"] = node.UUID
		if node.Flow != "" {
			out["flow"] = node.Flow
		}
		addTLS(node, out)
		addTransport(node, out)

	case model.ProtoTrojan:
		out["type"] = "trojan"
		out["server"] = node.Address
		out["server_port"] = node.Port
		out["password"] = node.Password
		addTLS(node, out)
		addTransport(node, out)

	case model.ProtoShadowsocks:
		out["type"] = "shadowsocks"
		out["server"] = node.Address
		out["server_port"] = node.Port
		out["method"] = node.SSMethod
		out["password"] = node.Password
		addTLS(node, out)

	case model.ProtoHysteria2:
		out["type"] = "hysteria2"
		out["server"] = node.Address
		out["server_port"] = node.Port
		if node.Password != "" {
			out["password"] = node.Password
		}
		if node.Hysteria2UpMbps > 0 {
			out["up_mbps"] = node.Hysteria2UpMbps
		}
		if node.Hysteria2DownMbps > 0 {
			out["down_mbps"] = node.Hysteria2DownMbps
		}
		if node.Hysteria2ObfsType != "" {
			out["obfs"] = map[string]interface{}{
				"type": node.Hysteria2ObfsType,
			}
			if node.Hysteria2ObfsPass != "" {
				out["obfs"].(map[string]interface{})["password"] = node.Hysteria2ObfsPass
			}
		}
		addTLS(node, out)

	case model.ProtoTUIC:
		out["type"] = "tuic"
		out["server"] = node.Address
		out["server_port"] = node.Port
		out["uuid"] = node.UUID
		if node.Password != "" {
			out["password"] = node.Password
		}
		if node.TUICCongestionControl != "" {
			out["congestion_control"] = node.TUICCongestionControl
		}
		if node.TUICUDPMode != "" {
			out["udp_relay_mode"] = node.TUICUDPMode
		}
		addTLS(node, out)

	case model.ProtoNaive:
		out["type"] = "naive"
		out["server"] = node.Address
		out["server_port"] = node.Port
		if node.Username != "" {
			out["username"] = node.Username
		}
		if node.Password != "" {
			out["password"] = node.Password
		}
		addTLS(node, out)

	case model.ProtoHTTP:
		out["type"] = "http"
		out["server"] = node.Address
		out["server_port"] = node.Port
		if node.Username != "" {
			out["username"] = node.Username
		}
		if node.Password != "" {
			out["password"] = node.Password
		}
		addTLS(node, out)

	case model.ProtoSOCKS:
		out["type"] = "socks"
		out["server"] = node.Address
		out["server_port"] = node.Port
		if node.Username != "" {
			out["username"] = node.Username
		}
		if node.Password != "" {
			out["password"] = node.Password
		}

	case model.ProtoAnyTLS:
		out["type"] = "anytls"
		out["server"] = node.Address
		out["server_port"] = node.Port
		if node.Password != "" {
			out["password"] = node.Password
		}
		if node.AnyTLSIdleCheckInterval != "" {
			out["idle_session_check_interval"] = node.AnyTLSIdleCheckInterval
		}
		if node.AnyTLSIdleTimeout != "" {
			out["idle_session_timeout"] = node.AnyTLSIdleTimeout
		}
		if node.AnyTLSMinIdleSession > 0 {
			out["min_idle_session"] = node.AnyTLSMinIdleSession
		}
		addTLS(node, out)

	case model.ProtoWireGuard:
		return nil, fmt.Errorf("wireguard not yet supported for shadow testing")

	case model.ProtoSSH:
		return nil, fmt.Errorf("ssh protocol not yet supported for shadow testing")

	default:
		return nil, fmt.Errorf("unsupported protocol: %s", node.Protocol)
	}

	return out, nil
}

func addTLS(node model.ProxyNode, out map[string]interface{}) {
	if !node.TLS && !node.Reality {
		return
	}

	tls := map[string]interface{}{
		"enabled": true,
	}

	if node.Insecure {
		tls["insecure"] = true
	}

	if node.SNI != "" {
		// sing-box uses "server_name" instead of "sni" in outbound tls
		tls["server_name"] = node.SNI
	} else if node.Address != "" {
		tls["server_name"] = node.Address
	}

	if len(node.ALPN) > 0 {
		tls["alpn"] = node.ALPN
	}

	if node.Fingerprint != "" {
		tls["utls"] = map[string]interface{}{
			"enabled":     true,
			"fingerprint": node.Fingerprint,
		}
	}

	if node.Reality {
		reality := map[string]interface{}{
			"enabled": true,
		}
		if node.RealityPublicKey != "" {
			reality["public_key"] = node.RealityPublicKey
		}
		if node.RealityShortID != "" {
			reality["short_id"] = node.RealityShortID
		}
		tls["reality"] = reality
	}

	out["tls"] = tls
}

func addTransport(node model.ProxyNode, out map[string]interface{}) {
	if node.Transport == "" || node.Transport == "tcp" || node.Transport == "raw" {
		return
	}

	if node.Transport == "xhttp" {
		node.Transport = "httpupgrade"
	}

	transport := map[string]interface{}{
		"type": node.Transport,
	}

	switch node.Transport {
	case "ws":
		path := node.WSPath
		if path == "" {
			path = "/"
		}
		transport["path"] = path
		if node.WSHost != "" {
			transport["headers"] = map[string]string{
				"Host": node.WSHost,
			}
		}
		if node.TLS {
			transport["early_data_header_name"] = "Sec-WebSocket-Protocol"
		}

	case "grpc":
		if node.GRPCServiceName != "" {
			transport["service_name"] = node.GRPCServiceName
		}
		transport["idle_timeout"] = "60s"
		transport["ping_timeout"] = "20s"

	case "httpupgrade":
		path := node.WSPath
		if path == "" {
			path = "/"
		}
		transport["path"] = path
		if node.WSHost != "" {
			transport["host"] = node.WSHost
		}

	case "tcp":
		// TCP transport has no additional options
		return
	}

	out["transport"] = transport
}
