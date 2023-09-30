package storage

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Counter struct {
	sync.RWMutex
	ts []int64
}

type File struct {
	sync.RWMutex
	filename string
}

func (f *File) SafeRead() (string, error) {
	f.RLock()
	defer f.RUnlock()
	res, err := os.ReadFile(f.filename)
	if err != nil {
		return "", err
	}
	if string(res) == "" {
		return "", errors.New("Backup file is empty")
	}
	return string(res), nil
}

func (f *File) SafeWrite(value string) {
	f.Lock()
	err := os.WriteFile(f.filename, []byte(value), 0666)
	if err != nil {
		log.Print(err)
	}
	f.Unlock()
}

func (f *File) SafeAppend(value int64) {
	f.Lock()
	defer f.Unlock()
	file, err := os.OpenFile(f.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Print(err)
		return
	}
	defer file.Close()
	if _, err := file.WriteString(fmt.Sprintf(",%d", value)); err != nil {
		log.Print(err)
	}
}

// Stores request timestamps in array and file, safe for concurrent use
type PersistentCounter struct {
	counter Counter
	file    File
}

func Init(filename string) *PersistentCounter {
	return &PersistentCounter{file: File{filename: filename}}
}

// Add puts current request timestamp into struct
func (c *PersistentCounter) Add(timestamp int64) {
	c.counter.Lock()
	c.counter.ts = append(c.counter.ts, timestamp)
	c.counter.Unlock()
}

// Count counts number of request timestamps in last 60 second window
func (c *PersistentCounter) Count() int {
	lastMinute := time.Now().Add(-1 * time.Minute).Unix()
	count := 0
	c.counter.RLock()
	if len(c.counter.ts) != 0 {
		for _, ts := range c.counter.ts {
			if ts > lastMinute {
				count += 1
			}
		}
		c.counter.RUnlock()
		return count
	}
	c.counter.RUnlock()

	// Service might have been restarted
	log.Printf("Loading backup from file")
	backup, err := c.file.SafeRead()
	if err != nil {
		log.Print(err)
		return count
	}
	timestampStrings := strings.Split(strings.Trim(backup, ","), ",")
	timestamps := []int64{}

	for i := len(timestampStrings) - 1; i >= 0; i-- {
		ts, err := strconv.ParseInt(timestampStrings[i], 10, 64)
		if err != nil {
			log.Print(err)
			continue
		}
		if ts >= lastMinute {
			count += 1
			timestamps = append(timestamps, ts)
		} else {
			// Timestamps are sorted since only one goroutine accepts requests, no need to iterate over every value
			break
		}
	}

	// Fill in timestamps list from backup
	c.counter.Lock()
	c.counter.ts = timestamps
	c.counter.Unlock()
	return count
}

// DumpToFile writes contents of timestamps array to file
func (c *PersistentCounter) DumpToFile(value int64) {
	// Queue for this function is FIFO, so timestamp order is kept without looking up c.counter.ts
	c.file.SafeAppend(value)
}

// Clean deletes obsolete timestamps from backup file
func (c *PersistentCounter) Clean() {
	c.file.Lock()
	defer c.file.Unlock()
	res, err := os.ReadFile(c.file.filename)
	if err != nil {
		log.Print(err)
		return
	}
	timestampStrings := strings.Split(strings.Trim(backup, ","), ",")
	freshTimestampStrings := []string{}
	lastMinute := time.Now().Add(-1 * time.Minute).Unix()

	for i := len(timestampStrings) - 1; i >= 0; i-- {
		ts, err := strconv.ParseInt(timestampStrings[i], 10, 64)
		if err != nil {
			log.Print(err)
			continue
		}
		if ts >= lastMinute {
			freshTimestampStrings = append(freshTimestampStrings, strconv.FormatInt(ts, 10))
		} else {
			// Timestamps are sorted since only one goroutine accepts requests, no need to iterate over every value
			break
		}
	}
	err = os.WriteFile(c.file.filename, []byte(strings.Join(freshTimestampStrings, ",")), 0666)
	if err != nil {
		log.Print(err)
	}
}
