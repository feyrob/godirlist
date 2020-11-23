package godirlist

// fast dir listing

// the good:
// - much faster than any other file listing code I'm aware of (including robocopy and windirstat)
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
// - not tested with junctions/symlinks/netmounted/user fs/etc.
// - not tested with long paths
// - no error handling
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
		// Note, that this depends on how go will not write to a channel that is nil...
		sink_chan := work_requests
		exit_chan := done
		next_buffer_val := ""

		if len(buffer) > 0 {
			// don't exit if there is still content in the buffer to process
			exit_chan = nil
			next_buffer_val = buffer[0]
		} else {
			// buffer is empty, don't read from it
			sink_chan = nil
		}

		select {
		case sink_chan <- next_buffer_val:
			{
				buffer = buffer[1:]
			}
		case buffer_request := <-buffer_requests:
			{
				buffer = append(buffer, buffer_request)
			}
		case <-exit_chan:
			{
				return
			}
		}
	}
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
