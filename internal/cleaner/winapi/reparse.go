package winapi

import (
	"io/fs"
	"os"
)

// reparseModeBits are the mode bits Go sets for Windows reparse points:
// symlinks map to ModeSymlink, junctions and other tags to ModeIrregular.
// Since Go 1.23 these bits come straight from the directory entry, so when
// neither bit is set the path is definitely not a reparse point and the
// extra GetFileAttributes syscall can be skipped.
const reparseModeBits = fs.ModeSymlink | fs.ModeIrregular

// EntryIsReparsePoint reports whether path is a reparse point, using the
// directory entry's type bits as a syscall-free fast path.
func EntryIsReparsePoint(path string, entry fs.DirEntry) (bool, error) {
	if entry == nil {
		return IsReparsePoint(path)
	}
	if entry.Type()&reparseModeBits == 0 {
		return false, nil
	}
	return IsReparsePoint(path)
}

// InfoIsReparsePoint reports whether path is a reparse point, using the
// Lstat result's mode bits as a syscall-free fast path.
func InfoIsReparsePoint(path string, info os.FileInfo) (bool, error) {
	if info == nil {
		return IsReparsePoint(path)
	}
	if info.Mode()&reparseModeBits == 0 {
		return false, nil
	}
	return IsReparsePoint(path)
}
