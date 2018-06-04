// Copyright (C) 2017, ccpaging <ccpaging@gmail.com>.  All rights reserved.

package filelog

import (
	"sync"
	"time"
	"strings"
	"os"
	"path"
	l4g "github.com/ccpaging/nxlog4go"
)

// This log appender sends output to a file
type FileAppender struct {
	mu sync.Mutex 		 // ensures atomic writes; protects the following fields
	layout l4g.Layout 	 // format record for output
	// 2nd cache, formatted message
	messages chan []byte
	// 3nd cache, destination for output with buffered and rotated
	out *l4g.RotateFileWriter
	// Rotate at size
	maxsize int
	// Rotate cycle in seconds
	cycle, clock int
	// write loop
	loopRunning bool
	loopReset chan time.Time
}

// Write log record
func (fa *FileAppender) Write(rec *l4g.LogRecord) {
	fa.messages <- fa.layout.Format(rec)
}

func (fa *FileAppender) Init() {
	if fa.loopRunning {
		return
	}
	fa.loopRunning = true
	ready := make(chan struct{})
	go fa.writeLoop(ready)
	<-ready
}

// Close file
func (fa *FileAppender) Close() {
	close(fa.messages)

	// drain the log channel before closing
	for i := 10; i > 0; i-- {
		// Must call Sleep here, otherwise, may panic send on closed channel
		time.Sleep(100 * time.Millisecond)
		if len(fa.messages) <= 0 {
			break
		}
	}
	if fa.out != nil {
		fa.out.Close()
	}

	close(fa.loopReset)
}

// NewFileAppender creates a new appender which writes to the given file and
// has rotation enabled if maxbackup > 0.
func NewAppender(filename string, maxbackup int) l4g.Appender {
	return &FileAppender{
		layout: 	 l4g.NewPatternLayout(l4g.PATTERN_DEFAULT),	
		messages: 	 make(chan []byte,  l4g.LogBufferLength),
		out: 		 l4g.NewRotateFileWriter(filename).SetMaxBackup(maxbackup),
		cycle:		 86400,
		clock:		 -1,
		loopRunning: false,
		loopReset: 	 make(chan time.Time, l4g.LogBufferLength),
	}
}

func nextTime(cycle, clock int) time.Time {
	if cycle <= 0 {
		cycle = 86400
	}
	nrt := time.Now()
	if clock < 0 {
		// Now + cycle
		return nrt.Add(time.Duration(cycle) * time.Second)
	}
	
	// clock >= 0, next cycle midnight + clock
	nextCycle := nrt.Add(time.Duration(cycle) * time.Second)
	nrt = time.Date(nextCycle.Year(), nextCycle.Month(), nextCycle.Day(), 
					0, 0, 0, 0, nextCycle.Location())
	return nrt.Add(time.Duration(clock) * time.Second)
}

func (fa *FileAppender) writeLoop(ready chan struct{}) {
	defer func() {
		fa.loopRunning = false
	}()

	l4g.LogLogTrace("filelog", "cycle = %d, clock = %d, maxsize = %d", fa.cycle, fa.clock, fa.maxsize)
	if fa.cycle > 0 && fa.out.Size() > fa.maxsize {
		fa.out.Rotate()
	}

	nrt := nextTime(fa.cycle, fa.clock)
	rotTimer := time.NewTimer(nrt.Sub(time.Now()))
	l4g.LogLogTrace("filelog", "Next time is %v", nrt.Sub(time.Now()))

	close(ready)
	for {
		select {
		case bb, ok := <-fa.messages:
			fa.mu.Lock()
			fa.out.Write(bb)
			if len(fa.messages) <= 0 {
				fa.out.Flush()
			}
			fa.mu.Unlock()
			
			if !ok {
 				// drain the log channel and write directly
				fa.mu.Lock()
				for bb := range fa.messages {
					fa.out.Write(bb)
				}
				fa.mu.Unlock()
				return
			}
		case <-rotTimer.C:
			nrt = nextTime(fa.cycle, fa.clock)
			rotTimer.Reset(nrt.Sub(time.Now()))
			l4g.LogLogTrace("filelog", "Next time is %v", nrt.Sub(time.Now()))
			if fa.cycle > 0 && fa.out.Size() > fa.maxsize {
				fa.out.Rotate()
			}
		case <-fa.loopReset:
			l4g.LogLogTrace("filelog", "Reset. cycle = %d, clock = %d, maxsize = %d", fa.cycle, fa.clock, fa.maxsize)
			nrt = nextTime(fa.cycle, fa.clock)
			rotTimer.Reset(nrt.Sub(time.Now()))
			l4g.LogLogTrace("filelog", "Next time is %v", nrt.Sub(time.Now()))
		}
	}
}

