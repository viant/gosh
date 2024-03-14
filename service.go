package gosh

import (
	"github.com/viant/gosh/runner"
	"strings"
)

type Service struct {
	runner runner.Runner
	osInfo *OSInfo
	hwInfo *HardwareInfo
}

func (s *Service) Run(command string, options ...runner.Option) (string, int, error) {
	return s.runner.Run(command, options...)
}

func (s *Service) PID() int {
	return s.runner.PID()
}

func (s *Service) OsInfo() *OSInfo {
	return s.osInfo
}

func (s *Service) HardwareInfo() *HardwareInfo {
	return s.hwInfo
}

func (s *Service) init() error {
	return s.detectSystem()
}

func (s *Service) detectSystem() (err error) {
	s.osInfo = &OSInfo{}
	s.hwInfo = &HardwareInfo{Architecture: "unknown"}
	if s.osInfo.System, _, err = s.runner.Run("uname -s"); err != nil {
		return err
	}
	s.osInfo.System = strings.ToLower(s.osInfo.System)
	if s.hwInfo.Hardware, _, err = s.runner.Run("uname -m"); err != nil {
		return err
	}
	s.hwInfo.Hardware = strings.ToLower(s.hwInfo.Hardware)
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
	output, _, err := s.runner.Run(checkCmd)
	if err != nil {
		return err
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
	return nil
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

func New(runner runner.Runner) (*Service, error) {
	ret := &Service{runner: runner}
	return ret, ret.init()
}
