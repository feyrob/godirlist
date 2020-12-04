package godirlist

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"
)

func Test_1(t *testing.T) {
	fmt.Println("hi")
	test_abspath := `C:\Program Files`
	test_abspath = `C:\`
	worker_count := 64
	paths := []string{
		test_abspath,
	}
	startTime := time.Now()

	file_count := int64(0)
	directory_count := int64(0)
	other_count := int64(0)
	total_size := int64(0)

	result_handler := func(some_results []FsitemInfo) {
		for _, fsitem := range some_results {

			//fsitem_chan <- fsitem
			// do stuff with found files ...
			if fsitem.Fi.Mode().IsRegular() {
				s := fsitem.Fi.Size()
				atomic.AddInt64(&total_size, s)
				atomic.AddInt64(&file_count, 1)

			} else if fsitem.Fi.IsDir() {
				atomic.AddInt64(&directory_count, 1)

			} else {
				atomic.AddInt64(&other_count, 1)
			}
		}
	}

	GenerateFsitemInfos(paths, result_handler, worker_count)

	duration := time.Since(startTime)
	fmt.Println("wait:", duration, "bytes:", total_size, "files:", file_count, "directories:", directory_count, "other:", other_count)
}
