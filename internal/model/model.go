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

type Bridge struct{}

type Interface struct {
	Name string
	Type string
}

type ResourceIfaceInfo struct {
	IfaceName  string // "tap10"
	BridgeName string // "ovsbr0"
	BridgeType string // "ovs"
}

type User struct {
	Name       string   `yaml:"name"`
	Sudo       string   `yaml:"sudo"`
	LockPasswd bool     `yaml:"lock_passwd"`
	Shell      string   `yaml:"shell"`
	SshKey     []string `yaml:"ssh_authorized_keys"`
	Password   string
}

type LogError struct {
	ResourceId int    `json:"resource_id"`
	Timestamp  string `json:"timestamp"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	TaskId     string `json:"taskId"`
	Error      error  `json:"error"`
}

type LogSuccess struct {
	ResourceId int    `json:"resource_id"`
	Timestamp  string `json:"timestamp"`
	Action     string `json:"action"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	TaskId     string `json:"taskId"`
	Resource   string `json:"resource"`
}
