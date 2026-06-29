//go:build windows

package doctor

import "fmt"

func worktreeFilesystemCapacity(path string) (filesystemCapacity, error) {
	return filesystemCapacity{}, fmt.Errorf("filesystem capacity probe unavailable on windows: %s", path)
}
