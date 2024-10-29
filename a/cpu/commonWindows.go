//go:build windows
// +build windows

package cpu

import (
	"context"
	"github.com/yusufpapurcu/wmi"
	"golang.org/x/sys/windows"
)

var (
	modkernel32        = windows.NewLazySystemDLL("kernel32.dll")
	procGetSystemTimes = modkernel32.NewProc("GetSystemTimes")
)

type FILETIME struct {
	DwLowDateTime  uint32
	DwHighDateTime uint32
}

func wmiQueryWithContext(ctx context.Context, query string, dst interface{}, connectServerArgs ...interface{}) error {
	if _, ok := ctx.Deadline(); !ok {
		ctxTimeout, cancel := context.WithTimeout(ctx, timeout)
		defer cancel()
		ctx = ctxTimeout
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- wmi.Query(query, dst, connectServerArgs...)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}
