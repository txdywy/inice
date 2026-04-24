package sshutil

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
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

// Execute runs a command on the remote host and returns stdout/stderr.
func (c *Client) Execute(cmd string) (string, string, error) {
	return c.ExecuteWithContext(context.Background(), cmd)
}

// ExecuteWithContext runs a command with context cancellation support.
func (c *Client) ExecuteWithContext(ctx context.Context, cmd string) (string, string, error) {
	session, err := c.conn.NewSession()
	if err != nil {
		return "", "", fmt.Errorf("new session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Handle context cancellation
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

// RemoteAddr returns the remote address of the connection.
func (c *Client) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Close closes the SSH connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// RawClient returns the underlying ssh.Client for use with SFTP.
func (c *Client) RawClient() *ssh.Client {
	return c.conn
}
