## Golimit A Distributed Rate limiter
Golimit is  Uber [ringpop](https://github.com/uber/ringpop-go "ringpop") based distributed and decentralized rate 
limiter. It is horizontally scalable and is based on shard nothing architecture. Every node in system is capable of 
handling read and writes of counters.
It is designed to offer sub milliseconds latency to caller application. Recommended deployment topology is sidecar 
model.
Every golimit node keeps local and global counter for api counter and local value is synchronized with other nodes on 
configurable periodic interval or at defined threshold. 

#### Architecture
<b>Http server</b>
provides  http interface to increment counter against any arbitrary <b>Key</b> string. It also exposes admin api to 
manage global configurations.

<b>Store</b> encapsulates the data structure and functions to store, manage and replicate counters
Counter synchronisation is done in asynchronous way so the the caller application is never blocked for cluster sync.
Synchronizer module  keeps aggregating counters in memory and broadcast to other nodes on periodic intervals or when the 
counter has crossed threshold. the interval and threshold are configurable.

<b>StatsD Emitter</b> pushed metrics to configure statsd server.
 
![Block Diagram](https://github.com/myntra/golimit/blob/master/images/block.png?raw=true)

#### Deployment

![Block Diagram](https://github.com/myntra/golimit/blob/master/images/block.png?raw=true)
#### Installation
1. build

     `$ GOOS=linux GOARCH=amd64 go build` 
     
     for other platforms
     
     `GOOS=linux GOARCH=amd64 go install
      GOOS=darwin GOARCH=amd64 go install
      GOOS=windows GOARCH=amd64 go install
      GOOS=windows GOARCH=386 go install`
     
     
2. Configure
    
    Yml config
    ```yml
        clustername:  IDEA # Cluster Name
        tchannelport: 2345 # Ringpop T Channel Port
        seed: "127.0.0.1:2345" # Seed node of cluster
        unsyncedctrlimit: 5  # Unsynced counter limit
        unsyncedtimelimit: 60000 # unsynced timeout in ms
        httpport: 8080  # Http server port
        statsdenabled: true # Enable statsd 
        statsdhostport: "metrics.xyz.com:80"  # statsd host port
        statsdsamplerate: 1 # Statsd sampling rate
        apisecret: alpha # secret key to use admin apis
        
    ```
    
3. Run
    
    `$ ./golimitV3 --config=./golimitconfig.yml `
  
4. Use from Http Apis

    [on madwriter](http://madwriter.myntra.com/docs/golimitv3-v3)
    
5. Use as go module
   
   To install library:
   
   `go get bitbucket.org/myntra/golimitv3`
   
```go
package main
i
func main(){
   //instansiate config
    configObj := store.NewDefaultStoreConfig()
    //Update configuation as required
     
     
    //Instantiate Store object, Use single store instance in one application
    store:=store.NewStore(configObj)
     
     
     
     
    blocked:=store.Incr("key",1,1000,60,true) // Increment api
     
     
    if(blocked){
        //Blocked
    }
     
    //Ensure Store is closed on program exit
    store.Close()
}
```



#### Development

go to github.com/uber/tchannel-go/thrift/thrift-gen
and do go build to build thrift-gen

To generate thrift code
`{gopath}/src/github.com/uber/tchannel-go/thrift/thrift-gen/thrift-gen --generateThrift --outputDir gen-go --inputFile store/store.thrift`
