package storage

import (
	"database/sql"
	"fmt"
	"incubator/internal/model"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	Cli *sql.DB
}

func InitDB() *DB {
	db, err := sql.Open("sqlite3", "file:/root/incubator.db?_foreign_keys=on")
	if err != nil {
		log.Fatal("Error: DB Connection Failed")
	}

	db.SetMaxOpenConns(1)

	return &DB{
		Cli: db,
	}
}

func (d *DB) InsertInitialVmMeta(vmResp *model.VM) (int64, error) {
	// 1. Prepare the complete SQL query statement
	query := `
		INSERT INTO metadata (
			resource_name,resource_id, disk_path, cloud_init, os_image, 
			cpu, memory, disk_size, disk_type, disk_index, 
			vnc_port, created_at, updated_at
		) VALUES (?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`

	// 2. Execute the query and pass the values from the vmResp struct mapping to the placeholders
	result, err := d.Cli.Exec(
		query,
		vmResp.Name,          // resource_name
		vmResp.ResourceID,    // resource_id
		vmResp.BootDisk,      // disk_path
		vmResp.CloudInitFile, // cloud_init_path
		vmResp.Image,         // os_image
		vmResp.CPUs,          // cpu
		vmResp.MemoryMB,      // memory
		vmResp.DiskSize,      // disk_size
		vmResp.DiskType,      // disk_type (Added since it's NOT NULL in your schema)
		vmResp.DiskIndex,     // disk_index
		vmResp.VNC,           // vnc_port
	)

	if err != nil {
		return 0, fmt.Errorf("failed to insert VM metadata: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get inserted row id: %w", err)
	}

	return id, nil
}

func (d *DB) InsertVmMeta(vmResp *model.VM) error {
	// 1. Prepare the complete SQL query statement
	query := `
		INSERT INTO metadata (
			resource_name,resource_id, status, disk_path, cloud_init, os_image, 
			process_id, cpu, memory, disk_size, disk_type, disk_index, 
			vnc_port, created_at, updated_at
		) VALUES (?, ?,?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);
	`

	// 2. Execute the query and pass the values from the vmResp struct mapping to the placeholders
	_, err := d.Cli.Exec(
		query,
		vmResp.Name,          // resource_name
		vmResp.ResourceID,    // resource_id
		vmResp.Status,        // status
		vmResp.BootDisk,      // disk_path
		vmResp.CloudInitFile, // cloud_init_path
		vmResp.Image,         // os_image
		vmResp.PID,           // process_id
		vmResp.CPUs,          // cpu
		vmResp.MemoryMB,      // memory
		vmResp.DiskSize,      // disk_size
		vmResp.DiskType,      // disk_type (Added since it's NOT NULL in your schema)
		vmResp.DiskIndex,     // disk_index
		vmResp.VNC,           // vnc_port
	)

	if err != nil {
		return fmt.Errorf("failed to insert VM metadata: %w", err)
	}

	return nil
}

func (d *DB) AllocateResourceID() (int, error) {
	const baseID = 10001

	query := `
		SELECT CASE 
			-- Case 1: If the base ID is not in the table, it's the smallest available number.
			WHEN NOT EXISTS (SELECT 1 FROM metadata WHERE resource_id = ?) THEN ?
			
			-- Case 2: If the base ID exists, find the first gap after any existing number.
			ELSE (
				SELECT MIN(v1.resource_id + 1)
				FROM metadata v1
				LEFT JOIN metadata v2 ON v1.resource_id + 1 = v2.resource_id
				WHERE v2.resource_id IS NULL AND v1.resource_id >= ?
			)
		END;
	`

	var resourceID int
	// Pass baseID three times to satisfy all the '?' placeholders
	err := d.Cli.QueryRow(query, baseID, baseID, baseID).Scan(&resourceID)
	if err != nil {
		return 0, fmt.Errorf("failed to allocate smallest available resource ID: %w", err)
	}

	fmt.Printf("\nVMID: %d\n", resourceID)
	return resourceID, nil
}

func (d *DB) AllocateVNCPort() (int, error) {
	query := `
		SELECT CASE 
			-- Case 1: If 1 is not in the table, it's the smallest available number.
			WHEN NOT EXISTS (SELECT 1 FROM metadata WHERE vnc_port = 1) THEN 1
			
			-- Case 2: If 1 exists, find the first gap after any existing number.
			ELSE (
				SELECT MIN(v1.vnc_port + 1)
				FROM metadata v1
				LEFT JOIN metadata v2 ON v1.vnc_port + 1 = v2.vnc_port
				WHERE v2.vnc_port IS NULL
			)
		END;
	`

	var display int
	err := d.Cli.QueryRow(query).Scan(&display)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch smallest available VNC display: %w", err)
	}

	fmt.Printf("\nVNC FROM ORCA: %d\n", display)
	return display, nil
}

func (d *DB) ResourceStatus(resourceId string) (string, error) {
	query := `SELECT status FROM metadata WHERE resource_id = ?`
	var status string

	err := d.Cli.QueryRow(query, resourceId).Scan(&status)
	if err != nil {
		return "", fmt.Errorf("failed to fetch vm status")
	}
	return status, nil
}

