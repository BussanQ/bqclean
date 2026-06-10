//go:build windows

package winapi

import "golang.org/x/sys/windows"

// FixedDriveRoots returns the roots of all fixed local drives, e.g. ["C:\\", "D:\\"].
func FixedDriveRoots() ([]string, error) {
	size, err := windows.GetLogicalDriveStrings(0, nil)
	if err != nil {
		return nil, err
	}
	buf := make([]uint16, size)
	if _, err := windows.GetLogicalDriveStrings(size, &buf[0]); err != nil {
		return nil, err
	}

	roots := make([]string, 0, 4)
	start := 0
	for i, ch := range buf {
		if ch != 0 {
			continue
		}
		if i > start {
			root := windows.UTF16ToString(buf[start:i])
			ptr, err := windows.UTF16PtrFromString(root)
			if err == nil && windows.GetDriveType(ptr) == windows.DRIVE_FIXED {
				roots = append(roots, root)
			}
		}
		start = i + 1
	}
	return roots, nil
}
