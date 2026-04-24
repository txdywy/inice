package config

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/txdywy/inice/internal/model"
)

var (
	// Matches anonymous and named node sections:
	// passwall2.@nodes[0]=nodes
	// passwall2.<section_id>=nodes
	reSection = regexp.MustCompile(`^passwall2\.(@nodes\[\d+\]|[-A-Za-z0-9_]+)=(?:nodes?|['"]nodes?['"])$`)
	// Matches values under anonymous and named sections:
	// passwall2.@nodes[0].type=Xray
	// passwall2.<section_id>.type=Xray
	reKeyVal = regexp.MustCompile(`^passwall2\.(@nodes\[\d+\]|[-A-Za-z0-9_]+)\.(.+?)=(.*)$`)
)

// rawNode holds the raw key-value pairs for a single UCI node section.
type rawNode struct {
	props map[string]string
}

// ParseUCIOutput parses the output of `uci show passwall2` and returns
// normalized ProxyNode structs for all node-type sections found.
func ParseUCIOutput(output string) ([]model.ProxyNode, error) {
	lines := strings.Split(output, "\n")
	rawMap := make(map[string]*rawNode)
	var sectionOrder []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this is a section declaration
		if m := reSection.FindStringSubmatch(line); m != nil {
			section := m[1]
			if _, exists := rawMap[section]; !exists {
				rawMap[section] = &rawNode{props: make(map[string]string)}
				sectionOrder = append(sectionOrder, section)
			}
			continue
		}

		// Check if this is a key=value pair
		if m := reKeyVal.FindStringSubmatch(line); m != nil {
			section := m[1]
			key := m[2]
			value := cleanUCIValue(m[3])

			raw, exists := rawMap[section]
			if !exists {
				continue
			}
			raw.props[key] = value
		}
	}

	var nodes []model.ProxyNode
	for _, section := range sectionOrder {
		raw := rawMap[section]
		node, err := normalizeNode(raw.props)
		if err != nil {
			// Log warning but continue with other nodes
			continue
		}
		nodes = append(nodes, node)
	}

	if len(nodes) == 0 {
		return nil, fmt.Errorf("no valid PassWall2 nodes found in UCI output")
	}

	return nodes, nil
}

func cleanUCIValue(value string) string {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return value
	}
	if (value[0] == '\'' && value[len(value)-1] == '\'') ||
		(value[0] == '"' && value[len(value)-1] == '"') {
		return value[1 : len(value)-1]
	}
	return value
}

// normalizeNode converts raw UCI properties into a ProxyNode.
func normalizeNode(props map[string]string) (model.ProxyNode, error) {
	n := model.ProxyNode{}

	// Determine node type
	nodeType := props["type"]
	switch nodeType {
	case "Xray":
		n.Type = model.NodeTypeXray
		return normalizeXrayNode(n, props)
	case "sing-box":
		n.Type = model.NodeTypeSingBox
		return normalizeSingBoxNode(n, props)
	case "Shadowsocks":
		n.Type = model.NodeTypeShadowsocks
		return normalizeShadowsocksNode(n, props)
	case "hysteria2":
		n.Type = model.NodeTypeHysteria2
		return normalizeHysteria2Node(n, props)
	case "TUIC":
		n.Type = model.NodeTypeTUIC
		return normalizeTUICNode(n, props)
	case "naiveproxy":
		n.Type = model.NodeTypeNaiveProxy
		return normalizeNaiveNode(n, props)
	default:
		return n, fmt.Errorf("unknown node type: %s", nodeType)
	}
}

// normalizeXrayNode handles Xray-type nodes (fields prefixed with xray_).
func normalizeXrayNode(n model.ProxyNode, props map[string]string) (model.ProxyNode, error) {
	n.Name = props["remarks"]
	n.Protocol = model.Protocol(props["protocol"])

	// Xray-type nodes use xray_ prefix
	n.Address = props["xray_address"]
	port, _ := strconv.Atoi(props["xray_port"])
	n.Port = port
	n.UUID = props["xray_uuid"]
	n.Password = props["xray_password"]
	n.Security = props["xray_security"]
	n.Flow = props["xray_flow"]
	n.TLS = props["xray_tls"] == "1"
	n.Reality = props["xray_reality"] == "1" || props["xray_reality"] == "true"
	n.Transport = props["xray_transport"]
	n.WSHost = props["xray_ws_host"]
	n.WSPath = props["xray_ws_path"]
	n.SNI = props["xray_tls_serverName"]
	n.Fingerprint = props["xray_fingerprint"]
	n.GRPCServiceName = props["xray_grpc_serviceName"]
	n.SSMethod = props["xray_ss_method"]

	// ALPN
	if alpn := props["xray_alpn"]; alpn != "" {
		n.ALPN = strings.Split(alpn, ",")
	}

	if n.Name == "" {
		n.Name = fmt.Sprintf("%s-%s-%s", n.Type, n.Protocol, n.Address)
	}
	if n.Address == "" {
		return n, fmt.Errorf("Xray node %s missing address", n.Name)
	}
	return n, nil
}

