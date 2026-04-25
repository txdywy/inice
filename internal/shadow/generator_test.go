package shadow

import (
	"encoding/json"
	"testing"

	"github.com/txdywy/inice/internal/model"
)

// Generate is a test helper that simulates the old Generate signature
func Generate(nodes []model.ProxyNode, basePort int) ([]byte, map[int]int, error) {
	portMap := make(map[int]int, len(nodes))
	for i := range nodes {
		portMap[i] = basePort + i
	}
	data, err := GenerateSingboxConfig(nodes, portMap)
	return data, portMap, err
}

func TestGenerate_EmptyNodes(t *testing.T) {
	_, _, err := Generate([]model.ProxyNode{}, 20000)
	if err == nil {
		t.Error("expected error for empty nodes")
	}
}

func TestGenerate_PortAssignment(t *testing.T) {
	nodes := []model.ProxyNode{
		{Name: "n1", Protocol: model.ProtoVMess, Address: "a1.com", Port: 443, UUID: "u1"},
		{Name: "n2", Protocol: model.ProtoVMess, Address: "a2.com", Port: 443, UUID: "u2"},
		{Name: "n3", Protocol: model.ProtoVMess, Address: "a3.com", Port: 443, UUID: "u3"},
	}
	data, portMap, err := Generate(nodes, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty config")
	}
	if len(portMap) != 3 {
		t.Fatalf("expected 3 port mappings, got %d", len(portMap))
	}
	if portMap[0] != 20000 {
		t.Errorf("port[0]: expected 20000, got %d", portMap[0])
	}
	if portMap[1] != 20001 {
		t.Errorf("port[1]: expected 20001, got %d", portMap[1])
	}
	if portMap[2] != 20002 {
		t.Errorf("port[2]: expected 20002, got %d", portMap[2])
	}
}

func TestGenerate_VMess(t *testing.T) {
	node := model.ProxyNode{
		Name:     "HK-VMess",
		Type:     model.NodeTypeXray,
		Protocol: model.ProtoVMess,
		Address:  "hk.example.com",
		Port:     443,
		UUID:     "12345678-1234-1234-1234-123456789abc",
		Security: "auto",
		TLS:      true,
		SNI:      "hk.example.com",
		Transport: "ws",
		WSPath:   "/path",
		WSHost:   "hk.example.com",
	}
	data, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	// Check inbounds
	inbounds := cfg["inbounds"].([]interface{})
	if len(inbounds) != 1 {
		t.Fatalf("expected 1 inbound, got %d", len(inbounds))
	}
	in := inbounds[0].(map[string]interface{})
	if in["type"] != "socks" {
		t.Errorf("inbound type: expected socks, got %v", in["type"])
	}
	if in["listen_port"].(float64) != 20000 {
		t.Errorf("inbound port: expected 20000, got %v", in["listen_port"])
	}

	// Check outbounds
	outbounds := cfg["outbounds"].([]interface{})
	if len(outbounds) != 1 {
		t.Fatalf("expected 1 outbound, got %d", len(outbounds))
	}
	out := outbounds[0].(map[string]interface{})
	if out["type"] != "vmess" {
		t.Errorf("outbound type: expected vmess, got %v", out["type"])
	}
	if out["server"] != "hk.example.com" {
		t.Errorf("outbound server mismatch")
	}
	if out["uuid"] != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("outbound uuid mismatch")
	}

	// Check TLS
	tls := out["tls"].(map[string]interface{})
	if tls["enabled"] != true {
		t.Error("tls should be enabled")
	}
	if tls["server_name"] != "hk.example.com" {
		t.Errorf("tls server_name mismatch")
	}

	// Check transport
	transport := out["transport"].(map[string]interface{})
	if transport["type"] != "ws" {
		t.Errorf("transport type: expected ws, got %v", transport["type"])
	}
	if transport["path"] != "/path" {
		t.Errorf("transport path mismatch")
	}
}

func TestGenerate_VLESS(t *testing.T) {
	node := model.ProxyNode{
		Name:     "JP-VLESS",
		Protocol: model.ProtoVLESS,
		Address:  "jp.example.com",
		Port:     443,
		UUID:     "uuid-123",
		Flow:     "xtls-rprx-vision",
		TLS:      true,
		Reality:  true,
		SNI:      "jp.example.com",
		Transport: "grpc",
		GRPCServiceName: "svc",
	}
	data, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	out := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	if out["type"] != "vless" {
		t.Errorf("expected vless, got %v", out["type"])
	}
	if out["flow"] != "xtls-rprx-vision" {
		t.Errorf("flow mismatch")
	}

	tls := out["tls"].(map[string]interface{})
	if tls["enabled"] != true {
		t.Error("tls should be enabled")
	}
	reality := tls["reality"].(map[string]interface{})
	if reality["enabled"] != true {
		t.Error("reality should be enabled")
	}

	transport := out["transport"].(map[string]interface{})
	if transport["type"] != "grpc" {
		t.Errorf("expected grpc, got %v", transport["type"])
	}
	if transport["service_name"] != "svc" {
		t.Errorf("grpc service_name mismatch")
	}
}

func TestGenerate_Trojan(t *testing.T) {
	node := model.ProxyNode{
		Name:     "US-Trojan",
		Protocol: model.ProtoTrojan,
		Address:  "us.example.com",
		Port:     443,
		Password: "secret123",
		TLS:      true,
	}
	data, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	out := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	if out["type"] != "trojan" {
		t.Errorf("expected trojan, got %v", out["type"])
	}
	if out["password"] != "secret123" {
		t.Errorf("password mismatch")
	}
}

