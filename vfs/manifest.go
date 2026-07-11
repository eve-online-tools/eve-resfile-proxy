package vfs

import (
	"errors"
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs/internal/md5"
)

var (
	errEmptyLine                  = errors.New("empty line")
	errNotEnoughColumns           = errors.New("not enough columns, expected at least 2")
	errFilenameAndCDNPathRequired = errors.New("filename and cdn filename are required")
	errInvalidSize                = errors.New("invalid size")
	errInvalidCompressedSize      = errors.New("invalid compressed size")
)

type Prefix string

const (
	PrefixApp Prefix = "app"
	PrefixRes Prefix = "res"
)

// Entry is one row from an EVE Resource File manifest.
type Entry struct {
	LogicalPath    string
	CDNPath        string
	MD5            md5.Digest
	Size           int64
	CompressedSize int64
	Mode           string
}

func (e *Entry) GetCDNPath() string {
	return e.CDNPath
}

func (e *Entry) Checksum() string {
	return e.MD5.String()
}

type options struct {
	prefix   Prefix
	validate bool
}

// Option configures NewFS.
type Option func(*options)

// WithPrefix sets the manifest namespace.
func WithPrefix(prefix Prefix) Option {
	return func(o *options) {
		o.prefix = prefix
	}
}

// WithValidate enables MD5 checksum verification when opening files.
func WithValidate() Option {
	return func(o *options) {
		o.validate = true
	}
}

// New builds a read-only fs.FS from resfile manifest bytes.
func New(manifest []byte, fetcher Fetcher, opts ...Option) (fs.FS, error) {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	if o.prefix == "" {
		return nil, fmt.Errorf("prefix is required")
	}
	if o.prefix != PrefixApp && o.prefix != PrefixRes {
		return nil, fmt.Errorf("invalid prefix %q (expected %q or %q)", o.prefix, PrefixApp, PrefixRes)
	}

	entries, err := parseManifest(string(manifest), o.prefix)
	if err != nil {
		return nil, err
	}

	return &manifestFS{
		entries:  entries,
		fetcher:  fetcher,
		prefix:   o.prefix,
		validate: o.validate,
	}, nil
}

func parseManifest(content string, prefix Prefix) (map[string]Entry, error) {
	entries := make(map[string]Entry)
	logicalPrefix := string(prefix) + ":/"

	for _, line := range strings.Split(content, "\n") {
		entry, err := parseIndexLine(line)
		if err != nil {
			if errors.Is(err, errEmptyLine) || errors.Is(err, errNotEnoughColumns) {
				continue
			}
			return nil, errors.Join(err, fmt.Errorf("parse manifest line %q", line))
		}

		fsPath, ok := strings.CutPrefix(entry.LogicalPath, logicalPrefix)
		if !ok {
			continue
		}
		if fsPath == "" {
			continue
		}

		if _, exists := entries[fsPath]; exists {
			continue
		}
		entries[fsPath] = entry
	}

	return entries, nil
}

func parseIndexLine(line string) (Entry, error) {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return Entry{}, errEmptyLine
	}

	parts := strings.Split(trimmed, ",")
	if len(parts) < 2 {
		return Entry{}, errNotEnoughColumns
	}

	logicalPath := strings.TrimSpace(parts[0])
	cdnPath := strings.TrimSpace(parts[1])
	if logicalPath == "" || cdnPath == "" {
		return Entry{}, errFilenameAndCDNPathRequired
	}

	entry := Entry{
		LogicalPath: strings.ToLower(logicalPath),
		CDNPath:     cdnPath,
	}

	if len(parts) > 2 {
		md5sum, err := md5.Parse(strings.TrimSpace(parts[2]))
		if err != nil {
			return Entry{}, err
		}
		entry.MD5 = md5sum
	}
	if len(parts) > 3 && strings.TrimSpace(parts[3]) != "" {
		size, err := strconv.ParseInt(strings.TrimSpace(parts[3]), 10, 64)
		if err != nil {
			return Entry{}, errInvalidSize
		}
		entry.Size = size
	}
	if len(parts) > 4 && strings.TrimSpace(parts[4]) != "" {
		compressedSize, err := strconv.ParseInt(strings.TrimSpace(parts[4]), 10, 64)
		if err != nil {
			return Entry{}, errInvalidCompressedSize
		}
		entry.CompressedSize = compressedSize
	}
	if len(parts) > 5 {
		entry.Mode = strings.TrimSpace(parts[5])
	}

	return entry, nil
}
