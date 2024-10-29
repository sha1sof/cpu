package process

import (
	"context"
	"cpuV3/a/cpu"
	"errors"
	"runtime"
	"sort"
	"time"
)

var (
	ErrorProcessNotRunning = errors.New("process does not exist")
)

type Process struct {
	Pid          int32 `json:"pid"`
	createTime   int64
	lastCPUTimes *cpu.TimesStat
	lastCPUTime  time.Time
}

type PageFaultsStat struct {
	MinorFaults      uint64 `json:"minorFaults"`
	MajorFaults      uint64 `json:"majorFaults"`
	ChildMinorFaults uint64 `json:"childMinorFaults"`
	ChildMajorFaults uint64 `json:"childMajorFaults"`
}

func NewProcess(pid int32) (*Process, error) {
	return newProcessWithContext(context.Background(), pid)
}

func newProcessWithContext(ctx context.Context, pid int32) (*Process, error) {
	p := &Process{
		Pid: pid,
	}

	exists, err := pidExistsWithContext(ctx, pid)
	if err != nil {
		return p, err
	}
	if !exists {
		return p, ErrorProcessNotRunning
	}
	p.CreateTimeWithContext(ctx)
	return p, nil
}

func pidsWithContext(ctx context.Context) ([]int32, error) {
	pids, err := pidsWithCtx(ctx)
	sort.Slice(pids, func(i, j int) bool { return pids[i] < pids[j] })
	return pids, err
}

func (p *Process) CreateTimeWithContext(ctx context.Context) (int64, error) {
	if p.createTime != 0 {
		return p.createTime, nil
	}
	createTime, err := p.createTimeWithContext(ctx)
	p.createTime = createTime
	return p.createTime, err
}

func (p *Process) PercentWithContext(ctx context.Context) (float64, error) {
	interval := cpu.GetTimeoutDuration(ctx)

	cpuTimes, err := p.timesWithContext(ctx)
	if err != nil {
		return 0, err
	}
	now := time.Now()

	if interval > 0 {
		p.lastCPUTimes = cpuTimes
		p.lastCPUTime = now
		if err := cpu.Sleep(ctx, interval); err != nil {
			return 0, err
		}
		cpuTimes, err = p.timesWithContext(ctx)
		now = time.Now()
		if err != nil {
			return 0, err
		}
	} else {
		if p.lastCPUTimes == nil {
			// invoked first time
			p.lastCPUTimes = cpuTimes
			p.lastCPUTime = now
			return 0, nil
		}
	}

	numcpu := runtime.NumCPU()
	delta := (now.Sub(p.lastCPUTime).Seconds()) * float64(numcpu)
	ret := calculatePercent(p.lastCPUTimes, cpuTimes, delta, numcpu)
	p.lastCPUTimes = cpuTimes
	p.lastCPUTime = now
	return ret, nil
}

func calculatePercent(t1, t2 *cpu.TimesStat, delta float64, numcpu int) float64 {
	if delta == 0 {
		return 0
	}
	delta_proc := t2.Total() - t1.Total()
	overall_percent := ((delta_proc / delta) * 100) * float64(numcpu)
	return overall_percent
}
