## Golimit A Distributed Rate limiter
Golimit is  Uber [ringpop](https://github.com/uber/ringpop-go "ringpop") based distributed and decentralized rate 
limiter. It is horizontally scalable and is based on shard nothing architecture. Every node in system is capable of 
handling read and writes of counters.
It is designed to offer sub milliseconds latency to caller application. Recommended deployment topology is sidecar 
model.
Every golimit node keeps local and global counter for api counter and local value is synchronized with other nodes on 
configurable periodic interval or at defined threshold. 

### Architecture
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
Suggested deployment model is to have golimit installed as sidecar. This will ensure application latency to 
sub milliseconds level.
For Go applications golimit can be directly integrated as a module,the way of using golimit as module is  explained 
later in document. Using as module takes away the pain of deployment and maintenance.

![Block Diagram](https://github.com/myntra/golimit/blob/master/images/deployment.png?raw=true)

#### Installation

1. Build

    ```
    $ GOOS=linux GOARCH=amd64 go build  //Linux
    
    $ GOOS=darwin GOARCH=amd64 go build //OSX 
    
    $ GOOS=windows GOARCH=amd64 go build //Windows
    ```
     
2. Configure
    
    Yml config
    ```yaml
        clustername:  MyGolimitCluster # Cluster Name
        tchannelport: 2345 # Ringpop T Channel Port
        seed: "127.0.0.1:2345" # Seed node of cluster
        unsyncedctrlimit: 5  # Unsynced counter limit
        unsyncedtimelimit: 60000 # unsynced timeout in ms
        httpport: 8080  # Http server port
        statsdenabled: true # Enable statsd 
        statsdhostport: "metrics.xyz.com:80"  # statsd host port
        statsdsamplerate: 1 # Statsd sampling rate
        apisecret: alpha # secret key to use admin apis
        hostname: "127.0.0.1"
        
    ```
    Note: Ensure the seed node is always reachable.
    
3. Run
    
    ```
    $ ./golimitV3 --config=./golimitconfig.yml
    ```
  
4. Use from Http Apis

    [on madwriter](http://madwriter.myntra.com/docs/golimitv3-v3)
    
    | Param | Description|
    |:-------:|:-----------|
    |K      | Key a string, against this the counters are calculated|
    |C|Count in number, numbers to increment in one api call, defaults to 1|
    |W|Window in seconds, time window for which provided threshold is applicable|
    |T|Threshold in number|
    |P|PeakAveraged 0 or 1, if P=1 the provided rate limit is transposed to per second limit and then applied|
    
    * INCR Request
       
       Application passes Key threshold and window and reply is block or not.
       This rate limiting is application driven as application has to pass all rate configuration in every call 
       In following example second curl within 10 seconds gives back blocked =true
        
        ```
        $  curl -X POST "http://localhost:8080/incr?K=abc&T=1&W=10" 
        
        {"Block":false}
        
        $  curl -X POST "http://localhost:8080/incr?K=abc&T=1&W=10" 
            
        {"Block":true}
        ```
    * Create/Update global rate configuration, this is for rate limiting which is golimit cluster driven
        
        ```
        $ curl -X PUT  "http://localhost:8080/rate" -d '{"Window":60,"Limit":5,"Key":"a","PeakAveraged":false}' -H "apisecret: alpha" 
           
        {"Success":true}
        
        $ curl -X POST  "http://localhost:8080/ratelimit?K=a" 
    
        {"Block":false}
        
        # after 5 times
        # {"Block":true}
        
        ```
    * Ratelimit Request
    
        ```
         $ curl -X POST  "http://localhost:8080/ratelimit?K=a" 
         
         {"Block":false}
         
        ```
        
    * Get All defined Rate Config
    
        ```
        $ curl  "http://localhost:8080/rateall" 
        
        {"a":{"Window":60,"Limit":5,"PeakAveraged":false}}
        
        ```
    
    * Get a specific Rate Config
    
        ```
        $ curl  "http://localhost:8080/rate?K=a" 
    
        {"Window":60,"Limit":5,"PeakAveraged":false}%
        
        ```
    * Get Cluster Info
        ```
        $ curl  "http://localhost:8080/clusterinfo" 
        
        {"Whoami":"127.0.0.1:2345","Ready":true,"Uptime":9223372036854775807,"Members":["127.0.0.1:2345"]}   

        ```

5. Use as Go module
   
    If application is in golang. Golimit can be used as module directly instead of deploying as separate process.
    
    
    To install library:
    
    ```go get github.com/myntra/golimit```
   
    ```go
    package main
    import("github.com/myntra/golimit/store")
    func main(){
      
         
        //Instantiate Store object, Use single store instance in one application
        store:=store.NewStore()
         
         
         
         
        blocked:=store.Incr("key",1,1000,60,true) // Increment api
         
         
        if(blocked){
            //Blocked
        }
         
        //Ensure Store is closed on program exit
        store.Close()
    }
    ```
