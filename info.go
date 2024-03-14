package gosh

type (
	OSInfo struct {
		System        string // Operating system type
		Name          string // Name of the OS
		DistributorID string // ID of the distributor (from lsb_release -a)
		Description   string // Full description of the OS (from lsb_release -a)
		Release       string // OS release number (from lsb_release -a)
		Codename      string // OS release codename (from lsb_release -a)
	}

	HardwareInfo struct {
		Hardware     string // Hardware details
		Architecture string // Full architecture name
		Arch         string // Architecture abbreviation, typically from uname -m
		Version      string // Hardware version
	}
)
