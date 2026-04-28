package opener

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func Command(target string) (*exec.Cmd, error) {
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("open target is required")
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", target), nil
	case "linux":
		return exec.Command("xdg-open", target), nil
	default:
		return nil, fmt.Errorf("opening files is not supported on %s", runtime.GOOS)
	}
}

func CustomCommand(command, target string) (*exec.Cmd, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return Command(target)
	}
	return exec.Command(parts[0], append(parts[1:], target)...), nil
}
