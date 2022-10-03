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

<img src="https://img.shields.io/badge/Version-1.2.0-informational?style=for-the-badge&logo=none" alt="version" />
</p>


## Installation
```bash
go get github.com/snapp-incubator/qsse
```

## Basic Usage

### Server
```Go
// Server
package main

import "github.com/snapp-incubator/qsse"

var (
    people = []Person{...}
    accounts = []Account{...}
) 

func main() {
	topics := []string{"people", "account"}

	server, err := qsse.NewServer("localhost:4242", topics, nil)
	if err != nil {
		panic(err)
	}

	// publish events
	server.Publish("people", people[0])
	server.Publish("accounts", accounts[0])
	// ...

	// more code
}
```

### Client
```Go
package main

import "github.com/snapp-incubator/qsse"

func main() {
    topics := []string{"people", "account"}
    
    client, err := qsse.NewClient("localhost:4242", topics, nil)
    if err != nil {
        panic(err)
    }
	
    client.SetEventHandler("people", func(data []byte) {
        // handle data on topic
    })
	
    client.SetErrorHandler(func(code int, data map[string]any) { 
        // handle different error
    })
    
	// more code
}
```

## Security
By default, all the clients are accepted. Use Authentication to check is new clients are valid and use Authorization to check client access on each topic.
```Go
// authentication

// func
server.SetAuthenticatorFunc(func(token string) bool {

})
// interface
server.SetAuthenticator()

// authorization on each topic

// func
server.SetAuthorizerFunc(func(token, topic string) bool {

})
// interface
server.SetAuthorizer()
```

## Topic Patterns
topics can be separated by `.` logically. also `*` can be used as wildcard placeholder. for example these are valid topics 
- `ride`
- `ride.*.start`
- `ride.passenger.*` 

**Note**: Putting `*` at the end of topic will publish or subscribe to every topic that start with `*` prefix. For example `ride.passenger.*` is equivalent of subscribing to `ride.passenger.start`, `ride.passenger.account.name`, and so on.

## Server Configurations
| config                                 	 | description                                                                                   	| default                        	|
|------------------------------------------|-----------------------------------------------------------------------------------------------	|--------------------------------	|
| Metric.namespace, <br>Metric.subsystem 	 | namespace and subsystem parameters of the Prometheus metrics                                  	| "qsse",<br>"qsse"              	|
| TLSConfig                              	 | TLS config of server                                                                          	| qsse.GetDefaultTLSConfig<br>() 	|
| Worker.CleaningInterval                	 | interval between cleaning idle clients                                                        	| 10 sec                         	|
| Worker.ClientAcceptorCount             	 | number of Goroutine accepting new clients                                                     	| 1                              	|
| Worker.ClientAcceptorQueueSize         	 | queue size of client acceptors. (this is usually equal to `clientAcceptorCount`)              	| 1                              	|
| Worker.EventDistributorCount           	 | number of concurrent goroutine distributing events to subscribers for each EventSource[topic] 	| 1                              	|
| Worker.EventDistributorQueueSize       	 | queue size of event distribution work                                                         	| 10                             	|

## Client Configurations
| config                        	| description                                                                                          	| default                 	|
|-------------------------------	|------------------------------------------------------------------------------------------------------	|-------------------------	|
| token                         	| token that will be send to server on the initial connection to verify the client.                    	| ""                      	|
| TLSConfig                     	| TLS config of client                                                                                 	| qsse.GetSimpleTLS<br>() 	|
| ReconnectPolicy.Retry         	| bool that indicate if client should retry connection if couldn't connect to server on the first try. 	| false                   	|
| ReconnectPolicy.RetryTimes    	| number of reconnect times to connect.                                                                	| 5                       	|
| ReconnectPolicy.RetryInterval 	| interval between reconnecting to server                                                              	| 5 sec                   	|

## Examples
- [Simple Client & Server](examples/simple)
- [Topic Patterns](examples/topic-pattern)
- [Multiple Generators](examples/multiple-generators)