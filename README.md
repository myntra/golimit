

##Golimit A Distributed Rate limiter

####Key features

- Uses [ringpop](https://github.com/uber/ringpop-go "ringpop") hashring clustering library 
- Stores counter and keys in memory therefore there is no requirement of external storage.
- Seamlessly pushes blocked and allowed metrics against every key to statsd
- Support of persistent store for rate configs. So in case of cluster loss. Cluster starts with last saved config
- Security: Added security to prevent unauthorised modification of rate config

####Installation
1. build

     `$ GOOS=linux GOARCH=amd64 go build` 
     
     for other platforms
     
     `GOOS=linux GOARCH=amd64 go install
      GOOS=darwin GOARCH=amd64 go install
      GOOS=windows GOARCH=amd64 go install
      GOOS=windows GOARCH=386 go install`
     
     
2. Configure
    
    Yml config
    ```yaml 
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
    


####Development

go to github.com/uber/tchannel-go/thrift/thrift-gen
and do go build to build thrift-gen

To generate thrift code
`{gopath}/src/github.com/uber/tchannel-go/thrift/thrift-gen/thrift-gen --generateThrift --outputDir gen-go --inputFile store/store.thrift`
