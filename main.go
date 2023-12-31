package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"task/storage"
)

type App struct {
	counter        *storage.PersistentCounter
	queue          chan int64
	debug          bool
	rateLimitQueue chan int
	sleepTime      time.Duration
}

func initApp(filename string, rateLimit int, sleepTime time.Duration) *App {
	debug, err := strconv.ParseBool(os.Getenv("DEBUG"))
	if err != nil {
		debug = false
	}
	log.Printf("Initializing app, debug = %t", debug)
	return &App{
		counter:        storage.Init(filename),
		queue:          make(chan int64, 1000),
		debug:          debug,
		rateLimitQueue: make(chan int, rateLimit), // Proceed no more than n requests in parallel
		sleepTime:      sleepTime,
	}
}

func (app *App) requestsCounter(w http.ResponseWriter, req *http.Request) {
	app.rateLimitQueue <- 1

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		timestamp := time.Now().Unix()
		if app.debug {
			log.Printf("Recieved request at %d", timestamp)
		}
		fmt.Fprintf(w, "Request count in the last minute: %d\n", app.counter.Count())
		app.counter.Add(timestamp)
		app.queue <- timestamp
		wg.Done()
	}()
	go func() {
		time.Sleep(app.sleepTime)
		wg.Done()
	}()
	wg.Wait()

	<-app.rateLimitQueue
}

// Asyncronously persisting data to file on each request
func (app *App) runPersisting() {
	go func() {
		for value := range app.queue {
			app.counter.DumpToFile(value)
		}
	}()
}

// Delete obsolete data from file once every minute
func (app *App) runCleaning() {
	go func() {
		for {
			app.counter.Clean()
			time.Sleep(1 * time.Minute)
		}
	}()
}

func main() {
	app := initApp("backup", 5, time.Second*2)
	app.runPersisting()
	app.runCleaning()

	// run the server
	port := ":8080"
	http.HandleFunc("/requests", app.requestsCounter)
	log.Printf("Starting server on %s", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
