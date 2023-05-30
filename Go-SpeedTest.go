package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"gopkg.in/ini.v1"
)

type AtomicCounter int64

func (c *AtomicCounter) Write(p []byte) (n int, err error) {
	n = len(p)
	atomic.AddInt64((*int64)(c), int64(n))
	return
}

func main() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		fmt.Printf("Fail to read file: %v", err)
		os.Exit(1)
	}

	url := cfg.Section("").Key("url").String()

	counter := new(AtomicCounter)

	ringBuffer := make([]float64, 60)
	index := 0

	go func() {
		for {
			time.Sleep(time.Second)

			Mbps := float64(atomic.LoadInt64((*int64)(counter)) * 8 / 1024 / 1024)
			ringBuffer[index] = Mbps

			avg3s := averageSpeed(ringBuffer, index, 3)
			avg10s := averageSpeed(ringBuffer, index, 10)
			avg60s := averageSpeed(ringBuffer, index, 60)

			fmt.Printf("| Time      | Current Speed | 3s Average | 10s Average | 60s Average |\n")
			fmt.Printf("|-----------|---------------|------------|-------------|-------------|\n")
			fmt.Printf("| %s | %-13.2f | %-10.2f | %-11.2f | %-11.2f |\n", time.Now().Format("15:04:05"), Mbps, avg3s, avg10s, avg60s)

			atomic.StoreInt64((*int64)(counter), 0)
			index = (index + 1) % 60
		}
	}()

	res, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer res.Body.Close()

	_, _ = io.Copy(io.Discard, io.TeeReader(res.Body, counter))

	for {
	}
}

func averageSpeed(ringBuffer []float64, currentIndex, seconds int) float64 {
	start := (currentIndex + 1 + 60 - seconds) % 60
	total := 0.0
	for i := 0; i < seconds; i++ {
		total += ringBuffer[(start+i)%60]
	}
	return total / float64(seconds)
}