package shadow

import (
	"fmt"
	"strings"

	"github.com/txdywy/inice/internal/model"
)

// GenerateHysteria2Config builds a single Hysteria2 client YAML config.
func GenerateHysteria2Config(node model.ProxyNode, port int) []byte {
	var builder strings.Builder
	address := node.Address
	if strings.Contains(address, ":") {
		address = fmt.Sprintf("[%s]", address)
	}
	builder.WriteString(fmt.Sprintf("server: \"%s:%d\"\n", address, node.Port))
	if node.Password != "" {
		builder.WriteString(fmt.Sprintf("auth: \"%s\"\n", node.Password))
	}
	
	if node.SNI != "" {
		builder.WriteString("tls:\n")
		builder.WriteString(fmt.Sprintf("  sni: %s\n", node.SNI))
		builder.WriteString("  insecure: true\n")
	} else {
		builder.WriteString("tls:\n  insecure: true\n")
	}

	if node.Hysteria2ObfsType != "" {
		builder.WriteString("obfs:\n")
		builder.WriteString(fmt.Sprintf("  type: %s\n", node.Hysteria2ObfsType))
		if node.Hysteria2ObfsPass != "" {
			builder.WriteString(fmt.Sprintf("  password: %s\n", node.Hysteria2ObfsPass))
		}
	}

	if node.Hysteria2UpMbps > 0 || node.Hysteria2DownMbps > 0 {
		builder.WriteString("bandwidth:\n")
		if node.Hysteria2UpMbps > 0 {
			builder.WriteString(fmt.Sprintf("  up: %d mbps\n", node.Hysteria2UpMbps))
		}
		if node.Hysteria2DownMbps > 0 {
			builder.WriteString(fmt.Sprintf("  down: %d mbps\n", node.Hysteria2DownMbps))
		}
	}

	builder.WriteString("socks5:\n")
	builder.WriteString(fmt.Sprintf("  listen: 0.0.0.0:%d\n", port))

	return []byte(builder.String())
}
