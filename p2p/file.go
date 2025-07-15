package p2p

import (
	"errors"
	"os"
	"syscall"

	"github.com/edsrzf/mmap-go"
)

// linux specific syscall to allocate the size of a file
// TODO: implement version for other operating systems
func fallocate(file *os.File, offset int64, length int64) error {
	if length == 0 {
		return nil
	}
	return syscall.Fallocate(int(file.Fd()), 0, offset, length)
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func OpenFile(path string, size int64) (mmap.MMap, error) {
	exists, err := fileExists(path)
	if err != nil {
		return nil, err
	}

	// TODO: should be able to create the file with
	// the same file permissions as the sender -- how to do in os-agnostic way?
	// the permission should be applied after all the file contents have been recived
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if !exists {
		err = fallocate(file, 0, size)
		if err != nil {
			return nil, err
		}
	}

	fileData, err := mmap.Map(file, mmap.RDWR, 0)
	return fileData, err
}
