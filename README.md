# go-retry

![](https://github.com/vimeo/go-retry/workflows/Go/badge.svg)
[![GoDoc](https://godoc.org/github.com/vimeo/go-retry?status.svg)](https://godoc.org/github.com/vimeo/go-retry)

`go-retry` is a package that helps facilitate retry logic with jittered
exponential backoff.  It provides a convenient interface for configuring various
parameters.  See below for more information.

## Example

```go
func makeNetworkCall(ctx context.Context) {
    defaultBackoff := retry.DefaultBackoff.Clone()

    // try at most 5 times
    retry.Retry(ctx, defaultBackoff, 5, func(ctx context.Context) error {
        response, err := http.Get("https://my.favorite.service")
        if err != nil {
            return err
        }
        // do something with response...
    })
}
```

Copyright Vimeo.