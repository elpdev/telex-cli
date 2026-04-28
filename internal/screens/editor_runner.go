package screens

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func runEditorCommand(editor, path, configErr string) error {
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return fmt.Errorf("%s", configErr)
	}
	cmd := exec.Command(parts[0], append(parts[1:], path)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	return waitForFileToSettle(path, 2*time.Second)
}

func waitForFileToSettle(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	var lastSize int64 = -1
	var lastMod time.Time
	stable := 0
	for {
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Size() == lastSize && info.ModTime().Equal(lastMod) {
			stable++
			if stable >= 2 {
				return nil
			}
		} else {
			stable = 0
			lastSize = info.Size()
			lastMod = info.ModTime()
		}
		if time.Now().After(deadline) {
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func readEditedFile(path string) ([]byte, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("%w; edited file kept at %s", err, path)
	}
	if err := os.Remove(path); err != nil {
		return nil, err
	}
	return content, nil
}
