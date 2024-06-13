# Demyst Tech Test

## Usage

You can run directly with 

```
go run main.go <flags>
```

...or compile and run

```
go build -o todofetcher .
./todofetcher -h
```

or build and run with Docker

```
docker build . -t todofetcher
docker run todofetcher <flags>
```

Run with the `-h` flag to see the available options, but essentially;

```
  -count int
        The amount of todo items to fetch (default 20)
  -deadline int
        The timeout for the application to complete processing (default 180)
  -debug
        Print debugging output to stdout
  -timeout int
        The timeout in seconds for the HTTP client config (default 30)
  -workers int
        The amount of concurrent workers to use (default 5)
```

## Design

Given the scope and requirements of the problem, I've gone for a more simplistic
solution. Many developers when solving this problem would do so with structs and
interfaces, and that is a 100% valid approach. However in Go functions are
values. If you understand this and how to use closures for state, you can do
quite a lot if you don't need the mutation that implementing a struct type
allows.

It's also worth noting that the service https://jsonplaceholder.typicode.com
only allows for a maximum index of 200, anything above this will return a 404.
My solution assumes that this restriction is not in place, effectively meaning
that any 404 errors are treated as unplanned and handled accordingly. This is
important as it influences some of my design decisions.

One of these is that to enable parallel fetching, but retain in order 
processing, I am using a buffer that consumes more memory than may be initially
needed (and will grow as required), and the space complexity will be north of
O(n), but still linear. This means that I don't have to sort any of the result 
set after the fetching is complete so the time complexity for processing is
O(n), and the lookup of an individual item (provided you know the item ID) is
O(1). Some people can grumble about this approach, but slices essentially do the
same thing when you append to them.

## Other Considerations

Tasks like this can be very open to interpretation, the following is a list of
things that I would implement if I were writing this for a production 
environment, but that given the time constraints, are missing from this 
solution;

- No circuit breaker or backoff retries. Results are returned until an error is
    encountered, then execution stops and no further items are fetched (unless
    the goroutine is already in flight). I could be handling cancellation
    propagation better here, but I don't think I'm handling it in an
    unacceptable way.
- Fetching the todo items continues until an error is reached and then stops,
    continuing on to processing the responses. This means that we should in all
    cases end up with at least a partial result set, but not the entire 
    potential partial result set we could have ended up with.
- There are some hard coded values which I would generally have more
    configurable either through flags, arguments, or env vars, but this is also
    contextual.
