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

	ctx, cancle := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancle()

	usage, currentProcessUsage, err := a.GetCpuUsage(ctx)
	if err != nil {
		fmt.Printf("Error getting cpu usage: %v\n", err)
	} else {
		fmt.Printf("CPU Usage: %v \n", usage)
		fmt.Printf("currentProcessUsage Usage: %v \n", currentProcessUsage)
	}
}