// normalizeSingBoxNode handles sing-box type nodes (fields without prefix).
func normalizeSingBoxNode(n model.ProxyNode, props map[string]string) (model.ProxyNode, error) {
	n.Name = props["remarks"]
	n.Protocol = model.Protocol(props["protocol"])
	n.Address = props["address"]
	port, _ := strconv.Atoi(props["port"])
	n.Port = port
	n.UUID = props["uuid"]
	n.Password = props["password"]
	n.Username = props["username"]
	n.TLS = props["tls"] == "1" || props["tls"] == "true"
	n.Reality = props["reality"] == "1" || props["reality"] == "true"
	n.Transport = props["transport"]
	n.WSHost = props["ws_host"]
	n.WSPath = props["ws_path"]
	n.SNI = props["tls_serverName"]
	n.Fingerprint = props["fingerprint"]
	n.SSMethod = props["ss_method"]

	// Hysteria2 fields
	n.Hysteria2UpMbps, _ = strconv.Atoi(props["hysteria2_up_mbps"])
	n.Hysteria2DownMbps, _ = strconv.Atoi(props["hysteria2_down_mbps"])
	n.Hysteria2ObfsType = props["hysteria2_obfs_type"]
	n.Hysteria2ObfsPass = props["hysteria2_obfs_password"]
	n.Hysteria2AuthPass = props["hysteria2_auth_password"]

	// TUIC fields
	n.TUICCongestionControl = props["tuic_congestion_control"]
	n.TUICUDPMode = props["tuic_udp_relay_mode"]

	if n.Name == "" {
		n.Name = fmt.Sprintf("%s-%s-%s", n.Type, n.Protocol, n.Address)
	}
	if n.Address == "" {
		return n, fmt.Errorf("sing-box node %s missing address", n.Name)
	}
	return n, nil
}

// normalizeShadowsocksNode handles standalone Shadowsocks-type nodes.
func normalizeShadowsocksNode(n model.ProxyNode, props map[string]string) (model.ProxyNode, error) {
	n.Name = props["remarks"]
	n.Protocol = model.Protocol(props["protocol"])
	n.Address = props["server"]
	port, _ := strconv.Atoi(props["server_port"])
	n.Port = port
	n.Password = props["password"]
	n.SSMethod = props["method"]
	n.Transport = "tcp"

	if n.Name == "" {
		n.Name = fmt.Sprintf("SS-%s", n.Address)
	}
	if n.Address == "" {
		return n, fmt.Errorf("Shadowsocks node %s missing server", n.Name)
	}
	return n, nil
}

// normalizeHysteria2Node handles standalone Hysteria2-type nodes.
func normalizeHysteria2Node(n model.ProxyNode, props map[string]string) (model.ProxyNode, error) {
	n.Name = props["remarks"]
	n.Protocol = model.ProtoHysteria2
	n.Address = props["server"]
	port, _ := strconv.Atoi(props["server_port"])
	n.Port = port
	n.Password = props["password"]
	n.Hysteria2UpMbps, _ = strconv.Atoi(props["up_mbps"])
	n.Hysteria2DownMbps, _ = strconv.Atoi(props["down_mbps"])
	n.Hysteria2ObfsType = props["obfs_type"]
	n.Hysteria2ObfsPass = props["obfs_password"]
	n.Hysteria2AuthPass = props["auth_password"]

	if n.Name == "" {
		n.Name = fmt.Sprintf("Hysteria2-%s", n.Address)
	}
	return n, nil
}

// normalizeTUICNode handles standalone TUIC-type nodes.
func normalizeTUICNode(n model.ProxyNode, props map[string]string) (model.ProxyNode, error) {
	n.Name = props["remarks"]
	n.Protocol = model.ProtoTUIC
	n.Address = props["server"]
	port, _ := strconv.Atoi(props["server_port"])
	n.Port = port
	n.UUID = props["uuid"]
	n.Password = props["password"]
	n.TUICCongestionControl = props["congestion_control"]
	n.TUICUDPMode = props["udp_relay_mode"]

	if n.Name == "" {
		n.Name = fmt.Sprintf("TUIC-%s", n.Address)
	}
	return n, nil
}

// normalizeNaiveNode handles NaiveProxy-type nodes.
func normalizeNaiveNode(n model.ProxyNode, props map[string]string) (model.ProxyNode, error) {
	n.Name = props["remarks"]
	n.Protocol = model.ProtoNaive
	n.Address = props["server"]
	port, _ := strconv.Atoi(props["server_port"])
	n.Port = port
	n.Username = props["username"]
	n.Password = props["password"]

	if n.Name == "" {
		n.Name = fmt.Sprintf("Naive-%s", n.Address)
	}
	return n, nil
}
