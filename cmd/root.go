package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/txdywy/inice/internal/config"
	sshutil "github.com/txdywy/inice/internal/ssh"
)

var (
	routerHost     string
	routerPort     int
	routerUser     string
	routerPassword string
	routerKeyFile  string
	configFile     string
)

var rootCmd = &cobra.Command{
	Use:   "inice",
	Short: "PassWall2 proxy inventory reader",
	Long: `inice connects to an iStoreOS router via SSH, reads PassWall2 configuration,
and prints a read-only inventory of configured proxy nodes.

This build is intentionally non-invasive: it does not write remote files,
start remote processes, or modify PassWall2 configuration.`,
	RunE: run,
}

func init() {
	rootCmd.Flags().StringVar(&routerHost, "router", "", "Router hostname or IP")
	rootCmd.Flags().IntVar(&routerPort, "port", 0, "SSH port (default: 22)")
	rootCmd.Flags().StringVar(&routerUser, "user", "", "SSH user (default: root)")
	rootCmd.Flags().StringVar(&routerPassword, "password", "", "SSH password")
	rootCmd.Flags().StringVar(&routerKeyFile, "key-file", "", "SSH private key file path")
	rootCmd.Flags().StringVar(&configFile, "config", "", "Config file path (default: ~/.inice.yaml)")
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
		fmt.Println("\nReceived interrupt, stopping read-only session...")
		cancel()
	}()

	fmt.Println("Reading PassWall2 configuration...")
	uciOutput, _, err := sshClient.ReadPassWall2(ctx)
	if err != nil {
		return fmt.Errorf("read UCI: %w", err)
	}
	if uciOutput == "" {
		return fmt.Errorf("no PassWall2 configuration found; is PassWall2 installed?")
	}

	nodes, err := config.ParseUCIOutput(uciOutput)
	if err != nil {
		return fmt.Errorf("parse UCI: %w", err)
	}

	fmt.Printf("Found %d proxy nodes\n", len(nodes))
	for _, n := range nodes {
		fmt.Printf("- %s | %s | %s | %s:%d\n", n.Name, n.Type, n.Protocol, n.Address, n.Port)
	}

	fmt.Println("\nRead-only mode complete. No remote files were written and no router processes were started.")
	return nil
}
