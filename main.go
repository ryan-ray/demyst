package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/sync/errgroup"
)

const apiStub = "https://jsonplaceholder.typicode.com/todos"

type ToDoItem struct {
	UserID    int    `json:"userId"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

var ErrNon200Response = errors.New("non 200 response")

func ToDoFetcher(client *http.Client, urlFn URLGenerator) func(context.Context) ([]byte, error) {
	return func(ctx context.Context) ([]byte, error) {
		var buf bytes.Buffer

		endpoint := urlFn()
		slog.Debug("GET", "endpoint", endpoint)
		req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
		if err != nil {
			return nil, err
		}

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if resp.StatusCode != http.StatusOK {
			slog.Error("Response", "endpoint", endpoint, "status", resp.StatusCode)
			return nil, ErrNon200Response
		}

		if _, err := io.Copy(&buf, resp.Body); err != nil {
			return nil, err
		}
		resp.Body.Close()

		return buf.Bytes(), nil
	}
}

type URLGenerator func() string

// Even will generate a url with an ID that increments by 2 on each call.
// This effectively means that only urls with even ID path params will be
// generated.
//
// Returns a URLGenerator function
func Even(endpoint string) func() string {
	var start int
	return func() string {
		start += 2
		return fmt.Sprintf("%s/%d", endpoint, start)
	}
}

// Threes is an example of a different type of URL generator that could be
// implemented, in this case the url generated will always have an ID path
// param divisible by 3. Not bothering with test coverage for this as it's
// just an example.
//
// Returns a URLGenerator function
func Threes(endpoint string) func() string {
	var start int
	return func() string {
		start += 3
		return fmt.Sprintf("%s/%d", endpoint, start)
	}
}

func main() {

	var (
		debug               bool
		workerCount         int
		count               int
		httpTimeout         int
		applicationDeadline int
	)

	fs := flag.NewFlagSet("todofetcher", flag.ExitOnError)
	fs.IntVar(&count, "count", 20, "The amount of todo items to fetch")
	fs.IntVar(&workerCount, "workers", 5, "The amount of concurrent workers to use")
	fs.BoolVar(&debug, "debug", false, "Print debugging output to stdout")
	fs.IntVar(&httpTimeout, "timeout", 30, "The timeout in seconds for the HTTP client config")
	fs.IntVar(&applicationDeadline, "deadline", 180, "The timeout for the application to complete processing")
	fs.Parse(os.Args[1:])

	fs.Usage = func() {
		fmt.Println("ToDo Fetcher")
		fmt.Println("Fetches a list of ToDo items from the jsonplaceholder.typicode.com service")
	}

	logLevel := slog.LevelError
	if debug {
		logLevel = slog.LevelDebug
	}

	slog.SetLogLoggerLevel(logLevel)

	if err := run(
		count,
		workerCount,
		time.Duration(httpTimeout)*time.Second,
		time.Duration(applicationDeadline)*time.Second,
		debug,
	); err != nil {
		slog.Error("Run error", "error", err.Error())
	}

}

var ErrApplicationDeadlineExceeded = errors.New("application deadline exceeded")

func run(count int, workers int, timeout time.Duration, deadline time.Duration, debug bool) error {
	fetchToDo := ToDoFetcher(
		&http.Client{
			Timeout: timeout,
		},
		Even(apiStub),
	)

	ctx, cancel := context.WithTimeout(context.Background(), deadline)
	defer cancel()

	buf := make([]*ToDoItem, 64)

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(workers)

loop:
	for range count {
		select {
		case <-ctx.Done():
			slog.Error("Execution exceeded deadline", "deadline", deadline)
			break loop
		default:
			g.Go(func() error {
				data, err := fetchToDo(ctx)
				if err != nil {
					return err
				}

				item := &ToDoItem{}
				if err := json.Unmarshal(data, item); err != nil {
					return err
				}

				// Generally speaking I would want to handle this case more
				// gracefully, but it would massively depend on the business
				// context. In this case, if there's an error with the payload,
				// we are just going to log it and move on.
				if item.ID <= 0 {
					slog.Error("API", "item_id", item.ID, "error", "less than or equal to zero")
					return nil
				}

				// For a more complex code base I'd probably encapsulate this
				// in its own type, but given this is pretty lean, I think it's
				// fine to have this inline.
				for item.ID >= len(buf) {
					newBuf := make([]*ToDoItem, len(buf)*2)
					copy(newBuf, buf)
					buf = newBuf
				}

				buf[item.ID] = item
				return nil
			})
		}
	}

	if err := g.Wait(); err != nil {
		slog.Error("Fetching todo items", "error", err)
	}

	for _, item := range buf {
		if item != nil {
			fmt.Printf("ID: %-7dCompleted: %-9tTitle: %s\n", item.ID, item.Completed, item.Title)
		}
	}

	return nil
}
