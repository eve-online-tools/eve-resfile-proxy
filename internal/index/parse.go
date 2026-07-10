package index

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/internal/lookup"
)

func ParseIndexLine(line string) (Entry, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Entry{}, fmt.Errorf("empty line")
	}

	parts := strings.Split(trimmed, ",")
	if len(parts) < 2 {
		return Entry{}, fmt.Errorf("expected at least 2 columns")
	}

	logicalPath := strings.TrimSpace(parts[0])
	cdnPath := strings.TrimSpace(parts[1])
	if logicalPath == "" || cdnPath == "" {
		return Entry{}, fmt.Errorf("filename and cdn filename are required")
	}

	entry := Entry{
		LogicalPath: logicalPath,
		CDNPath:     cdnPath,
	}

	if len(parts) > 2 {
		entry.Checksum = strings.TrimSpace(parts[2])
	}
	if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
		size, err := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		if err != nil {
			return Entry{}, fmt.Errorf("invalid size %q: %w", parts[3], err)
		}
		entry.Size = size
	}
	if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
		compressedSize, err := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		if err != nil {
			return Entry{}, fmt.Errorf("invalid compressed_size %q: %w", parts[4], err)
		}
		entry.CompressedSize = compressedSize
	}
	if len(parts) > 5 {
		entry.Mode = strings.TrimSpace(parts[5])
	}

	return entry, nil
}

func ParseBuildIndex(content string) (map[string]Entry, error) {
	entries := make(map[string]Entry)

	for _, line := range strings.Split(content, "\n") {
		entry, err := ParseIndexLine(line)
		if err != nil {
			if strings.Contains(err.Error(), "empty line") || strings.Contains(err.Error(), "expected at least") {
				continue
			}
			return nil, fmt.Errorf("parse build index line %q: %w", line, err)
		}
		if !strings.HasPrefix(entry.LogicalPath, "app:/") {
			continue
		}
		entries[entry.LogicalPath] = entry
	}

	return entries, nil
}

func ParseResfileIndex(content string) (map[string]Entry, error) {
	entries := make(map[string]Entry)

	for _, line := range strings.Split(content, "\n") {
		entry, err := ParseIndexLine(line)
		if err != nil {
			if strings.Contains(err.Error(), "empty line") || strings.Contains(err.Error(), "expected at least") {
				continue
			}
			return nil, fmt.Errorf("parse resfile index line %q: %w", line, err)
		}
		if !strings.HasPrefix(entry.LogicalPath, "res:/") {
			continue
		}
		key := lookup.ResPathKey(entry.LogicalPath)
		entry.LogicalPath = key
		entries[key] = entry
	}

	return entries, nil
}

func FindBuildIndexEntry(entries map[string]Entry, logicalPath string) (Entry, bool) {
	entry, ok := entries[logicalPath]
	return entry, ok
}
