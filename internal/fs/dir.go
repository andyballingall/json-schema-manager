package fs

import (
	"os"
	"slices"
	"strconv"
)

// GetUintSubdirectories returns a slice of uint64s corresponding to subdirectories
// of the given path that are compatible with uint64, sorted in ascending order.
func GetUintSubdirectories(dirPath string) ([]uint64, error) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	var nums []uint64
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if val, pErr := strconv.ParseUint(entry.Name(), 10, 64); pErr == nil {
			nums = append(nums, val)
		}
	}
	slices.Sort(nums)
	return nums, nil
}
