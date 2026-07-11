package vfs_test

import (
	"crypto/md5"
	"io/fs"
	"testing"

	"github.com/eve-online-tools/eve-resfile-proxy/vfs"
	md5pkg "github.com/eve-online-tools/eve-resfile-proxy/vfs/internal/md5"
)

const testMD5Hex = "faa842f6f3157c8d6cde37b496a43b30"

func TestCompressedSizeInfo_fromStat(t *testing.T) {
	manifest := "res:/icons/64/icon64.png,7d/icon64_hash," + testMD5Hex + ",1024,512\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	info, err := fs.Stat(fsys, "icons/64/icon64.png")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	li, ok := info.(vfs.CompressedSizeInfo)
	if !ok {
		t.Fatalf("Stat did not return CompressedSizeInfo, got %T", info)
	}
	if info.Size() != 1024 {
		t.Fatalf("Size() = %d, want 1024", info.Size())
	}
	if li.CompressedSize() != 512 {
		t.Fatalf("CompressedSize() = %d, want 512", li.CompressedSize())
	}
}

func TestManifestFileInfo_MD5_fromStat(t *testing.T) {
	manifest := "res:/icons/64/icon64.png,7d/icon64_hash," + testMD5Hex + ",1024,512\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	info, err := fs.Stat(fsys, "icons/64/icon64.png")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	mfi, ok := info.(vfs.ManifestFileInfo)
	if !ok {
		t.Fatalf("Stat did not return ManifestFileInfo, got %T", info)
	}

	want, err := md5pkg.Parse(testMD5Hex)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if mfi.MD5() != want.Sum() {
		t.Fatalf("MD5() = %x, want %x", mfi.MD5(), want.Sum())
	}
}

func TestManifestFileInfo_MD5_equals_md5Sum(t *testing.T) {
	data := []byte("eve resfile content")
	sum := md5.Sum(data)
	hexDigest := md5pkg.Digest(sum).String()

	manifest := "res:/a.png,aa/a_hash," + hexDigest + "\n"
	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	info, err := fs.Stat(fsys, "a.png")
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	mfi := info.(vfs.ManifestFileInfo)
	if mfi.MD5() != sum {
		t.Fatalf("MD5() = %x, want %x", mfi.MD5(), sum)
	}
}

func TestManifestFileInfo_MD5_fromReadDir(t *testing.T) {
	const dirMD5 = "abcdefabcdefabcdefabcdefabcdefab"
	manifest := "res:/a.png,aa/a_hash," + dirMD5 + "\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	info, err := entries[0].Info()
	if err != nil {
		t.Fatalf("Info: %v", err)
	}

	mfi, ok := info.(vfs.ManifestFileInfo)
	if !ok {
		t.Fatalf("Info did not return ManifestFileInfo, got %T", info)
	}
	want, err := md5pkg.Parse(dirMD5)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if mfi.MD5() != want.Sum() {
		t.Fatalf("MD5() = %x, want %x", mfi.MD5(), want.Sum())
	}
}

func TestManifestFileInfo_notOnDirectories(t *testing.T) {
	manifest := "res:/dir/file.png,aa/file," + testMD5Hex + "\n"

	fsys, err := vfs.New([]byte(manifest), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err != nil {
		t.Fatalf("NewFS: %v", err)
	}

	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("Info: %v", err)
		}
		if _, ok := info.(vfs.ManifestFileInfo); ok {
			t.Fatalf("directory %q should not implement ManifestFileInfo", entry.Name())
		}
	}
}

func TestNewFS_invalidMD5Checksum(t *testing.T) {
	_, err := vfs.New([]byte("res:/a.png,aa/a,not-a-valid-md5\n"), nil, vfs.WithPrefix(vfs.PrefixRes))
	if err == nil {
		t.Fatal("expected error")
	}
}
