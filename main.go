package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"task/storage"
)

type App struct {
	counter *storage.PersistentCounter
	queue   chan int64
	debug   bool
}

func initApp(filename string) *App {
	debug, err := strconv.ParseBool(os.Getenv("DEBUG"))
	if err != nil {
		debug = false
	}
	log.Printf("Initializing app, debug = %t", debug)
	return &App{
		counter: storage.Init(filename),
		queue:   make(chan int64, 1000),
		debug:   debug,
	}
}

func (app *App) requestsCounter(w http.ResponseWriter, req *http.Request) {
	timestamp := time.Now().Unix()
	if app.debug {
		log.Printf("Recieved request at %d", timestamp)
	}
	fmt.Fprintf(w, "Request count in the last minute: %d\n", app.counter.Count())
	app.counter.Add(timestamp)
	app.queue <- timestamp
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
	app := initApp("backup")
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
