package godirlist

import (
	"fmt"
	"testing"
	"time"
)

func Test_1(t *testing.T) {
	fmt.Println("hi")

	test_abspath := `C:\Program Files`

	// 14 worked best on my 12 core machine with SSD
	// 48 worked best for high latency NAS folders
	worker_count := 14

	paths := []string{
		test_abspath,
	}
	startTime := time.Now()
	fsitem_chan := make(chan FsitemInfo)
	go func() {
		GenerateFsitemInfos(paths, fsitem_chan, worker_count)
		close(fsitem_chan)
	}()

	file_count := 0
	directory_count := 0
	other_count := 0
	total_size := int64(0)
	for fsitem := range fsitem_chan {
		// do stuff with found files ...

		if fsitem.fi.Mode().IsRegular() {
			s := fsitem.fi.Size()
			total_size += s
			file_count++
		} else if fsitem.fi.IsDir() {
			directory_count++
		} else {
			other_count++
		}
	}
	duration := time.Since(startTime)
	fmt.Println("wait:", duration, "bytes:", total_size, "files:", file_count, "directories:", directory_count, "other:", other_count)
}
