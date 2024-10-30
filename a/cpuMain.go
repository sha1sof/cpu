package a

import (
	"context"
	"cpuV3/a/cpu"
	"cpuV3/a/process"
	"os"
	"sync"
)

func GetCpuUsage(ctx context.Context) (usage uint32, currentProcessUsage uint32, err error) {
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	wg.Add(2)

	go func() {
		defer wg.Done()

		resultUsage, err := cpu.PercentWithContext(ctx)
		if err != nil {
			errChan <- err
			usage = 0
		} else {
			usage = uint32(resultUsage[0])
		}

	}()

	go func() {
		defer wg.Done()

		pid := os.Getpid()
		p, err := process.NewProcess(int32(pid))
		if err != nil {
			errChan <- err
			return
		}

		resultCurrentProcessUsage, err := p.PercentWithContext(ctx)
		if err != nil {
			errChan <- err
			currentProcessUsage = 999
			return
		}

		totalLogicalCores, err := cpu.Counts(true)
		if err != nil {
			errChan <- err
		}
		currentProcessUsage = uint32((resultCurrentProcessUsage / 100) * float64(totalLogicalCores))
	}()

	wg.Wait()
	close(errChan)
	/*	go func() {
		wg.Wait()
		close(errChan)
	}()*/

	for e := range errChan {
		if e != nil {
			err = e
		} else {
			return usage, currentProcessUsage, err
		}
	}

	return
}
