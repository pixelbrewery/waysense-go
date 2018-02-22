## Overview

Package `waysense` provides a Go waysense client. See https://waysense.io for more information.

## Get the code

    $ go get github.com/pixelbrewery/waysense-go/waysense

## Usage
```go
// Create the client
c, err := waysense.New("127.0.0.1:8100", "key-123", "secret-123")
if err != nil {
    log.Fatal(err)
}

// Do some metrics!
err = c.SendGeoHash("thing-id-1", "GFJR1")
```