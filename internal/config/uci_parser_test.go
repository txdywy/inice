package config

import (
	"os"
	"testing"
)

func TestParseUCIOutput(t *testing.T) {
	// Read testdata
	data, err := os.ReadFile("../../testdata/uci_show_passwall2.txt")
	if err != nil {
		t.Fatalf("cannot read testdata: %v", err)
	}

	nodes, err := ParseUCIOutput(string(data))
	if err != nil {
		t.Fatalf("ParseUCIOutput failed: %v", err)
	}

	if len(nodes) != 4 {
		t.Fatalf("expected 4 nodes, got %d", len(nodes))
	}

	// Node 0: HK-VMess-WS (Xray type)
	n0 := nodes[0]
	if n0.Name != "HK-VMess-WS" {
		t.Errorf("node 0 name: expected HK-VMess-WS, got %s", n0.Name)
	}
	if n0.Type != "Xray" {
		t.Errorf("node 0 type: expected Xray, got %s", n0.Type)
	}
	if n0.Protocol != "vmess" {
		t.Errorf("node 0 protocol: expected vmess, got %s", n0.Protocol)
	}
	if n0.Address != "hk1.example.com" {
		t.Errorf("node 0 address: expected hk1.example.com, got %s", n0.Address)
	}
	if n0.Port != 443 {
		t.Errorf("node 0 port: expected 443, got %d", n0.Port)
	}
	if n0.UUID != "12345678-1234-1234-1234-123456789abc" {
		t.Errorf("node 0 uuid mismatch")
	}
	if n0.Transport != "ws" {
		t.Errorf("node 0 transport: expected ws, got %s", n0.Transport)
	}
	if n0.WSHost != "hk1.example.com" {
		t.Errorf("node 0 ws_host mismatch")
	}
	if !n0.TLS {
		t.Error("node 0: TLS should be enabled")
	}

	// Node 1: US-Trojan (sing-box type)
	n1 := nodes[1]
	if n1.Name != "US-Trojan" {
		t.Errorf("node 1 name: expected US-Trojan, got %s", n1.Name)
	}
	if n1.Type != "sing-box" {
		t.Errorf("node 1 type: expected sing-box, got %s", n1.Type)
	}
	if n1.Protocol != "trojan" {
		t.Errorf("node 1 protocol: expected trojan, got %s", n1.Protocol)
	}
	if n1.Password != "mypassword123" {
		t.Errorf("node 1 password mismatch")
	}

	// Node 2: JP-VLESS-Reality (Xray type)
	n2 := nodes[2]
	if n2.Name != "JP-VLESS-Reality" {
		t.Errorf("node 2 name: expected JP-VLESS-Reality, got %s", n2.Name)
	}
	if n2.Protocol != "vless" {
		t.Errorf("node 2 protocol: expected vless, got %s", n2.Protocol)
	}
	if n2.Flow != "xtls-rprx-vision" {
		t.Errorf("node 2 flow: expected xtls-rprx-vision, got %s", n2.Flow)
	}
	if !n2.Reality {
		t.Error("node 2: Reality should be enabled")
	}
	if n2.Transport != "grpc" {
		t.Errorf("node 2 transport: expected grpc, got %s", n2.Transport)
	}

	// Node 3: SG-Hysteria2 (hysteria2 type)
	n3 := nodes[3]
	if n3.Name != "SG-Hysteria2" {
		t.Errorf("node 3 name: expected SG-Hysteria2, got %s", n3.Name)
	}
	if n3.Type != "hysteria2" {
		t.Errorf("node 3 type: expected hysteria2, got %s", n3.Type)
	}
	if n3.Protocol != "hysteria2" {
		t.Errorf("node 3 protocol: expected hysteria2, got %s", n3.Protocol)
	}
	if n3.Hysteria2UpMbps != 100 {
		t.Errorf("node 3 up_mbps: expected 100, got %d", n3.Hysteria2UpMbps)
	}
	if n3.Hysteria2ObfsType != "salamander" {
		t.Errorf("node 3 obfs type: expected salalamander, got %s", n3.Hysteria2ObfsType)
	}
}

func TestParseUCIOutput_Empty(t *testing.T) {
	_, err := ParseUCIOutput("")
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestParseUCIOutput_NamedSectionsAndQuotedValues(t *testing.T) {
	output := `
passwall2.main=global
passwall2.node-abcd1234='nodes'
passwall2.node-abcd1234.type='Xray'
passwall2.node-abcd1234.protocol='vless'
passwall2.node-abcd1234.remarks='HK Named Node'
passwall2.node-abcd1234.xray_address='hk.example.com'
passwall2.node-abcd1234.xray_port='443'
passwall2.node-abcd1234.xray_uuid='12345678-1234-1234-1234-123456789abc'
passwall2.node-abcd1234.xray_tls='1'
`

	nodes, err := ParseUCIOutput(output)
	if err != nil {
		t.Fatalf("ParseUCIOutput failed: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 node, got %d", len(nodes))
	}
	if nodes[0].Name != "HK Named Node" {
		t.Errorf("node name: expected HK Named Node, got %s", nodes[0].Name)
	}
	if nodes[0].Type != "Xray" {
		t.Errorf("node type: expected Xray, got %s", nodes[0].Type)
	}
	if nodes[0].Address != "hk.example.com" {
		t.Errorf("node address: expected hk.example.com, got %s", nodes[0].Address)
	}
	if nodes[0].Port != 443 {
		t.Errorf("node port: expected 443, got %d", nodes[0].Port)
	}
	if !nodes[0].TLS {
		t.Error("node TLS should be enabled")
	}
}
