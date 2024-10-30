package main

import (
	"context"
	"cpuV3/a"
	"fmt"
	"time"
)

func main() {
	go func() {
		i := 0
		for {
			i++
		}
	}()

	for {
		ctx, cancle := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancle()

		usage, currentProcessUsage, err := a.GetCpuUsage(ctx)
		if err != nil {
			fmt.Println("Error:", err)
			continue
		} else {
			fmt.Printf("usage: %v\n", usage)
			fmt.Printf("currentProcessUsage: %v\n", currentProcessUsage)
		}
	}
}
