package util

import (
	"syscall"
)

// GetDiskUsage 获取指定路径所在磁盘的已用空间和剩余空间
// 返回: (已用字节数, 剩余字节数, 错误)
func GetDiskUsage(path string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	// 可用块数 * 每块大小 = 剩余空间(字节)
	free := stat.Bavail * uint64(stat.Bsize)
	// 总块数 * 每块大小 = 总空间(字节)
	total := stat.Blocks * uint64(stat.Bsize)
	used := total - free

	return used, free, nil
}
