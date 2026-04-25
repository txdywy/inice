package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/txdywy/inice/internal/config"
	"github.com/txdywy/inice/internal/engine"
	"github.com/txdywy/inice/internal/model"
	"github.com/txdywy/inice/internal/report"
	"github.com/txdywy/inice/internal/shadow"
	sshutil "github.com/txdywy/inice/internal/ssh"
)

var (
	routerHost     string
	routerPort     int
	routerUser     string
	routerPassword string
	routerKeyFile  string
	configFile     string
	testMode       bool
	outputFormat   string
)

var rootCmd = &cobra.Command{
	Use:   "inice",
	Short: "PassWall2 proxy inventory reader and tester",
	Long: `inice connects to an iStoreOS router via SSH, reads PassWall2 configuration,
and prints a read-only inventory of configured proxy nodes.

With --test, it starts shadow sing-box proxies on the router and runs
health checks (latency, exit IP, streaming unlock) through each node.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVar(&routerHost, "router", "", "Router hostname or IP")
	rootCmd.Flags().IntVar(&routerPort, "port", 0, "SSH port (default: 22)")
	rootCmd.Flags().StringVar(&routerUser, "user", "", "SSH user (default: root)")
	rootCmd.Flags().StringVar(&routerPassword, "password", "", "SSH password")
	rootCmd.Flags().StringVar(&routerKeyFile, "key-file", "", "SSH private key file path")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Config file path (default: ~/.inice.yaml)")
	rootCmd.Flags().BoolVar(&testMode, "test", false, "Run proxy health tests through each node")
	rootCmd.Flags().StringVar(&outputFormat, "format", "", "Output format: table, json, csv (overrides config)")
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if routerHost != "" {
		cfg.Router.Host = routerHost
	}
	if routerPort > 0 {
		cfg.Router.Port = routerPort
	}
	if routerUser != "" {
		cfg.Router.User = routerUser
	}
	if routerPassword != "" {
		cfg.Router.Password = routerPassword
	}
	if routerKeyFile != "" {
		cfg.Router.KeyFile = routerKeyFile
	}
	if outputFormat != "" {
		cfg.Output.Format = outputFormat
	}

	if cfg.Router.Host == "" {
		return fmt.Errorf("router host is required (set via --router, config file, or INICE_ROUTER_HOST env)")
	}

	if cfg.Router.Password == "" && cfg.Router.KeyFile == "" {
		prompted, err := sshutil.PromptPassword("请输入路由器 SSH 密码: ")
		if err != nil {
			return fmt.Errorf("SSH password prompt: %w", err)
		}
		cfg.Router.Password = prompted
	}

	auth, err := sshutil.AuthMethod(cfg.Router.Password, cfg.Router.KeyFile)
	if err != nil {
		return fmt.Errorf("SSH auth: %w", err)
	}

	fmt.Printf("Connecting to %s:%d as %s ...\n", cfg.Router.Host, cfg.Router.Port, cfg.Router.User)

	sshClient, err := sshutil.Dial(cfg.Router.Host, cfg.Router.Port, cfg.Router.User, auth)
	if err != nil {
		return fmt.Errorf("SSH connect: %w", err)
	}
	defer sshClient.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, stopping...")
		cancel()
	}()

	fmt.Println("Reading PassWall2 configuration...")
	uciOutput, uciErrOutput, err := sshClient.ReadPassWall2(ctx)
	if err != nil {
		if uciErrOutput != "" {
			return fmt.Errorf("read UCI: %w: %s", err, uciErrOutput)
		}
		return fmt.Errorf("read UCI: %w", err)
	}
	if uciOutput == "" {
		return fmt.Errorf("no PassWall2 configuration found; is PassWall2 installed?")
	}

	nodes, err := config.ParseUCIOutput(uciOutput)
	if err != nil {
		return fmt.Errorf("parse UCI: %w", err)
	}

	if !testMode {
		// Inventory mode: print node list only
		fmt.Printf("Found %d proxy nodes\n", len(nodes))
		for _, n := range nodes {
			fmt.Printf("- %s | %s | %s | %s:%d\n", n.Name, n.Type, n.Protocol, n.Address, n.Port)
		}
		fmt.Println("\nRead-only mode complete. No remote files were written and no router processes were started.")
		return nil
	}

	// Testing mode
	orch := shadow.New(sshClient, cfg.Router.Host, shadow.Options{
		BasePort: cfg.Shadow.BasePort,
	})
	defer func() {
		fmt.Println("\nCleaning up shadow proxies...")
		// Use a separate context for teardown so cleanup succeeds even
		// if the main ctx was cancelled (e.g. Ctrl+C).
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := orch.Teardown(cleanupCtx); err != nil {
			fmt.Fprintf(os.Stderr, "cleanup warning: %v\n", err)
		}
	}()

	fmt.Printf("Found %d proxy nodes\n", len(nodes))
	fmt.Println("Setting up shadow sing-box proxies on router...")

	nodes, err = orch.Setup(ctx, nodes)
	if err != nil {
		return fmt.Errorf("setup shadow proxies: %w", err)
	}

	testCfg, err := config.ParseTestConfig(cfg)
	if err != nil {
		return fmt.Errorf("parse test config: %w", err)
	}

	// Setup renderer for streaming results
	renderer, err := report.NewRenderer(cfg.Output.Format)
	if err != nil {
		return fmt.Errorf("create renderer: %w", err)
	}

	fmt.Printf("Running tests (concurrency=%d, timeout=%s)...\n", testCfg.Concurrency, testCfg.Timeout)
	
	renderer.RenderHeader(cfg.Router.Host, len(nodes), "sing-box", "")

	var mu sync.Mutex
	var completed int
	total := len(nodes)
	doneCh := make(chan struct{})

	isTable := cfg.Output.Format == "table" || cfg.Output.Format == ""

	if isTable {
		go func() {
			spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
			idx := 0
			startTime := time.Now()
			for {
				select {
				case <-doneCh:
					return
				case <-time.After(100 * time.Millisecond):
					mu.Lock()
					comp := completed
					mu.Unlock()
					
					barLen := 20
					filled := 0
					if total > 0 {
						filled = int(float64(comp) / float64(total) * float64(barLen))
					}
					bar := strings.Repeat("█", filled) + strings.Repeat("░", barLen-filled)
					elapsed := time.Since(startTime).Round(time.Second)
					
					fmt.Printf("\r\033[36m%s\033[0m Progress: [\033[32m%s\033[0m] %d/%d | Time: \033[33m%s\033[0m ", 
						spinner[idx%len(spinner)], bar, comp, total, elapsed)
					idx++
				}
			}
		}()
	}

	runner := engine.NewRunner(cfg.Router.Host, testCfg)
	var activeResults []model.TestResult
	
	results := runner.RunTests(ctx, nodes, func(idx int, totalNodes int, res model.TestResult) {
		if isTable {
			mu.Lock()
			completed++
			
			// 1. Add to active list
			activeResults = append(activeResults, res)
			
			// 2. Sort active results by score (descending)
			sort.Slice(activeResults, func(i, j int) bool {
				si, _ := report.CalculateScore(activeResults[i])
				sj, _ := report.CalculateScore(activeResults[j])
				return si > sj
			})
			
			// 3. Clear previously printed table rows (including progress bar)
			// Each result completion redraws the table header and all rows.
			// Move cursor up: 1 (progress bar) + len(activeResults)-1 (previous rows) + 3 (table header)
			if completed > 1 {
				linesToMove := (completed - 1) + 3 + 1
				fmt.Printf("\033[%dA\033[J", linesToMove)
			} else {
				// For the very first row, just clear the progress bar
				fmt.Print("\r\033[K")
			}
			
			// 4. Redraw table header and all sorted rows
			renderer.RenderTableHeader()
			for i, r := range activeResults {
				renderer.RenderRow(r, i+1)
			}
			
			mu.Unlock()
		} else {
			renderer.RenderRow(res, idx+1)
		}
	})

	if isTable {
		close(doneCh)
		fmt.Print("\r\033[K") // clear progress bar line at the end
	}

	// Final results should be the sorted ones
	results = activeResults

	if err := renderer.RenderSummary(results); err != nil {
		return fmt.Errorf("render summary: %w", err)
	}

	return nil
}
