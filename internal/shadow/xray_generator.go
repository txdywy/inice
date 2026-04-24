package shadow

import (
	"encoding/json"
	"fmt"

	"github.com/txdywy/inice/internal/model"
)

// GenerateXrayConfig builds a multi-inbound/outbound Xray JSON config.
func GenerateXrayConfig(nodes []model.ProxyNode, portMap map[int]int) ([]byte, error) {
	inbounds := make([]map[string]interface{}, 0, len(nodes))
	outbounds := make([]map[string]interface{}, 0, len(nodes)+1)
	rules := make([]map[string]interface{}, 0, len(nodes))

	for i, node := range nodes {
		inTag := fmt.Sprintf("in-%d", i)
		outTag := fmt.Sprintf("out-%d", i)

		// Inbound
		inbounds = append(inbounds, map[string]interface{}{
			"tag":      inTag,
			"port":     portMap[i],
			"protocol": "socks",
			"listen":   "0.0.0.0",
			"settings": map[string]interface{}{
				"auth": "noauth",
				"udp":  true,
			},
		})

		// Outbound
		outbound := map[string]interface{}{
			"tag":      outTag,
			"protocol": string(node.Protocol),
		}

		settings := map[string]interface{}{}
		switch node.Protocol {
		case model.ProtoVLESS:
			user := map[string]interface{}{
				"id":         node.UUID,
				"encryption": "none",
			}
			if node.Flow != "" {
				user["flow"] = node.Flow
			}
			settings["vnext"] = []interface{}{
				map[string]interface{}{
					"address": node.Address,
					"port":    node.Port,
					"users":   []interface{}{user},
				},
			}
		case model.ProtoVMess:
			settings["vnext"] = []interface{}{
				map[string]interface{}{
					"address": node.Address,
					"port":    node.Port,
					"users": []interface{}{
						map[string]interface{}{
							"id":       node.UUID,
							"alterId":  0,
							"security": node.Security,
						},
					},
				},
			}
		case model.ProtoTrojan:
			settings["servers"] = []interface{}{
				map[string]interface{}{
					"address":  node.Address,
					"port":     node.Port,
					"password": node.Password,
				},
			}
		case model.ProtoShadowsocks:
			settings["servers"] = []interface{}{
				map[string]interface{}{
					"address":  node.Address,
					"port":     node.Port,
					"method":   node.SSMethod,
					"password": node.Password,
				},
			}
		}
		outbound["settings"] = settings

		// StreamSettings
		streamSettings := map[string]interface{}{
			"network": "tcp",
		}

		if node.Transport != "" && node.Transport != "tcp" && node.Transport != "raw" {
			streamSettings["network"] = node.Transport
			switch node.Transport {
			case "ws":
				streamSettings["wsSettings"] = map[string]interface{}{
					"path": node.WSPath,
				}
				if node.WSHost != "" {
					streamSettings["wsSettings"].(map[string]interface{})["headers"] = map[string]string{
						"Host": node.WSHost,
					}
				}
			case "grpc":
				streamSettings["grpcSettings"] = map[string]interface{}{
					"serviceName": node.GRPCServiceName,
				}
			case "httpupgrade":
				streamSettings["network"] = "httpupgrade"
				streamSettings["httpupgradeSettings"] = map[string]interface{}{
					"path": node.WSPath,
					"host": node.WSHost,
				}
			case "xhttp":
				streamSettings["network"] = "xhttp"
				streamSettings["xhttpSettings"] = map[string]interface{}{
					"path": node.WSPath,
					"host": node.WSHost,
				}
			}
		}

		if node.TLS || node.Reality {
			if node.Reality {
				streamSettings["security"] = "reality"
				realitySettings := map[string]interface{}{
					"publicKey": node.RealityPublicKey,
					"shortId":   node.RealityShortID,
					"fingerprint": node.Fingerprint,
					"serverName": node.SNI,
					"spiderX": "/",
				}
				if realitySettings["fingerprint"] == "" {
					realitySettings["fingerprint"] = "chrome"
				}
				streamSettings["realitySettings"] = realitySettings
			} else {
				streamSettings["security"] = "tls"
				tlsSettings := map[string]interface{}{
					"serverName": node.SNI,
					"allowInsecure": true,
				}
				if node.Fingerprint != "" {
					tlsSettings["fingerprint"] = node.Fingerprint
				}
				if len(node.ALPN) > 0 {
					tlsSettings["alpn"] = node.ALPN
				}
				streamSettings["tlsSettings"] = tlsSettings
			}
		}

		outbound["streamSettings"] = streamSettings
		outbounds = append(outbounds, outbound)

		// Rule
		rules = append(rules, map[string]interface{}{
			"type":        "field",
			"inboundTag":  []string{inTag},
			"outboundTag": outTag,
		})
	}

	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "error",
		},
		"inbounds":  inbounds,
		"outbounds": outbounds,
		"routing": map[string]interface{}{
			"domainStrategy": "AsIs",
			"rules":          rules,
		},
	}

	return json.MarshalIndent(config, "", "  ")
}
