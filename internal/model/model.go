package model

type VM struct {
	ResourceID    int    `json:"resource_id"`
	PID           int    `json:"pid"`
	CPUs          int    `json:"cpu"`
	MemoryMB      int    `json:"memory_mb"`
	VNC           int    `json:"vnc"`
	DiskSize      int    `json:"disk_size"`
	DiskIndex     int    `json:"disk_index"`
	Name          string `json:"resource_name"`
	BootDisk      string `json:"disk_path"`
	CloudInitFile string `json:"cloud_init"`
	Image         string `json:"os_image"`
	Status        string `json:"status"`
	Version       string `json:"version"`
	DiskType      string `json:"disk_type"`
}

type VmListing struct {
	PID    int
	Name   string
	Image  string
	Status string
	UpTime string
}

type User struct {
	Name       string   `yaml:"name"`
	Sudo       string   `yaml:"sudo"`
	LockPasswd bool     `yaml:"lock_passwd"`
	Shell      string   `yaml:"shell"`
	SshKey     []string `yaml:"ssh_authorized_keys"`
	Password   string
}
