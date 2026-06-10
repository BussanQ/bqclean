//go:build !windows

package winapi

func FixedDriveRoots() ([]string, error) {
	return nil, nil
}
