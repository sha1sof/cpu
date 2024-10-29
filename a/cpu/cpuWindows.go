//go:build windows
// +build windows

package cpu

import (
	"context"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows"
	"strings"
	"unsafe"
)

var (
	procGetActiveProcessorCount = modkernel32.NewProc("GetActiveProcessorCount")
	procGetNativeSystemInfo     = modkernel32.NewProc("GetNativeSystemInfo")
)

type systemInfo struct {
	wProcessorArchitecture      uint16
	wReserved                   uint16
	dwPageSize                  uint32
	lpMinimumApplicationAddress uintptr
	lpMaximumApplicationAddress uintptr
	dwActiveProcessorMask       uintptr
	dwNumberOfProcessors        uint32
	dwProcessorType             uint32
	dwAllocationGranularity     uint32
	wProcessorLevel             uint16
	wProcessorRevision          uint16
}

type Win32_ProcessorWithoutLoadPct struct {
	Family                    uint16
	Manufacturer              string
	Name                      string
	NumberOfLogicalProcessors uint32
	NumberOfCores             uint32
	ProcessorID               *string
	Stepping                  *string
	MaxClockSpeed             uint32
}

func times() ([]TimesStat, error) {
	return timesWithContext()
}

func timesWithContext() ([]TimesStat, error) {
	var ret []TimesStat
	var lpIdleTime FILETIME
	var lpKernelTime FILETIME
	var lpUserTime FILETIME
	r, _, _ := procGetSystemTimes.Call(
		uintptr(unsafe.Pointer(&lpIdleTime)),
		uintptr(unsafe.Pointer(&lpKernelTime)),
		uintptr(unsafe.Pointer(&lpUserTime)))
	if r == 0 {
		return ret, windows.GetLastError()
	}

	LOT := float64(0.0000001)
	HIT := (LOT * 4294967296.0)
	idle := ((HIT * float64(lpIdleTime.DwHighDateTime)) + (LOT * float64(lpIdleTime.DwLowDateTime)))
	user := ((HIT * float64(lpUserTime.DwHighDateTime)) + (LOT * float64(lpUserTime.DwLowDateTime)))
	kernel := ((HIT * float64(lpKernelTime.DwHighDateTime)) + (LOT * float64(lpKernelTime.DwLowDateTime)))
	system := (kernel - idle)

	ret = append(ret, TimesStat{
		CPU:    "cpu-total",
		Idle:   float64(idle),
		User:   float64(user),
		System: float64(system),
	})
	return ret, nil
}

func countsWithContext(ctx context.Context, logical bool) (int, error) {
	if logical {
		err := procGetActiveProcessorCount.Find()
		if err == nil {
			ret, _, _ := procGetActiveProcessorCount.Call(uintptr(0xffff))
			if ret != 0 {
				return int(ret), nil
			}
		}
		var systemInfo systemInfo
		_, _, err = procGetNativeSystemInfo.Call(uintptr(unsafe.Pointer(&systemInfo)))
		if systemInfo.dwNumberOfProcessors == 0 {
			return 0, err
		}
		return int(systemInfo.dwNumberOfProcessors), nil
	}
	var dst []Win32_ProcessorWithoutLoadPct
	q := wmi.CreateQuery(&dst, "")
	q = strings.ReplaceAll(q, "Win32_ProcessorWithoutLoadPct", "Win32_Processor")
	if err := wmiQueryWithContext(ctx, q, &dst); err != nil {
		return 0, err
	}
	var count uint32
	for _, d := range dst {
		count += d.NumberOfCores
	}
	return int(count), nil
}
