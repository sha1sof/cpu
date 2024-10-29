//go:build linux
// +build linux

package cpu

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func HostProc(combineWith ...string) string {
	return getEnv("HOST_PROC", "/proc", combineWith...)
}

func hostSys(combineWith ...string) string {
	return getEnv("HOST_SYS", "/sys", combineWith...)
}

func getEnv(key string, dfault string, combineWith ...string) string {
	value := os.Getenv(key)
	if value == "" {
		value = dfault
	}

	switch len(combineWith) {
	case 0:
		return value
	case 1:
		return filepath.Join(value, combineWith[0])
	default:
		all := make([]string, len(combineWith)+1)
		all[0] = value
		copy(all[1:], combineWith)
		return filepath.Join(all...)
	}
}

func ReadLines(filename string) ([]string, error) {
	return readLinesOffsetN(filename, 0, -1)
}

func readLinesOffsetN(filename string, offset uint, n int) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return []string{""}, err
	}
	defer f.Close()

	var ret []string

	r := bufio.NewReader(f)
	for i := 0; i < n+int(offset) || n < 0; i++ {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}
		if i < int(offset) {
			continue
		}
		ret = append(ret, strings.Trim(line, "\n"))
	}

	return ret, nil
}

func HostEtc(combineWith ...string) string {
	return getEnv("HOST_ETC", "/etc", combineWith...)
}
