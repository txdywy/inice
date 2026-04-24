package sshutil

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

// Client wraps an SSH connection with convenience methods.
type Client struct {
	conn *ssh.Client
}

// AuthMethod returns an ssh.AuthMethod based on available credentials.
func AuthMethod(password, keyFile string) (ssh.AuthMethod, error) {
	if keyFile != "" {
		key, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("cannot read key file %s: %w", keyFile, err)
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("cannot parse key file %s: %w", keyFile, err)
		}
		return ssh.PublicKeys(signer), nil
	}
	if password != "" {
		return ssh.Password(password), nil
	}
	return nil, fmt.Errorf("no password or key_file provided")
}

// PromptPassword securely reads an SSH password from the terminal.
func PromptPassword(prompt string) (string, error) {
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("stdin is not a terminal; please provide --password, INICE_SSH_PASSWORD, or --key-file")
	}

	fmt.Print(prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return "", fmt.Errorf("read password: %w", err)
	}
	return string(password), nil
}

// Dial connects to an SSH server and returns a Client.
func Dial(host string, port int, user string, auth ssh.AuthMethod) (*Client, error) {
	config := &ssh.ClientConfig{
		User:            user,
		Auth:            []ssh.AuthMethod{auth},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: support known_hosts
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", addr, err)
	}

	return &Client{conn: conn}, nil
}

const readPassWall2Command = "/sbin/uci -q show passwall2"

// ReadPassWall2 runs the only remote command this tool needs. Keep remote
// execution narrow so inventory reads cannot grow into router mutation paths.
func (c *Client) ReadPassWall2(ctx context.Context) (string, string, error) {
	return c.runReadOnly(ctx, readPassWall2Command)
}

func (c *Client) runReadOnly(ctx context.Context, cmd string) (string, string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	errCh := make(chan error, 1)
	go func() {
		errCh <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return stdout.String(), stderr.String(), ctx.Err()
	case err := <-errCh:
		return stdout.String(), stderr.String(), err
	}
}

// Execute runs a remote command and returns stdout, stderr, exit code, and error.
func (c *Client) Execute(ctx context.Context, cmd string) (string, string, int, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", "", -1, fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	errCh := make(chan error, 1)
	go func() {
		errCh <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return stdout.String(), stderr.String(), -1, ctx.Err()
	case err := <-errCh:
		exitCode := 0
		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				exitCode = exitErr.ExitStatus()
			} else {
				return stdout.String(), stderr.String(), -1, fmt.Errorf("run command: %w", err)
			}
		}
		return stdout.String(), stderr.String(), exitCode, nil
	}
}

// UploadFile writes data to a remote file using stdin redirection.
func (c *Client) UploadFile(ctx context.Context, remotePath string, data []byte) error {
	session, err := c.conn.NewSession()
	if err != nil {
		return fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	errCh := make(chan error, 1)
	go func() {
		errCh <- session.Run("cat > '" + remotePath + "'")
	}()

	_, writeErr := stdin.Write(data)
	stdin.Close()

	select {
	case <-ctx.Done():
		session.Signal(ssh.SIGKILL)
		return ctx.Err()
	case err := <-errCh:
		if writeErr != nil {
			return fmt.Errorf("write data: %w", writeErr)
		}
		if err != nil {
			return fmt.Errorf("upload file: %w", err)
		}
		return nil
	}
}

// StartBackground starts a command in the background and returns its PID.
// Uses a shell wrapper to emit the shell's PID before exec.
func (c *Client) StartBackground(ctx context.Context, cmd string) (int, error) {
	// Use nohup and echo $$ to get the PID reliably
	wrapped := fmt.Sprintf("nohup sh -c 'echo $$; exec %s' >/dev/null 2>&1 &", cmd)
	stdout, _, exitCode, err := c.Execute(ctx, wrapped)
	if err != nil {
		return 0, fmt.Errorf("start background: %w", err)
	}
	if exitCode != 0 {
		return 0, fmt.Errorf("start background failed with exit code %d", exitCode)
	}

	// Parse PID from first line of output
	pidStr := strings.TrimSpace(stdout)
	if pidStr == "" {
		return 0, fmt.Errorf("could not read PID from background process")
	}
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return 0, fmt.Errorf("parse PID %q: %w", pidStr, err)
	}
	return pid, nil
}

// KillProcess sends SIGTERM then SIGKILL if the process still exists.
func (c *Client) KillProcess(ctx context.Context, pid int) error {
	// Try SIGTERM first
	_, _, _, _ = c.Execute(ctx, fmt.Sprintf("kill -TERM %d 2>/dev/null || true", pid))

	// Wait a bit for graceful shutdown
	time.Sleep(200 * time.Millisecond)

	// Check if still alive, then SIGKILL
	_, _, exitCode, _ := c.Execute(ctx, fmt.Sprintf("kill -0 %d 2>/dev/null", pid))
	if exitCode == 0 {
		_, _, _, _ = c.Execute(ctx, fmt.Sprintf("kill -KILL %d 2>/dev/null || true", pid))
	}
	return nil
}

// RemoveFile removes a remote file.
func (c *Client) RemoveFile(ctx context.Context, path string) error {
	_, _, exitCode, err := c.Execute(ctx, "rm -f '"+path+"'")
	if err != nil {
		return fmt.Errorf("remove file: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("remove file failed with exit code %d", exitCode)
	}
	return nil
}

// RemoveDir removes a remote directory recursively.
func (c *Client) RemoveDir(ctx context.Context, path string) error {
	_, _, exitCode, err := c.Execute(ctx, "rm -rf '"+path+"'")
	if err != nil {
		return fmt.Errorf("remove dir: %w", err)
	}
	if exitCode != 0 {
		return fmt.Errorf("remove dir failed with exit code %d", exitCode)
	}
	return nil
}

// FileExists checks if a remote file exists.
func (c *Client) FileExists(ctx context.Context, path string) (bool, error) {
	_, _, exitCode, err := c.Execute(ctx, "test -f '"+path+"'")
	if err != nil {
		return false, err
	}
	return exitCode == 0, nil
}

// Which finds the path of a binary on the remote system.
func (c *Client) Which(ctx context.Context, binary string) (string, error) {
	stdout, _, exitCode, err := c.Execute(ctx, "which "+binary)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		// Try common paths
		paths := []string{
			"/usr/bin/" + binary,
			"/usr/local/bin/" + binary,
			"/opt/" + binary + "/" + binary,
		}
		for _, p := range paths {
			if exists, _ := c.FileExists(ctx, p); exists {
				return p, nil
			}
		}
		return "", fmt.Errorf("%s not found on router", binary)
	}
	return strings.TrimSpace(stdout), nil
}

// RemoteAddr returns the remote address of the connection.
func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Close closes the SSH connection.
func (c *Client) Close() error {
	return c.conn.Close()
}
