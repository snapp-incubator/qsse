<h1 align="center">
  <img alt="QSSE logo" src="assets/icon.png" width="500px"/><br/>
  SSE Over QUIC
</h1>
<p align="center">Implementation of Server Sent Events by QUIC. A faster replacement for traditional SSE over HTTP/2.</p>

<p align="center">
<a href="https://pkg.go.dev/github.com/snapp-incubator/qsse/v3?tab=doc"target="_blank">
    <img src="https://img.shields.io/badge/Go-1.18+-00ADD8?style=for-the-badge&logo=go" alt="go version" />
</a>&nbsp;
<img src="https://img.shields.io/badge/license-apache_2.0-red?style=for-the-badge&logo=none" alt="license" />

<img src="https://img.shields.io/badge/Version-1.1.0-informational?style=for-the-badge&logo=none" alt="version" />
</p>


## Installation
```bash
go get github.com/snapp-incubator/qsse
```

## Basic Usage
```Go
// Client

import "github.com/snapp-incubator/qsse"

func main() {
    config := &qsse.ClientConfig{
        Token:     "secret",
        TLSConfig: nil,
    }
    topics := []string{"firstnames", "lastnames"}
    
    _, err := qsse.NewClient("localhost:4242", topics, config)
    if err != nil {
        panic(err)
    }
    
    select {}
}


```

```Go
// Server

import (
	"github.com/snapp-incubator/qsse"
	"log"
	"math/rand"
	"time"
)

var firstNames = []string{...}

var lastNames = []string{...}

func main() {
	authenticateFunc := func(token string) bool {
		log.Printf("Authenticating token: %s", token)
		return token == "secret"
	}

	topics := []string{"firstnames", "lastnames"}

	server, err := qsse.NewServer("localhost:4242", qsse.GetDefaultTLSConfig(), topics)
	if err != nil {
		panic(err)
	}
	server.SetAuthentication(authenticateFunc)

	go func() {
		for {
            if rand.NormFloat64() > 0.5 {
                server.Publish("firstnames", RandomItem(firstNames))
            } else {
                server.Publish("lastnames", RandomItem(lastNames))
            }
			<-time.After(2 * time.Second)
		}
	}()

	select {}
}

func RandomItem(items []string) []byte {
    return []byte(items[rand.Intn(len(items))])
}

```
