package godirlist

import (
	"fmt"
	"testing"
	"time"
)

func Test_1(t *testing.T) {
	fmt.Println("hi")
	test_abspath := `C:\Program Files`
	//test_abspath = `C:\`
	worker_count := 64
	paths := []string{
		test_abspath,
	}
	startTime := time.Now()
	fsitem_chan := make(chan FsitemInfo)

	result_handler := func(some_results []FsitemInfo) {
		for _, result := range some_results {
			_ = result
			fsitem_chan <- result
		}
	}

	go func() {
		GenerateFsitemInfos(paths, result_handler, worker_count)
		close(fsitem_chan)
	}()
	file_count := 0
	directory_count := 0
	other_count := 0
	total_size := int64(0)
	for fsitem := range fsitem_chan {
		// do stuff with found files ...
		if fsitem.Fi.Mode().IsRegular() {
			s := fsitem.Fi.Size()
			total_size += s
			file_count++
		} else if fsitem.Fi.IsDir() {
			directory_count++
		} else {
			other_count++
		}
	}
	duration := time.Since(startTime)
	fmt.Println("wait:", duration, "bytes:", total_size, "files:", file_count, "directories:", directory_count, "other:", other_count)
}
