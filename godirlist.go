package godirlist

// fast dir listing

// the good:
// - faster than any other file listing code I'm aware of (including robocopy and windirstat)
// - the code has been optimized very little, so there are probably ways to make it faster still
// - ~100 lines of verbose code, so should be easy to:
//   - understand
//   - clean up
//   - change
//   - integrate into code bases
// - contains example usage
// - aside from the caveats below it has worked reliably for me
//
// the bad:
// - not adhering to any coding style
// - probably not using good go concurrency patterns
// - not tested with junctions/symlinks/netmounted/user fs/etc.
// - not tested with long paths
// - not handling errors
// - the 2 similar select statements are not elegant
// - the buffer is not bounded (might cause issues if your 1 TiB video game library with the memory of a 20 year old phone)
//
// I'd be happy to hear of speed improvements! :)
//

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
)

type FsitemInfo struct {
	fi      os.FileInfo
	abspath string
}

func GenerateFsitemInfos(
	start_dir_abspaths []string,
	results chan FsitemInfo,
	worker_count int,
) {
	var incomplete_request_count int64

	work_requests := make(chan string)
	buffer_requests := make(chan string)
	done := make(chan struct{})

	for i := 0; i < worker_count; i++ {
		go dir_listing_worker(
			work_requests,
			buffer_requests,
			results,
			&incomplete_request_count,
			done,
		)
	}

	var buffer []string

	for _, start_dir_abspath := range start_dir_abspaths {
		buffer = append(buffer, start_dir_abspath)
		atomic.AddInt64(&incomplete_request_count, 1)
	}

	for {
		if len(buffer) > 0 {
			select {
			case work_requests <- buffer[0]:
				{
					buffer = buffer[1:]
				}
			case buffer_request := <-buffer_requests:
				{
					buffer = append(buffer, buffer_request)
				}
			}
		} else {
			select {
			case buffer_request := <-buffer_requests:
				{
					buffer = append(buffer, buffer_request)
				}
			case <-done:
				{
					goto exit_for
				}
			}
		}
	}
exit_for:
}

func dir_listing_worker(
	work_requests chan string,
	buffer_requests chan string,
	results chan FsitemInfo,
	incomplete_request_count *int64,
	done chan struct{},
) {
	for request := range work_requests {
		f, _ := os.Open(request)
		for {
			fsitems, err := f.Readdir(1)
			if err == io.EOF || len(fsitems) == 0 {
				break
			}
			fsitem := fsitems[0]
			abspath := filepath.Join(request, fsitem.Name())
			fsi := FsitemInfo{
				fsitem,
				abspath,
			}
			results <- fsi

			if fsitem.IsDir() {
				atomic.AddInt64(incomplete_request_count, 1)
				buffer_requests <- abspath
			}
		}
		f.Close()
		atomic.AddInt64(incomplete_request_count, -1)
		if atomic.LoadInt64(incomplete_request_count) == 0 {
			done <- struct{}{}
		}
	}
}