// Set option. chainable
func (fa *FileAppender) Set(name string, v interface{}) l4g.Appender {
	fa.SetOption(name, v)
	return fa
}

/*
Set option. checkable. Better be set before SetFilters()
Option names include:
	filename  - The opened file
	flush	  - Flush file cache buffer size
	maxbackup - Max number for log file storage
	maxsize	  - Rotate at size
	pattern	  - Layout format pattern
	utc	  - Log recorder time zone
	head 	  - File head format layout pattern
	foot 	  - File foot format layout pattern
	cycle	  - The cycle seconds of checking rotate size
	clock	  - The seconds since midnight
	daily	  - Checking rotate size at every midnight
*/
func (fa *FileAppender) SetOption(name string, v interface{}) error {
	fa.mu.Lock()
	defer fa.mu.Unlock()

	switch name {
	case "filename":
		if filename, ok := v.(string); ok {
			if len(filename) <= 0 {
				return l4g.ErrBadValue
			}
			// Directory exist already, return nil
			err := os.MkdirAll(path.Dir(filename), l4g.FilePermDefault)
			if err != nil {
				return err
			}
			fa.out.SetFileName(filename)
		} else {
			return l4g.ErrBadValue
		}
	case "flush":
		var flush int
		switch value := v.(type) {
		case int:
			flush = value
		case string:
			flush = l4g.StrToNumSuffix(strings.Trim(value, " \r\n"), 1024)
		default:
			return l4g.ErrBadValue
		}
		fa.out.SetFlush(flush)
	case "maxbackup":
		var maxbackup int
		switch value := v.(type) {
		case int:
			maxbackup = value
		case string:
			maxbackup = l4g.StrToNumSuffix(strings.Trim(value, " \r\n"), 1)
		default:
			return l4g.ErrBadValue
		}
		fa.out.SetMaxBackup(maxbackup)
	case "maxsize":
		var maxsize int
		switch value := v.(type) {
		case int:
			maxsize = value
		case string:
			maxsize = l4g.StrToNumSuffix(strings.Trim(value, " \r\n"), 1024)
		default:
			return l4g.ErrBadValue
		}
		fa.maxsize = maxsize
		if fa.cycle <= 0 {
			fa.out.SetMaxSize(fa.maxsize)
		}
	case "pattern", "format", "utc":
		return fa.layout.SetOption(name, v)
	case "head":
		if header, ok := v.(string); ok {
			fa.out.SetHead(header)
		} else {
			return l4g.ErrBadValue
		}
	case "foot":
		if footer, ok := v.(string); ok {
			fa.out.SetFoot(footer)
		} else {
			return l4g.ErrBadValue
		}
	case "cycle":
		switch value := v.(type) {
		case int:
			fa.cycle = value
		case string:
			// Each with optional fraction and a unit suffix, 
			// such as "300ms", "-1.5h" or "2h45m". 
			// Valid time units are "ns", "us", "ms", "s", "m", "h".
			dur, _ := time.ParseDuration(value)
			fa.cycle = int(dur/time.Second)
		default:
			return l4g.ErrBadValue
		}
		if fa.cycle <= 0 {
			fa.out.SetMaxSize(fa.maxsize)
		} else {
			fa.out.SetMaxSize(0)
		}
		if fa.loopRunning {
			fa.loopReset <- time.Now()
		}
	case "clock", "delay0":
		switch value := v.(type) {
		case int:
			fa.clock = value
		case string:
			dur, _ := time.ParseDuration(value)
			fa.clock = int(dur/time.Second)
		default:
			return l4g.ErrBadValue
		}
		if fa.loopRunning {
			fa.loopReset <- time.Now()
		}
	case "daily":
		var daily bool
		switch value := v.(type) {
		case string:
			daily = strings.Trim(value, " \r\n") != "false"
		case bool:
			daily = value
		default:
			return l4g.ErrBadValue
		}
		if daily {
			fa.cycle = 86400
			fa.clock = 0
			fa.maxsize = 0
			fa.out.SetMaxSize(0)
			if fa.loopRunning {
				fa.loopReset <- time.Now()
			}
		}
	default:
		return l4g.ErrBadOption
	}
	return nil
}