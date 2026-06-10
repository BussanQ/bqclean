//go:build windows

package winapi

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestEntryIsReparsePointWithJunction(t *testing.T) {
	temp := t.TempDir()
	target := filepath.Join(temp, "target")
	junction := filepath.Join(temp, "junction")
	regular := filepath.Join(temp, "regular.txt")

	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(regular, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("cmd", "/c", "mklink", "/J", junction, target).CombinedOutput(); err != nil {
		t.Skipf("mklink /J failed (filesystem may not support junctions): %v: %s", err, out)
	}

	entries, err := os.ReadDir(temp)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]fs.DirEntry{}
	for _, entry := range entries {
		byName[entry.Name()] = entry
	}

	junctionEntry := byName["junction"]
	if junctionEntry == nil {
		t.Fatal("junction entry not found")
	}
	if junctionEntry.Type()&(fs.ModeSymlink|fs.ModeIrregular) == 0 {
		t.Fatalf("junction dirent should carry symlink or irregular mode bits, got %v", junctionEntry.Type())
	}
	if reparse, err := EntryIsReparsePoint(junction, junctionEntry); err != nil || !reparse {
		t.Fatalf("junction should be detected as reparse point, got %v, %v", reparse, err)
	}

	for _, name := range []string{"target", "regular.txt"} {
		entry := byName[name]
		if entry == nil {
			t.Fatalf("%s entry not found", name)
		}
		path := filepath.Join(temp, name)
		if reparse, err := EntryIsReparsePoint(path, entry); err != nil || reparse {
			t.Fatalf("%s should not be a reparse point, got %v, %v", name, reparse, err)
		}
	}

	info, err := os.Lstat(junction)
	if err != nil {
		t.Fatal(err)
	}
	if reparse, err := InfoIsReparsePoint(junction, info); err != nil || !reparse {
		t.Fatalf("junction Lstat info should be detected as reparse point, got %v, %v", reparse, err)
	}
}
