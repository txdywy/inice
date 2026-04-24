package shadow

import (
	"context"
	"fmt"

	"github.com/txdywy/inice/internal/model"
	sshutil "github.com/txdywy/inice/internal/ssh"
)

// CoreType identifies the available proxy core.
type CoreType string

const (
	CoreSingBox CoreType = "sing-box"
	CoreXray    CoreType = "xray"
)

// ShadowCore manages a temporary proxy core process on the remote router.
type ShadowCore interface {
	// Type returns the core type identifier.
	Type() CoreType
	// GenerateConfig produces a core-specific JSON config with SOCKS5 inbounds.
	GenerateConfig(nodes []model.ProxyNode) ([]byte, error)
	// Launch starts the core process on the remote and returns its PID.
	Launch(ctx context.Context, client *sshutil.Client, configPath string) (int, error)
	// Kill terminates the core process.
	Kill(ctx context.Context, client *sshutil.Client, pid int) error
	// ConfigPath returns the remote path for the temporary config file.
	ConfigPath() string
	// LogPath returns the remote path for the log file.
	LogPath() string
}

// DetectCore discovers available proxy cores on the remote system.
// Preference: sing-box > xray.
func DetectCore(ctx context.Context, client *sshutil.Client, preferred string) (CoreType, error) {
	if preferred == "xray" {
		return CoreXray, nil
	}

	// Check for sing-box first
	stdout, _, _ := client.ExecuteWithContext(ctx, "which sing-box 2>/dev/null && echo 'FOUND_SINGBOX'")
	if stdout != "" && containsFOUND(stdout) {
		return CoreSingBox, nil
	}

	// Check for xray
	stdout, _, _ = client.ExecuteWithContext(ctx, "which xray 2>/dev/null && echo 'FOUND_XRAY'")
	if stdout != "" && containsFOUND(stdout) {
		return CoreXray, nil
	}

	return "", fmt.Errorf("no proxy core binary found on router; install sing-box via: opkg update && opkg install sing-box")
}

func containsFOUND(s string) bool {
	return len(s) > 0 && (containsSub(s, "FOUND_SINGBOX") || containsSub(s, "FOUND_XRAY"))
}

func containsSub(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
