package sshutil

import (
	"bytes"
	"fmt"
	"io"

	"github.com/pkg/sftp"
)

// SFTPClient wraps an SFTP connection for file transfer operations.
type SFTPClient struct {
	client *sftp.Client
}

// NewSFTPClient creates an SFTP client from an existing SSH client.
func NewSFTPClient(sshClient *Client) (*SFTPClient, error) {
	client, err := sftp.NewClient(sshClient.RawClient())
	if err != nil {
		return nil, fmt.Errorf("SFTP init: %w", err)
	}
	return &SFTPClient{client: client}, nil
}

// UploadBytes creates a remote file with the given content.
func (s *SFTPClient) UploadBytes(remotePath string, data []byte) error {
	f, err := s.client.Create(remotePath)
	if err != nil {
		return fmt.Errorf("SFTP create %s: %w", remotePath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("SFTP write %s: %w", remotePath, err)
	}
	return nil
}

// Close closes the SFTP connection.
func (s *SFTPClient) Close() error {
	return s.client.Close()
}
