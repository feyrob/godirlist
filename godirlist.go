package godirlist

// fast dir listing

// the good:
// - much faster than any other file listing code I'm aware of (including robocopy and windirstat)
// - ~100 lines of verbose code, so should be easy to:
//   - understand
//   - clean up
//   - change
//   - integrate into code bases
// - contains example usage
// - aside from the caveats below it has worked reliably for me
//
// the bad:
// - no error handling

import (
	"io"
	"os"
	"path/filepath"
	"sync/atomic"
)

type FsitemInfo struct {
	Fi      os.FileInfo
	Abspath string
}

func GenerateFsitemInfos(
	start_dir_abspaths []string,
	results chan FsitemInfo,
	worker_count int,
) {
	dirlist_chan := make(chan string)
	queue_chan := make(chan string)
	var incomplete_request_count int64
	done_chan := make(chan struct{})

	for i := 0; i < worker_count; i++ {
		go dir_listing_worker(
			results,
			dirlist_chan,
			queue_chan,
			&incomplete_request_count,
			done_chan,
		)
	}

	var queue []string
	for _, start_dir_abspath := range start_dir_abspaths {
		queue = append(queue, start_dir_abspath)
		atomic.AddInt64(&incomplete_request_count, 1)
	}

	for {
		// Note, that this depends on how go will not write to a channel that is nil...
		sink := dirlist_chan
		exit_chan := done_chan
		next_queue_val := ""

		if len(queue) > 0 {
			// don't exit if there is still content in the queue to process
			exit_chan = nil
			next_queue_val = queue[0]
		} else {
			// queue is empty, don't read from it
			sink = nil
		}

		select {
		case sink <- next_queue_val:
			{
				queue = queue[1:]
			}
		case queue_request := <-queue_chan:
			{
				queue = append(queue, queue_request)
			}
		case <-exit_chan:
			{
				return
			}
		}
	}
}

func dir_listing_worker(
	result_chan chan FsitemInfo,
	dirlist_chan chan string,
	queue_chan chan string,
	incomplete_request_count *int64,
	done_chan chan struct{},
) {
	for request := range dirlist_chan {
		f, _ := os.Open(request)
		for {
			fsitems, err := f.Readdir(1)
			if err == io.EOF || len(fsitems) == 0 {
				break
			}
			fsitem := fsitems[0]
			abspath := filepath.Join(request, fsitem.Name())
			fsi := FsitemInfo{fsitem, abspath}
			result_chan <- fsi

			if fsitem.IsDir() {
				atomic.AddInt64(incomplete_request_count, 1)
				queue_chan <- abspath
			}
		}
		f.Close()
		atomic.AddInt64(incomplete_request_count, -1)
		if atomic.LoadInt64(incomplete_request_count) == 0 {
			done_chan <- struct{}{}
		}
	}
}
