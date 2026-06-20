package qemu

import (
	"fmt"
	"os"
	"path/filepath"
)

func (q *QEMU) DestroyVM(resourceId string) error {

	d := filepath.Join(q.DiskStore, resourceId)
	if err := os.RemoveAll(d); err != nil {
		return fmt.Errorf("error deleting the resource, %v", err)
	}
	return nil

}
