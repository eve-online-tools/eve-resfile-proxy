package transform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/index"
)

func newCommandRunner(name string, cfg commandConfig) func(context.Context, Input) ([]byte, error) {
	return func(ctx context.Context, in Input) ([]byte, error) {
		return runCommand(ctx, name, cfg, in)
	}
}

func runCommand(ctx context.Context, ruleName string, cfg commandConfig, in Input) ([]byte, error) {
	if len(cfg.Args) == 0 {
		return nil, fmt.Errorf("command args are empty")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, cfg.Args[0], cfg.Args[1:]...)
	cmd.Stdin = bytes.NewReader(in.Data)
	cmd.Env = append(os.Environ(),
		"RES_PATH="+in.ResPath,
		"CDN_PATH="+in.CDNPath,
		"EVE_PLATFORM="+string(in.Platform),
		"RULE_NAME="+ruleName,
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg != "" {
			return nil, fmt.Errorf("command failed: %w: %s", err, msg)
		}
		return nil, fmt.Errorf("command failed: %w", err)
	}

	out := stdout.Bytes()
	if len(out) > cfg.MaxOutputBytes {
		return nil, fmt.Errorf("command output exceeds max_output_bytes (%d > %d)", len(out), cfg.MaxOutputBytes)
	}

	return out, nil
}

func platformCode(p index.Platform) int32 {
	switch p {
	case index.PlatformWindows:
		return 0
	case index.PlatformMacOS:
		return 1
	default:
		return -1
	}
}

func resolvePath(baseDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
