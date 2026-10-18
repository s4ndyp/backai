package smbmount

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type Config struct {
	Host     string
	Share    string
	Username string
	Password string
}

type Mount struct {
	LocalPath string
	unc       string
}

func MountTemp(cfg Config) (*Mount, error) {
	root := filepath.Join(os.TempDir(), "backai-smb-"+sanitize(cfg.Host+cfg.Share))
	if err := os.MkdirAll(root, 0o700); err != nil {
		return nil, err
	}
	m := &Mount{LocalPath: root}
	if err := m.mount(cfg); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *Mount) Unmount() error {
	switch runtime.GOOS {
	case "linux":
		cmd := exec.Command("umount", m.LocalPath)
		return cmd.Run()
	case "windows":
		cmd := exec.Command("net", "use", m.unc, "/delete", "/y")
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported os: %s", runtime.GOOS)
	}
}

func (m *Mount) mount(cfg Config) error {
	share := strings.TrimPrefix(cfg.Share, "/")
	share = strings.TrimPrefix(share, "\\")
	switch runtime.GOOS {
	case "linux":
		if err := exec.Command("which", "mount.cifs").Run(); err != nil {
			return fmt.Errorf("mount.cifs not found; install cifs-utils")
		}
		unc := "//" + cfg.Host + "/" + share
		opts := "username=" + cfg.Username + ",password=" + cfg.Password + ",vers=3.0"
		cmd := exec.Command("mount", "-t", "cifs", unc, m.LocalPath, "-o", opts)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("mount cifs: %w", err)
		}
		return nil
	case "windows":
		unc := `\\` + cfg.Host + `\` + strings.ReplaceAll(share, "/", `\`)
		m.unc = unc
		args := []string{"use", unc, cfg.Password, "/user:" + cfg.Username, "/persistent:no"}
		cmd := exec.Command("net", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("net use: %s: %w", string(out), err)
		}
		m.LocalPath = unc
		return nil
	default:
		return fmt.Errorf("unsupported os: %s", runtime.GOOS)
	}
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}