func (d *DB) DeleteResource(tableName, columnName, resourceId string) error {
	query := fmt.Sprintf("DELETE FROM %s WHERE %s = ?", tableName, columnName)

	if _, err := d.Cli.Exec(query, resourceId); err != nil {
		return fmt.Errorf("failed to delete vm, %v", err)
	}
	return nil
}

func (d *DB) FetchPid(resourceId string) (int, error) {
	var pid int
	query := `SELECT process_id FROM metadata WHERE resource_id = ?`

	if err := d.Cli.QueryRow(query, resourceId).Scan(&pid); err != nil {
		return 0, fmt.Errorf("failed to fetch pid, %v", err)
	}
	return pid, nil
}

func (d *DB) UpdateResourceStatusAndPid(resourceId, status string, pid int) error {
	query := "UPDATE metadata SET status = ?, process_id = ? WHERE resource_id = ?"

	_, err := d.Cli.Exec(query, status, pid, resourceId)
	if err != nil {
		return fmt.Errorf("failed to change resource status and pid %v", err)
	}
	return nil
}

func (d *DB) FetchVmdetails(resourceId string) (model.VM, error) {
	query := "SELECT memory, resource_name, cpu, disk_path, cloud_init, vnc_port, resource_id FROM metadata WHERE resource_id = ?"
	var vmDetails model.VM

	err := d.Cli.QueryRow(query, resourceId).Scan(
		&vmDetails.MemoryMB,
		&vmDetails.Name,
		&vmDetails.CPUs,
		&vmDetails.BootDisk,
		&vmDetails.CloudInitFile,
		&vmDetails.VNC,
		&vmDetails.ResourceID,
	)
	if err != nil {
		return vmDetails, fmt.Errorf("failed to fetch details")
	}
	// vmDetails.ResourceID == resourceId
	return vmDetails, nil
}

func (d *DB) UpdateResourcePid(resourceId string, pid int) error {
	query := fmt.Sprintf("UPDATE metadata SET status = '%d' WHERE process_id = ?", pid)

	_, err := d.Cli.Exec(query, resourceId)
	if err != nil {
		return fmt.Errorf("failed to change resource status, %v", err)
	}
	return nil
}

func (d *DB) ListAllResource() ([]model.VM, error) {
	query := "SELECT memory, resource_name, cpu, vnc_port, os_image, process_id, resource_id, status, disk_size FROM metadata"
	var vms []model.VM

	rows, err := d.Cli.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch details")
	}
	defer rows.Close()
	for rows.Next() {
		var vmDetails model.VM
		err := rows.Scan(
			&vmDetails.MemoryMB,
			&vmDetails.Name,
			&vmDetails.CPUs,
			&vmDetails.VNC,
			&vmDetails.Image,
			&vmDetails.PID,
			&vmDetails.ResourceID,
			&vmDetails.Status,
			&vmDetails.DiskSize,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		vms = append(vms, vmDetails)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return vms, nil
}

func (d *DB) ListSingleResource(resourceId string) ([]model.VM, error) {
	var vms []model.VM
	var vmDetails model.VM

	query := "SELECT memory, resource_name, cpu, vnc_port, os_image, process_id, resource_id, status, disk_size FROM metadata WHERE resource_id = ?"
	err := d.Cli.QueryRow(query, resourceId).Scan(
		&vmDetails.MemoryMB,
		&vmDetails.Name,
		&vmDetails.CPUs,
		&vmDetails.VNC,
		&vmDetails.Image,
		&vmDetails.PID,
		&vmDetails.ResourceID,
		&vmDetails.Status,
		&vmDetails.DiskSize,
	)
	vms = append(vms, vmDetails)
	if err != nil {
		return vms, fmt.Errorf("failed to scan row: %w", err)
	}
	return vms, nil
}

func (d *DB) InserNetworkIface(resourceId, bridgeId int, iface string) error {
	query := `
		INSERT INTO networks (
		resource_id,
		bridge_id,
		tap_name
		)
		VALUES (?, ?, ?)
	`
	_, err := d.Cli.Exec(query,
		resourceId,
		bridgeId,
		iface,
	)
	if err != nil {
		return fmt.Errorf("failed to insert network interface: %w", err)
	}
	return nil
}

func (d *DB) FetchInterfaces(resourceId int) ([]string, error) {
	query := "SELECT tap_name  FROM networks WHERE resource_id = ?"
	results, err := d.Cli.Query(query, resourceId)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	defer results.Close()

	var ifaces []string
	for results.Next() {
		var id string
		err := results.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		ifaces = append(ifaces, id)
	}
	return ifaces, nil

}

func (d *DB) FetchBridgeId(bridgeName string) (int, error) {
	var bridgeId int
	query := "SELECT id FROM network_bridges WHERE name = ?"

	err := d.Cli.QueryRow(query, bridgeName).Scan(&bridgeId)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch bridge id: %w", err)
	}

	return bridgeId, nil

}
