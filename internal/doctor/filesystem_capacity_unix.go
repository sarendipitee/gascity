//go:build !windows

package doctor

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func worktreeFilesystemCapacity(path string) (filesystemCapacity, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(path, &stat); err != nil {
		return filesystemCapacity{}, fmt.Errorf("statfs %q: %w", path, err)
	}
	return filesystemCapacity{
		freeBytes:  int64(stat.Bavail * uint64(stat.Bsize)),
		totalBytes: int64(stat.Blocks * uint64(stat.Bsize)),
	}, nil
}
