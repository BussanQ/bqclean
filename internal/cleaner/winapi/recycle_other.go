//go:build !windows

package winapi

import "errors"

func QueryRecycleBin(root string) (sizeBytes int64, itemCount int64, err error) {
	return 0, 0, errors.New("recycle bin is only supported on Windows")
}

func EmptyRecycleBin(root string) error {
	return errors.New("recycle bin is only supported on Windows")
}