func TestGenerate_Shadowsocks(t *testing.T) {
	node := model.ProxyNode{
		Name:     "SG-SS",
		Protocol: model.ProtoShadowsocks,
		Address:  "sg.example.com",
		Port:     8388,
		Password: "sspass",
		SSMethod: "aes-256-gcm",
	}
	data, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	out := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	if out["type"] != "shadowsocks" {
		t.Errorf("expected shadowsocks, got %v", out["type"])
	}
	if out["method"] != "aes-256-gcm" {
		t.Errorf("method mismatch")
	}
}

func TestGenerate_Hysteria2(t *testing.T) {
	node := model.ProxyNode{
		Name:            "HK-Hy2",
		Protocol:        model.ProtoHysteria2,
		Address:         "hk.example.com",
		Port:            443,
		Password:        "hypassword",
		Hysteria2UpMbps: 100,
		Hysteria2DownMbps: 200,
		Hysteria2ObfsType: "salamander",
		Hysteria2ObfsPass: "obfspass",
	}
	data, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	out := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	if out["type"] != "hysteria2" {
		t.Errorf("expected hysteria2, got %v", out["type"])
	}
	if out["up_mbps"].(float64) != 100 {
		t.Errorf("up_mbps mismatch")
	}
	if out["down_mbps"].(float64) != 200 {
		t.Errorf("down_mbps mismatch")
	}
	obfs := out["obfs"].(map[string]interface{})
	if obfs["type"] != "salamander" {
		t.Errorf("obfs type mismatch")
	}
	if obfs["password"] != "obfspass" {
		t.Errorf("obfs password mismatch")
	}
}

func TestGenerate_UnsupportedProtocol(t *testing.T) {
	node := model.ProxyNode{
		Name:     "Bad",
		Protocol: model.Protocol("unknown"),
		Address:  "bad.example.com",
		Port:     443,
	}
	_, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err == nil {
		t.Error("expected error for unsupported protocol")
	}
}

func TestGenerate_RouteRules(t *testing.T) {
	nodes := []model.ProxyNode{
		{Name: "n1", Protocol: model.ProtoVMess, Address: "a1.com", Port: 443, UUID: "u1"},
		{Name: "n2", Protocol: model.ProtoTrojan, Address: "a2.com", Port: 443, Password: "p2"},
	}
	data, _, err := Generate(nodes, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	rules := cfg["route"].(map[string]interface{})["rules"].([]interface{})
	if len(rules) != 2 {
		t.Fatalf("expected 2 route rules, got %d", len(rules))
	}

	rule0 := rules[0].(map[string]interface{})
	inbound0 := rule0["inbound"].([]interface{})
	if inbound0[0] != "socks-in-0" {
		t.Errorf("rule0 inbound mismatch")
	}
	if rule0["outbound"] != "out-0" {
		t.Errorf("rule0 outbound mismatch")
	}

	rule1 := rules[1].(map[string]interface{})
	if rule1["outbound"] != "out-1" {
		t.Errorf("rule1 outbound mismatch")
	}
}

func TestGenerate_AnyTLS(t *testing.T) {
	node := model.ProxyNode{
		Name:                    "HK-AnyTLS",
		Protocol:                model.ProtoAnyTLS,
		Address:                 "hk.example.com",
		Port:                    443,
		Password:                "8JCsPssfgS8tiRwiMlhARg==",
		TLS:                     true,
		SNI:                     "hk.example.com",
		AnyTLSIdleCheckInterval: "30s",
		AnyTLSIdleTimeout:       "30s",
		AnyTLSMinIdleSession:    5,
	}
	data, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	out := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	if out["type"] != "anytls" {
		t.Errorf("expected anytls, got %v", out["type"])
	}
	if out["server"] != "hk.example.com" {
		t.Errorf("server mismatch")
	}
	if out["server_port"].(float64) != 443 {
		t.Errorf("server_port mismatch")
	}
	if out["password"] != "8JCsPssfgS8tiRwiMlhARg==" {
		t.Errorf("password mismatch")
	}
	if out["idle_session_check_interval"] != "30s" {
		t.Errorf("idle_session_check_interval mismatch")
	}
	if out["idle_session_timeout"] != "30s" {
		t.Errorf("idle_session_timeout mismatch")
	}
	if out["min_idle_session"].(float64) != 5 {
		t.Errorf("min_idle_session mismatch")
	}

	// Check TLS
	tls := out["tls"].(map[string]interface{})
	if tls["enabled"] != true {
		t.Error("tls should be enabled")
	}
	if tls["server_name"] != "hk.example.com" {
		t.Errorf("tls server_name mismatch")
	}
}

func TestGenerate_AnyTLS_Minimal(t *testing.T) {
	node := model.ProxyNode{
		Name:     "AnyTLS-Min",
		Protocol: model.ProtoAnyTLS,
		Address:  "srv.example.com",
		Port:     443,
		Password: "pass123",
		TLS:      true,
	}
	data, _, err := Generate([]model.ProxyNode{node}, 20000)
	if err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	var cfg map[string]interface{}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("unmarshal config: %v", err)
	}

	out := cfg["outbounds"].([]interface{})[0].(map[string]interface{})
	if out["type"] != "anytls" {
		t.Errorf("expected anytls, got %v", out["type"])
	}
	// Optional fields should not be present
	if _, ok := out["idle_session_check_interval"]; ok {
		t.Error("idle_session_check_interval should not be set")
	}
	if _, ok := out["idle_session_timeout"]; ok {
		t.Error("idle_session_timeout should not be set")
	}
	if _, ok := out["min_idle_session"]; ok {
		t.Error("min_idle_session should not be set")
	}
}
