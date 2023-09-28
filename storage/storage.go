package storage

import (
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
	b, err := os.ReadFile(c.file.filename)
	if err != nil {
		log.Print(err)
	}
	timestampStrings := strings.Split(string(b), ",")
	timestamps := []int64{}

	for i := len(timestampStrings) - 1; i >= 0; i-- {
		ts, err := strconv.ParseInt(timestampStrings[i], 10, 64)
		if err != nil {
			log.Print(err)
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
func (c *PersistentCounter) DumpToFile() {
	c.counter.RLock()
	timestampStrings := make([]string, len(c.counter.ts))
	for i, t := range c.counter.ts {
		timestampStrings[i] = strconv.FormatInt(t, 10)
	}
	c.counter.RUnlock()

	err := os.WriteFile(c.file.filename, []byte(strings.Join(timestampStrings, ",")), 0666)
	if err != nil {
		log.Print(err)
	}
}

// Clean deletes obsolete timestamps from backup file
func (c *PersistentCounter) Clean() {
	b, err := os.ReadFile(c.file.filename)
	if err != nil {
		log.Print(err)
	}
	timestampStrings := strings.Split(string(b), ",")
	freshTimestampStrings := []string{}
	lastMinute := time.Now().Add(-1 * time.Minute).Unix()

	for i := len(timestampStrings) - 1; i >= 0; i-- {
		ts, err := strconv.ParseInt(timestampStrings[i], 10, 64)
		if err != nil {
			log.Print(err)
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