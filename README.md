# shutdown

[![Go Reference](https://pkg.go.dev/badge/github.com/jsocol/shutdown.svg)](https://pkg.go.dev/github.com/jsocol/shutdown)

A small Go library to deduplicate the signal-handling and timeout logic around
graceful shutdowns. Import it into your main() method like so:

```go
import (
    "context"
    "fmt"
    "net/http"
    "github.com/jsocol/shutdown"
)

func main() {
    server := &http.Server{}

    shutdown.Listen(server.Shutdown)

    if err := server.ListenAndServe(); err != nil {
        fmt.Println("start up error", err)
    }
}
```

You can give it any number of shutdown tasks (`func(context.Context) error`) to
execute, in case you have multiple goroutines that can shutdown independently.

By default, there is a timeout of 10 seconds for graceful shutdown, after which
the server will be stopped even if the shutdown tasks have not completed. You
can change this default by calling `shutdown.SetTimeout(d time.Duration)`. The
graceful shutdown can be abandoned by sending another interrupt signal.

If any of the following happen, `Listen()` will call `os.Exit` with a specific
exit status:

- The timeout is exceeded. (4)
- A second interrupt is received. (3)
- Any task returns an error. (2)

In other cases, `Listen()`'s internal goroutines simply exit, leaving the larger
program to determine the exit status, if any.

`shutdown` uses `log/slog` for output, so should use the configured logger.
