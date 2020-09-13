package main

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
// - the 2 similar select statements bother me
// - as is, the code does not clean up after itself (should not be used repeatedly in a running service)
//   - speficially the workers never exit (should be easy to fix, but I didn't need it)
// - the buffer is not bounded (might cause issues if your 1 TiB video game library with the memory of a 20 year old phone)
//
// I'd be happy to hear of speed improvements! :)
//

import (
	"os"
	"fmt"
	"testing"
	"sync/atomic"
	"path/filepath"
	"time"
	"io"
)


func Test_example_1(t *testing.T){
	fmt.Println("hi")
	
	test_abspath := `d:\test`
	
	// 14 worked best on my 12 core machine with SSD
	// 48 worked best for high latency NAS folders
	worker_count := 14 
	
	paths := []string{
		test_abspath,
	}
	startTime := time.Now()
	fsitem_chan := make(chan T_fsitem_info)
	go func(){
		generate_fsitems(paths, fsitem_chan, worker_count)
		close(fsitem_chan)
	}()

	found_count := 0
	total_size := int64(0)
	for fsitem := range(fsitem_chan){
		// do stuff with found files ...
		if fsitem.fi.Mode().IsRegular(){
			s := fsitem.fi.Size()
			total_size += s
		}
		found_count++
	}

	duration := time.Since(startTime)
	size_str := fmt.Sprintf("%d", total_size)
	found_count_str := fmt.Sprintf("%d", found_count)
	fmt.Println("Duration: " + duration.String() + "  total_size: " + size_str + " found_count: " + found_count_str)
}


type T_fsitem_info struct {
	fi os.FileInfo
	abspath string
}


func generate_fsitems(
	start_dir_abspaths []string, 
	results chan T_fsitem_info,
	worker_count int,
){
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

	for _, start_dir_abspath := range(start_dir_abspaths){
		buffer = append(buffer, start_dir_abspath)
		atomic.AddInt64(&incomplete_request_count, 1)
	}
	
	for {
		if len(buffer) > 0 {
			select{
				case work_requests <- buffer[0]: {
					buffer = buffer[1:]
				}
				case buffer_request := <- buffer_requests: {
					buffer = append(buffer, buffer_request)
				}
			}
		}else{
			select{
				case buffer_request := <- buffer_requests: {
					buffer = append(buffer, buffer_request)
				}
				case <- done:{
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
	results chan T_fsitem_info,
	incomplete_request_count *int64,
	done chan struct{},
) {
	for request := range(work_requests) {
		f, _ := os.Open(request)
		fsitems, err := f.Readdir(1) 
		for err != io.EOF && len(fsitems) > 0{
			fsitem := fsitems[0]
			abspath := filepath.Join(request, fsitem.Name())
			fsi := T_fsitem_info{
				fsitem,
				abspath,
			}
			results <- fsi

			if fsitem.IsDir() {
				atomic.AddInt64(incomplete_request_count, 1)
				buffer_requests <- abspath
			}
			fsitems, err = f.Readdir(1)
		}
		f.Close()
		atomic.AddInt64(incomplete_request_count, -1)
		if atomic.LoadInt64(incomplete_request_count) == 0 {
			done <- struct{}{}
		}
	}
}
