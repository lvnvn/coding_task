package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"task/storage"
)

type App struct {
	counter *storage.PersistentCounter
	queue   chan int
}

func (app *App) requestsCounter(w http.ResponseWriter, req *http.Request) {
	timestamp := time.Now().Unix()
	log.Printf("Recieved request at %d", timestamp)
	io.WriteString(w, fmt.Sprintf("Request count in the last minute: %d\n", app.counter.Count()))
	app.counter.Add(timestamp)
	app.queue <- 1
}

func main() {
	app := &App{
		counter: storage.Init("backup"),
		queue:   make(chan int, 1000),
	}

	// asyncronously persisting data to file on each request
	go func() {
		for _ = range app.queue {
			app.counter.DumpToFile()
		}
	}()

	// delete obsolete data from file once every minute
	go func() {
		for {
			app.counter.Clean()
			time.Sleep(1 * time.Minute)
		}
	}()

	// run the server
	port := ":8080"
	http.HandleFunc("/requests", app.requestsCounter)
	log.Printf("Starting server on %s", port)
	err := http.ListenAndServe(port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
