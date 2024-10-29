//go:build linux
// +build linux

package process

import (
	"bytes"
	"context"
	"cpuV3/a/cpu"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"
)

var ClockTicks = 100

func pidsWithCtx(ctx context.Context) ([]int32, error) {
	return readPidsFromDir(cpu.HostProc())
}

func readPidsFromDir(path string) ([]int32, error) {
	var ret []int32

	d, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer d.Close()

	fnames, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	for _, fname := range fnames {
		pid, err := strconv.ParseInt(fname, 10, 32)
		if err != nil {
			// if not numeric name, just skip
			continue
		}
		ret = append(ret, int32(pid))
	}

	return ret, nil
}

func (p *Process) timesWithContext(ctx context.Context) (*cpu.TimesStat, error) {
	_, _, cpuTimes, _, _, _, _, err := p.fillFromStatWithContext(ctx)
	if err != nil {
		return nil, err
	}
	return cpuTimes, nil
}

func (p *Process) fillFromStatWithContext(ctx context.Context) (uint64, int32, *cpu.TimesStat, int64, uint32, int32, *PageFaultsStat, error) {
	return p.fillFromTIDStatWithContext(ctx, -1)
}

func (p *Process) fillFromTIDStatWithContext(ctx context.Context, tid int32) (uint64, int32, *cpu.TimesStat, int64, uint32, int32, *PageFaultsStat, error) {
	pid := p.Pid
	var statPath string

	if tid == -1 {
		statPath = cpu.HostProc(strconv.Itoa(int(pid)), "stat")
	} else {
		statPath = cpu.HostProc(strconv.Itoa(int(pid)), "task", strconv.Itoa(int(tid)), "stat")
	}

	contents, err := ioutil.ReadFile(statPath)
	if err != nil {
		return 0, 0, nil, 0, 0, 0, nil, err
	}
	// Indexing from one, as described in `man proc` about the file /proc/[pid]/stat
	fields := splitProcStat(contents)

	terminal, err := strconv.ParseUint(fields[7], 10, 64)
	if err != nil {
		return 0, 0, nil, 0, 0, 0, nil, err
	}

	ppid, err := strconv.ParseInt(fields[4], 10, 32)
	if err != nil {
		return 0, 0, nil, 0, 0, 0, nil, err
	}
	utime, err := strconv.ParseFloat(fields[14], 64)
	if err != nil {
		return 0, 0, nil, 0, 0, 0, nil, err
	}

	stime, err := strconv.ParseFloat(fields[15], 64)
	if err != nil {
		return 0, 0, nil, 0, 0, 0, nil, err
	}

	var iotime float64
	if len(fields) > 42 {
		iotime, err = strconv.ParseFloat(fields[42], 64)
		if err != nil {
			iotime = 0
		}
	} else {
		iotime = 0
	}

	cpuTimes := &cpu.TimesStat{
		CPU:    "cpu",
		User:   utime / float64(ClockTicks),
		System: stime / float64(ClockTicks),
		Iowait: iotime / float64(ClockTicks),
	}

	bootTime, _ := BootTimeWithContext(ctx)
	t, err := strconv.ParseUint(fields[22], 10, 64)
	if err != nil {
		return 0, 0, nil, 0, 0, 0, nil, err
	}
	ctime := (t / uint64(ClockTicks)) + uint64(bootTime)
	createTime := int64(ctime * 1000)

	return terminal, int32(ppid), cpuTimes, createTime, 0, 0, nil, nil
}

func splitProcStat(content []byte) []string {
	nameStart := bytes.IndexByte(content, '(')
	nameEnd := bytes.LastIndexByte(content, ')')
	restFields := strings.Fields(string(content[nameEnd+2:])) // +2 skip ') '
	name := content[nameStart+1 : nameEnd]
	pid := strings.TrimSpace(string(content[:nameStart]))
	fields := make([]string, 3, len(restFields)+3)
	fields[1] = string(pid)
	fields[2] = string(name)
	fields = append(fields, restFields...)
	return fields
}

func (p *Process) createTimeWithContext(ctx context.Context) (int64, error) {
	_, _, _, createTime, _, _, _, err := p.fillFromStatWithContext(ctx)
	if err != nil {
		return 0, err
	}
	return createTime, nil
}

func BootTimeWithContext(ctx context.Context) (uint64, error) {
	system, role, err := Virtualization()
	if err != nil {
		return 0, err
	}

	statFile := "stat"
	if system == "lxc" && role == "guest" {
		// if lxc, /proc/uptime is used.
		statFile = "uptime"
	} else if system == "docker" && role == "guest" {
		// also docker, guest
		statFile = "uptime"
	}

	filename := cpu.HostProc(statFile)
	lines, err := cpu.ReadLines(filename)
	if err != nil {
		return 0, err
	}

	if statFile == "uptime" {
		if len(lines) != 1 {
			return 0, fmt.Errorf("wrong uptime format")
		}
		f := strings.Fields(lines[0])
		b, err := strconv.ParseFloat(f[0], 64)
		if err != nil {
			return 0, err
		}
		t := uint64(time.Now().Unix()) - uint64(b)
		return t, nil
	}
	if statFile == "stat" {
		for _, line := range lines {
			if strings.HasPrefix(line, "btime") {
				f := strings.Fields(line)
				if len(f) != 2 {
					return 0, fmt.Errorf("wrong btime format")
				}
				b, err := strconv.ParseInt(f[1], 10, 64)
				if err != nil {
					return 0, err
				}
				t := uint64(b)
				return t, nil
			}
		}
	}

	return 0, fmt.Errorf("could not find btime")
}
