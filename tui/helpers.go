package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func isRepoCloned(cloneDir, name string) bool {
	dest := filepath.Join(cloneDir, name)
	info, err := os.Stat(dest)
	return err == nil && info.IsDir()
}

func cloneRepoCmd(url, cloneDir, name string, shallow bool) (string, error) {
	dest := filepath.Join(cloneDir, name)

	if _, err := os.Stat(dest); err == nil {
		return dest, fmt.Errorf("already cloned at %s", dest)
	}

	if err := os.MkdirAll(cloneDir, 0o755); err != nil {
		return "", fmt.Errorf("could not create directory: %w", err)
	}

	args := []string{"clone"}
	if shallow {
		args = append(args, "--depth", "1")
	}
	args = append(args, url, dest)

	cmd := exec.Command("git", args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone failed: %s", string(out))
	}
	return dest, nil
}

func openEditorCmd(cloneDir, name string) (string, error) {
	dest := filepath.Join(cloneDir, name)

	if _, err := exec.LookPath("zed"); err == nil {
		cmd := exec.Command("zed", dest)
		if err := cmd.Start(); err != nil {
			return "", fmt.Errorf("could not open zed: %w", err)
		}
		return dest, nil
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		return dest, nil
	}

	cmd := exec.Command(editor, dest)
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("could not open %s: %w", editor, err)
	}
	return dest, nil
}
