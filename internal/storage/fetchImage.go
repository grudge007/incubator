package storage

import (
	"fmt"
)

func (d *DB) FindImage(imageName, version string) (string, error) {
	var path string
	query := "SELECT path FROM images WHERE image_name = ? AND VERSION = ? LIMIT 1"
	err := d.Cli.QueryRow(query, imageName, version).Scan(&path)
	if err != nil {
		return "", fmt.Errorf("Failed to fetch image, %v", err)
	}

	return path, nil

}
