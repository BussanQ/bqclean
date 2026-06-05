//go:build windows

package winapi

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	recycleNoConfirmation = 0x00000001
	recycleNoProgressUI   = 0x00000002
	recycleNoSound        = 0x00000004
)

var (
	shell32                = windows.NewLazySystemDLL("shell32.dll")
	procSHQueryRecycleBinW = shell32.NewProc("SHQueryRecycleBinW")
	procSHEmptyRecycleBinW = shell32.NewProc("SHEmptyRecycleBinW")
)

type recycleBinInfo struct {
	cbSize uint32
	size   int64
	items  int64
}

func QueryRecycleBin(root string) (sizeBytes int64, itemCount int64, err error) {
	ptr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0, err
	}
	info := recycleBinInfo{cbSize: uint32(unsafe.Sizeof(recycleBinInfo{}))}
	ret, _, _ := procSHQueryRecycleBinW.Call(
		uintptr(unsafe.Pointer(ptr)),
		uintptr(unsafe.Pointer(&info)),
	)
	if ret != 0 {
		return 0, 0, fmt.Errorf("SHQueryRecycleBinW failed: 0x%x", ret)
	}
	return info.size, info.items, nil
}

func EmptyRecycleBin(root string) error {
	ptr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return err
	}
	ret, _, _ := procSHEmptyRecycleBinW.Call(
		0,
		uintptr(unsafe.Pointer(ptr)),
		recycleNoConfirmation|recycleNoProgressUI|recycleNoSound,
	)
	if ret != 0 {
		return fmt.Errorf("SHEmptyRecycleBinW failed: 0x%x", ret)
	}
	return nil
}
