package shadow

import (
	"context"
	"fmt"
	"time"

	"github.com/txdywy/inice/internal/model"
	sshutil "github.com/txdywy/inice/internal/ssh"
)

type managedProcess struct {
	pid     int
	logPath string
	name    string
}

// Orchestrator manages the lifecycle of shadow proxies on the router.
type Orchestrator struct {
	sshClient *sshutil.Client
	routerIP  string
	basePort  int
	tempDir   string
	processes []managedProcess
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
		sshClient: sshClient,
		routerIP:  routerIP,
		basePort:  opts.BasePort,
		tempDir:   tempDir,
	}
}

// Setup generates configs, uploads them, and starts the processes.
func (o *Orchestrator) Setup(ctx context.Context, nodes []model.ProxyNode) ([]model.ProxyNode, error) {
	// Create remote temp directory
	_, _, exitCode, err := o.sshClient.Execute(ctx, "mkdir -p "+o.tempDir)
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	if exitCode != 0 {
		return nil, fmt.Errorf("create temp dir failed with exit code %d", exitCode)
	}

	result := make([]model.ProxyNode, len(nodes))
	
	type coreGroup struct {
		nodes   []model.ProxyNode
		portMap map[int]int
	}
	singboxGroup := coreGroup{portMap: make(map[int]int)}
	xrayGroup := coreGroup{portMap: make(map[int]int)}

	// 1. Assign ports and group nodes
	for i, node := range nodes {
		port := o.basePort + i
		node.SOCKS5Port = port
		result[i] = node

		switch node.Type {
		case model.NodeTypeXray:
			xrayGroup.nodes = append(xrayGroup.nodes, node)
			xrayGroup.portMap[len(xrayGroup.nodes)-1] = port
		case model.NodeTypeHysteria2:
			// Hysteria2 is handled individually
			cfgData := GenerateHysteria2Config(node, port)
			cfgPath := fmt.Sprintf("%s/hysteria_%d.yaml", o.tempDir, i)
			logPath := fmt.Sprintf("%s/hysteria_%d.log", o.tempDir, i)
			
			if err := o.sshClient.UploadFile(ctx, cfgPath, cfgData); err != nil {
				o.Teardown(ctx)
				return nil, fmt.Errorf("upload hysteria config: %w", err)
			}
			cmd := fmt.Sprintf("/usr/bin/hysteria -c %s server", cfgPath) // client mode is just implicit or requires client subcommand
			// Wait, hysteria v2 client mode is `hysteria -c config.yaml`
			cmd = fmt.Sprintf("/usr/bin/hysteria -c %s", cfgPath)
			pid, err := o.sshClient.StartBackground(ctx, cmd, logPath)
			if err != nil {
				o.Teardown(ctx)
				return nil, fmt.Errorf("start hysteria: %w", err)
			}
			o.processes = append(o.processes, managedProcess{pid: pid, logPath: logPath, name: "hysteria"})

		default:
			singboxGroup.nodes = append(singboxGroup.nodes, node)
			singboxGroup.portMap[len(singboxGroup.nodes)-1] = port
		}
	}

	// 2. Setup sing-box
	if len(singboxGroup.nodes) > 0 {
		cfgData, err := GenerateSingboxConfig(singboxGroup.nodes, singboxGroup.portMap)
		if err != nil {
			o.Teardown(ctx)
			return nil, fmt.Errorf("generate singbox config: %w", err)
		}
		cfgPath := o.tempDir + "/singbox.json"
		logPath := o.tempDir + "/singbox.log"
		if err := o.sshClient.UploadFile(ctx, cfgPath, cfgData); err != nil {
			o.Teardown(ctx)
			return nil, fmt.Errorf("upload singbox config: %w", err)
		}
		cmd := fmt.Sprintf("/usr/bin/sing-box run -c %s -D %s", cfgPath, o.tempDir)
		pid, err := o.sshClient.StartBackground(ctx, cmd, logPath)
		if err != nil {
			o.Teardown(ctx)
			return nil, fmt.Errorf("start singbox: %w", err)
		}
		o.processes = append(o.processes, managedProcess{pid: pid, logPath: logPath, name: "sing-box"})
	}

	// 3. Setup xray
	if len(xrayGroup.nodes) > 0 {
		cfgData, err := GenerateXrayConfig(xrayGroup.nodes, xrayGroup.portMap)
		if err != nil {
			o.Teardown(ctx)
			return nil, fmt.Errorf("generate xray config: %w", err)
		}
		cfgPath := o.tempDir + "/xray.json"
		logPath := o.tempDir + "/xray.log"
		if err := o.sshClient.UploadFile(ctx, cfgPath, cfgData); err != nil {
			o.Teardown(ctx)
			return nil, fmt.Errorf("upload xray config: %w", err)
		}
		cmd := fmt.Sprintf("/usr/bin/xray run -c %s", cfgPath)
		pid, err := o.sshClient.StartBackground(ctx, cmd, logPath)
		if err != nil {
			o.Teardown(ctx)
			return nil, fmt.Errorf("start xray: %w", err)
		}
		o.processes = append(o.processes, managedProcess{pid: pid, logPath: logPath, name: "xray"})
	}

	// 4. Wait for processes to initialize
	time.Sleep(1 * time.Second)

	// 5. Verify processes are alive
	for _, p := range o.processes {
		if alive, _ := o.isAlive(ctx, p.pid); !alive {
			logContent, _ := o.sshClient.ReadFile(ctx, p.logPath, 4096)
			o.Teardown(ctx)
			if logContent != "" {
				return nil, fmt.Errorf("%s process (pid=%d) died immediately after start. Log output:\n%s", p.name, p.pid, logContent)
			}
			return nil, fmt.Errorf("%s process (pid=%d) died immediately after start", p.name, p.pid)
		}
	}

	return result, nil
}

// Teardown stops the managed processes and removes temp files.
func (o *Orchestrator) Teardown(ctx context.Context) error {
	var errs []error

	// Kill processes
	for _, p := range o.processes {
		if p.pid > 0 {
			if err := o.sshClient.KillProcess(ctx, p.pid); err != nil {
				errs = append(errs, fmt.Errorf("kill %s process: %w", p.name, err))
			}
		}
	}
	o.processes = nil

	// Remove temp directory
	if o.tempDir != "" {
		if err := o.sshClient.RemoveDir(ctx, o.tempDir); err != nil {
			errs = append(errs, fmt.Errorf("remove temp dir: %w", err))
		}
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

// IsAlive checks if all managed processes are still running.
func (o *Orchestrator) IsAlive(ctx context.Context) bool {
	for _, p := range o.processes {
		if alive, _ := o.isAlive(ctx, p.pid); !alive {
			return false
		}
	}
	return true
}

func (o *Orchestrator) isAlive(ctx context.Context, pid int) (bool, error) {
	if pid <= 0 {
		return false, nil
	}
	_, _, exitCode, err := o.sshClient.Execute(ctx, fmt.Sprintf("kill -0 %d", pid))
	if err != nil {
		return false, err
	}
	return exitCode == 0, nil
}
