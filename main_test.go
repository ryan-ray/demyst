package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

const endpoint = "https://jsonplaceholder.typicode.com/todos"

func TestEven(t *testing.T) {
	genFn := Even(endpoint)

	i := 0
	for range 10 {
		got := genFn()
		i += 2
		want := fmt.Sprintf("%s/%d", endpoint, i)
		if got != want {
			t.Errorf("Got generated url: %q, want %q", got, want)
		}
	}
}

func TestToDoFetcher(t *testing.T) {
	tests := []struct {
		name   string
		status int
		err    error
	}{
		{name: "StatusOK", status: http.StatusOK, err: nil},
		{name: "StatusNon200", status: http.StatusNotFound, err: ErrNon200Response},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	fetch := ToDoFetcher(
		srv.Client(),
		Even(srv.URL),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv.Config.Handler = http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
				},
			)

			ctx := context.Background()
			_, err := fetch(ctx)
			if !errors.Is(err, tt.err) {
				t.Errorf("Got error %v, want %v", err, tt.err)
			}
		})
	}
}
