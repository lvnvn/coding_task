package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"task/storage"
)

type App struct {
	counter *storage.PersistentCounter
	queue   chan int
}

func initApp() *App {
	return &App{
		counter: storage.Init("backup"),
		queue:   make(chan int, 1000),
	}
}

func (app *App) requestsCounter(w http.ResponseWriter, req *http.Request) {
	timestamp := time.Now().Unix()
	log.Printf("Recieved request at %d", timestamp)
	fmt.Fprintf(w, "Request count in the last minute: %d\n", app.counter.Count())
	app.counter.Add(timestamp)
	app.queue <- 1
}

// Asyncronously persisting data to file on each request
func (app *App) runPersisting() {
	go func() {
		for _ = range app.queue {
			app.counter.DumpToFile()
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
	app := initApp()
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
