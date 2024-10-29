package a

import (
	"context"
	"cpuV3/a/cpu"
	"cpuV3/a/process"
	"fmt"
	"os"
	"sync"
)

func GetCpuUsage(ctx context.Context) (usage uint32, currentProcessUsage uint32, err error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()

		resultUsage, err := cpu.PercentWithContext(ctx)
		if err != nil {
			errChan <- err
			usage = 0
			fmt.Printf("Error resultUsage: %v", err)
			return
		}

		usage = uint32(resultUsage[0])
	}()

	go func() {
		defer wg.Done()

		pid := os.Getpid()
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			errChan <- err
			fmt.Printf("Error resultUsage: %v", err)
			return
		}

		resultCurrentProcessUsage, err := p.PercentWithContext(ctx)
		if err != nil {
			errChan <- err
			currentProcessUsage = 0
			fmt.Printf("Error resultCurrentProcessUsage: %v", err)
			return
		}

		totalLogicalCores, _ := cpu.Counts(true)
		currentProcessUsage = uint32((resultCurrentProcessUsage / 100) * float64(totalLogicalCores))
	}()

	go func() {
		wg.Wait()
		close(errChan)
	}()

	for e := range errChan {
		if err != nil {
			err = e
		} else {
			return usage, currentProcessUsage, err
		}
	}

	return
}
