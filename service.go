package gosh

import (
	"context"
	"github.com/viant/gosh/runner"
	"strings"
)

// Service represents a shell service
type Service struct {
	runner runner.Runner
	osInfo *OSInfo
	hwInfo *HardwareInfo
	user   string
}

func (s *Service) User() string {
	return s.user
}

func (s *Service) Close() error {
	return s.runner.Close()
}

// Run runs supplied command
func (s *Service) Run(ctx context.Context, command string, options ...runner.Option) (string, int, error) {
	return s.runner.Run(ctx, command, options...)
}

// PID returns process id
func (s *Service) PID() int {
	return s.runner.PID()
}

// OsInfo represents OS information
func (s *Service) OsInfo() *OSInfo {
	return s.osInfo
}

// HardwareInfo represents hardware information
func (s *Service) HardwareInfo() *HardwareInfo {
	return s.hwInfo
}

func (s *Service) init(ctx context.Context) error {
	return s.detectSystem(ctx)
}

func (s *Service) detectSystem(ctx context.Context) (err error) {
	s.osInfo = &OSInfo{}
	s.hwInfo = &HardwareInfo{Architecture: "unknown"}
	var e error
	if s.osInfo.System, _, e = s.runner.Run(ctx, "uname -s"); err != nil {
		err = e
	}
	s.osInfo.System = strings.TrimSpace(strings.TrimSpace(strings.ToLower(s.osInfo.System)))
	if s.hwInfo.Hardware, _, e = s.runner.Run(ctx, "uname -m"); err != nil {
		err = e
	}
	s.hwInfo.Hardware = strings.TrimSpace(strings.ToLower(s.hwInfo.Hardware))
	checkCmd := "lsb_release -a"
	if s.osInfo.System == "darwin" {
		checkCmd = "sw_vers"
	}
	if isAmd64Architecture(s.hwInfo.Hardware) {
		s.hwInfo.Architecture = "amd64"
		s.hwInfo.Arch = "x64"
	}
	if isArm64Architecture(s.hwInfo.Hardware) {
		s.hwInfo.Architecture = "arm64"
		s.hwInfo.Arch = "aarch64"
	}
	if isAppleArm64Architecture(s.hwInfo.Hardware) {
		s.hwInfo.Architecture = "arm64"
		s.hwInfo.Arch = "x64"
	}
	output, _, e := s.runner.Run(ctx, checkCmd)
	if e != nil {
		err = e
	}
	lines := strings.Split(output, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		pair := strings.Split(line, ":")
		if len(pair) != 2 {
			continue
		}
		var key = strings.Replace(strings.ToLower(pair[0]), " ", "", len(pair[0]))
		var val = strings.Replace(strings.Trim(pair[1], " \t\r"), " ", "", len(line))
		switch key {
		case "distributorid":
			s.osInfo.DistributorID = strings.ToLower(val)
		case "productname":
			s.osInfo.Name = strings.ToLower(val)
		case "buildversion":
			s.osInfo.Codename = strings.ToLower(val)
		case "productversion", "release":
			s.osInfo.Release = strings.ToLower(val)
		}

	}
	if s.user, _, e = s.runner.Run(ctx, "echo $USER"); err != nil {
		err = e
	}
	s.user = strings.TrimSpace(s.user)
	if isNotFound(err) {
		return nil
	}
	return err
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(strings.ToLower(err.Error()), "not found")
}

func isAmd64Architecture(candidate string) bool {
	return strings.Contains(candidate, "amd64") || strings.Contains(candidate, "x86_64")
}
func isArm64Architecture(hardware string) bool {
	return strings.Contains(hardware, "aarch64") || strings.Contains(hardware, "arm64")
}

func isAppleArm64Architecture(hardware string) bool {
	return strings.Contains(hardware, "arm64")
}

// New creates a new shell service
func New(ctx context.Context, runner runner.Runner) (*Service, error) {
	ret := &Service{runner: runner}
	return ret, ret.init(ctx)
}
