//go:build !windows

package winapi

func IsReparsePoint(path string) (bool, error) {
	return false, nil
}
