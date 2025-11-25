package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Adapter struct{}

func NewAdapter() *Adapter {
	return &Adapter{}
}

// DeleteDirectory deletes a directory and all its contents
func (a *Adapter) DeleteDirectory(ctx context.Context, path string) error {
	// Safety check: ensure path is absolute and not a dangerous directory
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// Prevent deletion of system directories
	dangerousPaths := []string{"/", "/etc", "/var", "/usr", "/bin", "/sbin", "/home", "/root"}
	for _, dangerous := range dangerousPaths {
		if path == dangerous || strings.HasPrefix(path, dangerous+"/") {
			// Allow if it's within /var/backups or similar backup directories
			if !strings.HasPrefix(path, "/var/backups/") && !strings.Contains(path, "backup") {
				return fmt.Errorf("refusing to delete system directory: %s", path)
			}
		}
	}

	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}

	return nil
}


// CreateDirectory creates a directory with the given permissions
func (a *Adapter) CreateDirectory(path string, perm uint32) error {
	if err := os.MkdirAll(path, os.FileMode(perm)); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}
	return nil
}
