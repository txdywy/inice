package shadow

import (
	"context"
	"fmt"
	"time"

	"github.com/txdywy/inice/internal/model"
	sshutil "github.com/txdywy/inice/internal/ssh"
)

// Orchestrator manages the lifecycle of a shadow sing-box process on the router.
type Orchestrator struct {
	sshClient  *sshutil.Client
	routerIP   string
	basePort   int
	singBoxPath string
	tempDir    string
	configPath string
	pid        int
	started    bool
}

// Options configures the orchestrator.
type Options struct {
	BasePort    int
	TempDir     string // optional, defaults to /tmp/inice-<timestamp>
	SingBoxPath string // optional, auto-detected if empty
}

// New creates a new shadow orchestrator.
func New(sshClient *sshutil.Client, routerIP string, opts Options) *Orchestrator {
	tempDir := opts.TempDir
	if tempDir == "" {
		tempDir = fmt.Sprintf("/tmp/inice-%d", time.Now().Unix())
	}
	return &Orchestrator{
		sshClient:   sshClient,
		routerIP:    routerIP,
		basePort:    opts.BasePort,
		singBoxPath: opts.SingBoxPath,
		tempDir:     tempDir,
		configPath:  tempDir + "/config.json",
	}
}

// Setup generates the sing-box config, uploads it, and starts the process.
// Returns nodes with SOCKS5Port populated.
func (o *Orchestrator) Setup(ctx context.Context, nodes []model.ProxyNode) ([]model.ProxyNode, error) {
	// Detect sing-box binary if not provided
	if o.singBoxPath == "" {
		path, err := o.sshClient.Which(ctx, "sing-box")
		if err != nil {
			return nil, fmt.Errorf("sing-box not found on router; please install it: %w", err)
		}
		o.singBoxPath = path
	}

	// Generate config
	configData, portMap, err := Generate(nodes, o.basePort)
	if err != nil {
		return nil, fmt.Errorf("generate shadow config: %w", err)
	}

	// Create remote temp directory
	_, _, exitCode, err := o.sshClient.Execute(ctx, "mkdir -p "+o.tempDir)
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("create temp dir failed with exit code %d", exitCode)
	}

	// Upload config
	if err := o.sshClient.UploadFile(ctx, o.configPath, configData); err != nil {
		_ = o.Teardown(ctx)
		return nil, fmt.Errorf("upload config: %w", err)
	}

	// Start sing-box in background
	cmd := fmt.Sprintf("%s run -c %s -D %s", o.singBoxPath, o.configPath, o.tempDir)
	pid, err := o.sshClient.StartBackground(ctx, cmd)
	if err != nil {
		_ = o.Teardown(ctx)
		return nil, fmt.Errorf("start sing-box: %w", err)
	}
	o.pid = pid
	o.started = true

	// Wait briefly for sing-box to initialize
	time.Sleep(500 * time.Millisecond)

	// Verify process is alive
	if alive, _ := o.isAlive(ctx); !alive {
		_ = o.Teardown(ctx)
		return nil, fmt.Errorf("sing-box process (pid=%d) died immediately after start", pid)
	}

	// Populate SOCKS5 ports on nodes
	result := make([]model.ProxyNode, len(nodes))
	for i, node := range nodes {
		node.SOCKS5Port = portMap[i]
		result[i] = node
	}

	return result, nil
}

// Teardown stops the sing-box process and removes temp files.
// It is safe to call multiple times (idempotent).
func (o *Orchestrator) Teardown(ctx context.Context) error {
	if !o.started {
		return nil
	}

	var errs []error

	// Kill process
	if o.pid > 0 {
		if err := o.sshClient.KillProcess(ctx, o.pid); err != nil {
			errs = append(errs, fmt.Errorf("kill process: %w", err))
		}
	}

	// Remove temp directory
	if o.tempDir != "" {
		if err := o.sshClient.RemoveDir(ctx, o.tempDir); err != nil {
			errs = append(errs, fmt.Errorf("remove temp dir: %w", err))
		}
	}

	o.started = false
	o.pid = 0

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// IsAlive checks if the sing-box process is still running.
func (o *Orchestrator) IsAlive(ctx context.Context) bool {
	alive, _ := o.isAlive(ctx)
	return alive
}

func (o *Orchestrator) isAlive(ctx context.Context) (bool, error) {
	if o.pid <= 0 {
		return false, nil
	}
	_, _, exitCode, err := o.sshClient.Execute(ctx, fmt.Sprintf("kill -0 %d", o.pid))
	if err != nil {
		return false, err
	}
	return exitCode == 0, nil
}
