package qemu

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func (q *QEMU) DestroyVM(ctx context.Context, resourceId string) error {

	d := filepath.Join(q.DiskStore, resourceId)
	if err := os.RemoveAll(d); err != nil {
		return fmt.Errorf("error deleting the resource, %v", err)
	}
	return nil

}
