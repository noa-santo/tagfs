package logic

import "syscall"

func ToStoredMode(mode uint32, isDir bool) int64 {
	typeBits := syscall.S_IFREG
	if isDir {
		typeBits = syscall.S_IFDIR
	}
	return int64(typeBits | int(mode&0777))
}
