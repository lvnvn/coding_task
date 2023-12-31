// Integration tests for /requests endpoint
package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"
)

func checkRequestsCount(t *testing.T, resp *http.Response, count int) {
	expected := fmt.Sprintf("Request count in the last minute: %d\n", count)
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Error reading response. Expected: '%s'", expected)
	}
	if string(got) != expected {
		t.Errorf("Expected: '%s', got: '%s'", expected, got)
	}
}

// Setup and Teardown for each test
func clear() {
	os.Remove("backup_test")
}

func TestSingleRequest(t *testing.T) {
	clear()
	defer clear()
	testServer := httptest.NewServer(http.HandlerFunc(initApp("backup_test", 10, time.Millisecond).requestsCounter))
	testClient := testServer.Client()

	resp, _ := testClient.Get(testServer.URL)
	if resp.StatusCode != http.StatusOK {
		t.Errorf("response code is not 200: %d", resp.StatusCode)
	}
	checkRequestsCount(t, resp, 0)
}

func TestCounter(t *testing.T) {
	clear()
	defer clear()
	testServer := httptest.NewServer(http.HandlerFunc(initApp("backup_test", 10, time.Millisecond).requestsCounter))
	testClient := testServer.Client()
	var resp *http.Response

	for i := 0; i < 5; i++ {
		resp, _ = testClient.Get(testServer.URL)
	}
	checkRequestsCount(t, resp, 4) // Request counter is working correctly

	time.Sleep(20 * time.Second)
	for i := 0; i < 3; i++ {
		testClient.Get(testServer.URL)
	}
	time.Sleep(45 * time.Second)
	resp, _ = testClient.Get(testServer.URL)
	checkRequestsCount(t, resp, 3) // Request counter does not count requests earlier than a minute ago
}

func TestCounterAfterRestart(t *testing.T) {
	clear()
	defer clear()
	app := initApp("backup_test", 10, time.Millisecond)
	app.runPersisting()
	app.runCleaning()
	testServer := httptest.NewServer(http.HandlerFunc(app.requestsCounter))
	testClient := testServer.Client()
	var resp *http.Response

	for i := 0; i < 5; i++ {
		testClient.Get(testServer.URL)
	}

	testServer.Close()
	restartedTestServer := httptest.NewServer(http.HandlerFunc(initApp("backup_test", 10, time.Millisecond).requestsCounter))
	defer restartedTestServer.Close()
	restartedTestClient := testServer.Client()

	resp, _ = restartedTestClient.Get(restartedTestServer.URL)
	checkRequestsCount(t, resp, 5) // Old counter is kept after restart

	resp, _ = restartedTestClient.Get(restartedTestServer.URL)
	checkRequestsCount(t, resp, 6) // Counting continues correctly after restart
}

func TestRateLimit(t *testing.T) {
	clear()
	defer clear()
	sleepTimeSeconds := time.Second * 2
	rateLimit := 5

	testServer := httptest.NewServer(http.HandlerFunc(initApp("backup_test", rateLimit, sleepTimeSeconds).requestsCounter))
	testClient := testServer.Client()

	timestampBefore := time.Now()

	// Send rateLimit + 1 requests
	var wg sync.WaitGroup
	for i := 0; i <= rateLimit; i++ {
		wg.Add(1)
		go func() {
			testClient.Get(testServer.URL)
			wg.Done()
		}()
	}
	wg.Wait()

	timestampAfter := time.Now()

	// Expected time = waiting for previous request (sleepTimeSeconds) + current request (sleepTimeSeconds)
	expected := sleepTimeSeconds * 2

	if timestampAfter.Sub(timestampBefore) < expected {
		t.Errorf("6th request wasn't rate limited")
	}
}
