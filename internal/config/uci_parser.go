package config

import (
	"fmt"
	"os"
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
		if _, hasType := raw.props["type"]; !hasType {
			// Skip sections that don't have a 'type' property (not a proxy node)
			continue
		}
		node, err := normalizeNode(raw.props)
		if err != nil {
			fmt.Fprintf(os.Stderr, "skipping node %s: %v\n", section, err)
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
	case "Shadowsocks", "SS-Rust", "ss":
		n.Type = model.NodeTypeShadowsocks
		return normalizeShadowsocksNode(n, props)
	case "Hysteria2", "hysteria2":
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

	// Helper to get prefixed or unprefixed property
	getProp := func(key string) string {
		if v, ok := props["xray_"+key]; ok && v != "" {
			return v
		}
		if v, ok := props[key]; ok && v != "" {
			return v
		}
		return ""
	}

	n.Address = getProp("address")
	port, _ := strconv.Atoi(getProp("port"))
	n.Port = port
	n.UUID = getProp("uuid")
	n.Password = getProp("password")
	n.Security = getProp("security")
	n.Flow = getProp("flow")
	
	tlsVal := getProp("tls")
	n.TLS = tlsVal == "1" || tlsVal == "true"
	
	realityVal := getProp("reality")
	n.Reality = realityVal == "1" || realityVal == "true"
	n.RealityPublicKey = getProp("reality_publicKey")
	n.RealityShortID = getProp("reality_shortId")
	
	n.Transport = getProp("transport")
	n.WSHost = getProp("ws_host")
	n.WSPath = getProp("ws_path")
	n.SNI = getProp("tls_serverName")
	if n.SNI == "" {
		n.SNI = getProp("sni") // another common field name
	}
	n.Fingerprint = getProp("fingerprint")
	n.GRPCServiceName = getProp("grpc_serviceName")
	n.SSMethod = getProp("ss_method")
	if n.SSMethod == "" {
		n.SSMethod = getProp("method")
	}

	// ALPN
	if alpn := getProp("alpn"); alpn != "" {
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
	if n.Protocol == model.ProtoAnyTLS || n.Protocol == model.ProtoTUIC || n.Protocol == model.ProtoHysteria2 {
		n.TLS = true
	}
	n.Reality = props["reality"] == "1" || props["reality"] == "true"
	n.RealityPublicKey = props["reality_publicKey"]
	n.RealityShortID = props["reality_shortId"]
	n.Transport = props["transport"]
	n.WSHost = props["ws_host"]
	n.WSPath = props["ws_path"]
	if n.Transport == "xhttp" {
		if path := props["xhttp_path"]; path != "" {
			n.WSPath = path
		}
		if host := props["xhttp_host"]; host != "" {
			n.WSHost = host
		}
	}
	n.SNI = props["tls_serverName"]
	if props["tls_disable_sni"] == "1" {
		n.SNI = ""
	}
	n.Fingerprint = props["fingerprint"]
	n.Insecure = props["tls_allowInsecure"] == "1" || props["tls_allowInsecure"] == "true"
	n.SSMethod = props["ss_method"]

	if alpn := props["alpn"]; alpn != "" {
		n.ALPN = strings.Split(alpn, ",")
	} else if alpn := props["tuic_alpn"]; alpn != "" {
		n.ALPN = strings.Split(alpn, ",")
	}

	// Hysteria2 fields
	n.Hysteria2UpMbps, _ = strconv.Atoi(props["hysteria2_up_mbps"])
	n.Hysteria2DownMbps, _ = strconv.Atoi(props["hysteria2_down_mbps"])
	n.Hysteria2ObfsType = props["hysteria2_obfs_type"]
	n.Hysteria2ObfsPass = props["hysteria2_obfs_password"]
	n.Hysteria2AuthPass = props["hysteria2_auth_password"]

	// TUIC fields
	n.TUICCongestionControl = props["tuic_congestion_control"]
	n.TUICUDPMode = props["tuic_udp_relay_mode"]

	// AnyTLS fields
	n.AnyTLSIdleCheckInterval = props["anytls_idle_session_check_interval"]
	n.AnyTLSIdleTimeout = props["anytls_idle_session_timeout"]
	if v, err := strconv.Atoi(props["anytls_min_idle_session"]); err == nil {
		n.AnyTLSMinIdleSession = v
	}

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
	if n.Protocol == "" {
		n.Protocol = model.ProtoShadowsocks
	}
	n.Address = props["address"]
	if n.Address == "" {
		n.Address = props["server"]
	}
	portStr := props["port"]
	if portStr == "" {
		portStr = props["server_port"]
	}
	port, _ := strconv.Atoi(portStr)
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
	n.Address = props["address"]
	if n.Address == "" {
		n.Address = props["server"]
	}
	portStr := props["port"]
	if portStr == "" {
		portStr = props["server_port"]
	}
	port, _ := strconv.Atoi(portStr)
	n.Port = port

	n.Password = props["hysteria2_auth_password"]
	if n.Password == "" {
		n.Password = props["auth_password"]
	}
	if n.Password == "" {
		n.Password = props["password"]
	}

	upStr := props["hysteria2_up_mbps"]
	if upStr == "" {
		upStr = props["up_mbps"]
	}
	n.Hysteria2UpMbps, _ = strconv.Atoi(upStr)

	downStr := props["hysteria2_down_mbps"]
	if downStr == "" {
		downStr = props["down_mbps"]
	}
	n.Hysteria2DownMbps, _ = strconv.Atoi(downStr)

	n.Hysteria2ObfsType = props["hysteria2_obfs_type"]
	if n.Hysteria2ObfsType == "" {
		n.Hysteria2ObfsType = props["obfs_type"]
	}

	n.Hysteria2ObfsPass = props["hysteria2_obfs_password"]
	if n.Hysteria2ObfsPass == "" {
		n.Hysteria2ObfsPass = props["obfs_password"]
	}

	n.Hysteria2AuthPass = n.Password // fallback
	n.TLS = true

	if n.Name == "" {
		n.Name = fmt.Sprintf("Hysteria2-%s", n.Address)
	}
	if n.Address == "" {
		return n, fmt.Errorf("Hysteria2 node %s missing address", n.Name)
	}
	return n, nil
}

// normalizeTUICNode handles standalone TUIC-type nodes.
func normalizeTUICNode(n model.ProxyNode, props map[string]string) (model.ProxyNode, error) {
	n.Name = props["remarks"]
	n.Protocol = model.ProtoTUIC
	n.Address = props["address"]
	if n.Address == "" {
		n.Address = props["server"]
	}
	portStr := props["port"]
	if portStr == "" {
		portStr = props["server_port"]
	}
	port, _ := strconv.Atoi(portStr)
	n.Port = port
	n.UUID = props["uuid"]
	n.Password = props["password"]
	n.TUICCongestionControl = props["congestion_control"]
	n.TUICUDPMode = props["udp_relay_mode"]
	n.TLS = true

	if n.Name == "" {
		n.Name = fmt.Sprintf("TUIC-%s", n.Address)
	}
	if n.Address == "" {
		return n, fmt.Errorf("TUIC node %s missing address", n.Name)
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
