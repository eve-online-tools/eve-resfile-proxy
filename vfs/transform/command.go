package transform

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func newCommandRunner(name string, cfg compiledCommand) func(context.Context, assetInput) ([]byte, error) {
	return func(ctx context.Context, in assetInput) ([]byte, error) {
		return runCommand(ctx, name, cfg, in)
	}
}

func runCommand(ctx context.Context, ruleName string, cfg compiledCommand, in assetInput) ([]byte, error) {
	if len(cfg.args) == 0 {
		return nil, fmt.Errorf("command args are empty")
	}

	cmdCtx, cancel := context.WithTimeout(ctx, cfg.timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, cfg.args[0], cfg.args[1:]...)
	cmd.Stdin = bytes.NewReader(in.Data)
	cmd.Env = append(os.Environ(),
		"RES_PATH="+in.ResPath,
		"CDN_PATH="+in.CDNPath,
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
	if len(out) > cfg.maxOutputBytes {
		return nil, fmt.Errorf("command output exceeds max_output_bytes (%d > %d)", len(out), cfg.maxOutputBytes)
	}

	return out, nil
}

func resolvePath(baseDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(baseDir, path)
}
