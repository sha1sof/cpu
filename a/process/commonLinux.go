//go:build linux
// +build linux

package process

import (
	"context"
	"cpuV3/a/cpu"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func Virtualization() (string, string, error) {
	return VirtualizationWithContext(context.Background())
}

var (
	cachedVirtMap   map[string]string
	cachedVirtMutex sync.RWMutex
	cachedVirtOnce  sync.Once
)

func pidExistsWithContext(ctx context.Context, pid int32) (bool, error) {
	if pid == 0 { // special case for pid 0 System Idle Process
		return true, nil
	}
	if pid < 0 {
		return false, fmt.Errorf("invalid pid %v", pid)
	}

	procPath := fmt.Sprintf("/proc/%d", pid)
	if _, err := os.Stat(procPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	stasFile := fmt.Sprintf("/proc/%d/stat", pid)
	statData, err := ioutil.ReadFile(stasFile)
	if err != nil {
		return false, err
	}

	var pidNum int
	var comm string
	var state byte
	_, err = fmt.Sscanf(string(statData), "%d (%s %c", &pidNum, &comm, &state)
	if err != nil {
		return false, err
	}

	if state == 'Z' {
		return false, nil
	}

	return true, nil
}

func VirtualizationWithContext(ctx context.Context) (string, string, error) {
	var system, role string

	// if cached already, return from cache
	cachedVirtMutex.RLock() // unlock won't be deferred so concurrent reads don't wait for long
	if cachedVirtMap != nil {
		cachedSystem, cachedRole := cachedVirtMap["system"], cachedVirtMap["role"]
		cachedVirtMutex.RUnlock()
		return cachedSystem, cachedRole, nil
	}
	cachedVirtMutex.RUnlock()

	filename := cpu.HostProc("xen")
	if PathExists(filename) {
		system = "xen"
		role = "guest" // assume guest

		if PathExists(filepath.Join(filename, "capabilities")) {
			contents, err := cpu.ReadLines(filepath.Join(filename, "capabilities"))
			if err == nil && StringsContains(contents, "control_d") {
				role = "host"
			}
		}
	}

	filename = cpu.HostProc("modules")
	if PathExists(filename) {
		contents, err := cpu.ReadLines(filename)
		if err == nil {
			switch {
			case StringsContains(contents, "kvm"):
				system = "kvm"
				role = "host"
			case StringsContains(contents, "vboxdrv"):
				system = "vbox"
				role = "host"
			case StringsContains(contents, "vboxguest"):
				system = "vbox"
				role = "guest"
			case StringsContains(contents, "vmware"):
				system = "vmware"
				role = "guest"
			}
		}
	}

	filename = cpu.HostProc("cpuinfo")
	if PathExists(filename) {
		contents, err := cpu.ReadLines(filename)
		if err == nil {
			if StringsContains(contents, "QEMU Virtual CPU") ||
				StringsContains(contents, "Common KVM processor") ||
				StringsContains(contents, "Common 32-bit KVM processor") {
				system = "kvm"
				role = "guest"
			}
		}
	}

	filename = cpu.HostProc("bus/pci/devices")
	if PathExists(filename) {
		contents, err := cpu.ReadLines(filename)
		if err == nil {
			if StringsContains(contents, "virtio-pci") {
				role = "guest"
			}
		}
	}

	filename = cpu.HostProc()
	if PathExists(filepath.Join(filename, "bc", "0")) {
		system = "openvz"
		role = "host"
	} else if PathExists(filepath.Join(filename, "vz")) {
		system = "openvz"
		role = "guest"
	}

	// not use dmidecode because it requires root
	if PathExists(filepath.Join(filename, "self", "status")) {
		contents, err := cpu.ReadLines(filepath.Join(filename, "self", "status"))
		if err == nil {
			if StringsContains(contents, "s_context:") ||
				StringsContains(contents, "VxID:") {
				system = "linux-vserver"
			}
			// TODO: guest or host
		}
	}

	if PathExists(filepath.Join(filename, "1", "environ")) {
		contents, err := ReadFile(filepath.Join(filename, "1", "environ"))

		if err == nil {
			if strings.Contains(contents, "container=lxc") {
				system = "lxc"
				role = "guest"
			}
		}
	}

	if PathExists(filepath.Join(filename, "self", "cgroup")) {
		contents, err := cpu.ReadLines(filepath.Join(filename, "self", "cgroup"))
		if err == nil {
			switch {
			case StringsContains(contents, "lxc"):
				system = "lxc"
				role = "guest"
			case StringsContains(contents, "docker"):
				system = "docker"
				role = "guest"
			case StringsContains(contents, "machine-rkt"):
				system = "rkt"
				role = "guest"
			case PathExists("/usr/bin/lxc-version"):
				system = "lxc"
				role = "host"
			}
		}
	}

	if PathExists(cpu.HostEtc("os-release")) {
		p, _, err := GetOSRelease()
		if err == nil && p == "coreos" {
			system = "rkt" // Is it true?
			role = "host"
		}
	}

	// before returning for the first time, cache the system and role
	cachedVirtOnce.Do(func() {
		cachedVirtMutex.Lock()
		defer cachedVirtMutex.Unlock()
		cachedVirtMap = map[string]string{
			"system": system,
			"role":   role,
		}
	})

	return system, role, nil
}

func PathExists(filename string) bool {
	if _, err := os.Stat(filename); err == nil {
		return true
	}
	return false
}

func StringsContains(target []string, src string) bool {
	for _, t := range target {
		if strings.Contains(t, src) {
			return true
		}
	}
	return false
}

func ReadFile(filename string) (string, error) {
	content, err := ioutil.ReadFile(filename)

	if err != nil {
		return "", err
	}

	return string(content), nil
}

func GetOSRelease() (platform string, version string, err error) {
	contents, err := cpu.ReadLines(cpu.HostEtc("os-release"))
	if err != nil {
		return "", "", nil // return empty
	}
	for _, line := range contents {
		field := strings.Split(line, "=")
		if len(field) < 2 {
			continue
		}
		switch field[0] {
		case "ID": // use ID for lowercase
			platform = trimQuotes(field[1])
		case "VERSION":
			version = trimQuotes(field[1])
		}
	}
	return platform, version, nil
}

func trimQuotes(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return s
}
