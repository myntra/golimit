package main

import (
	"bitbucket.org/myntra/golimitv3/store"
	"gopkg.in/yaml.v2"
	"flag"
	"io/ioutil"
	"os"
	"strings"
	"fmt"
	"time"
	"os/signal"
	"syscall"
	"bitbucket.org/myntra/golimitv3/store/http"
	log "github.com/sirupsen/logrus"
)



func main()  {


	configfileName := "golimit8080.yml"
	flag.StringVar(&configfileName, "config", configfileName, "Configuration File")
	loglevel:= "info"
	flag.StringVar(&loglevel, "loglevel", loglevel, "Log Level")
	flag.Parse()

	log.Infof(loglevel)
	switch loglevel {
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "error":
		log.SetLevel(log.ErrorLevel)
	default:
		log.SetLevel(log.InfoLevel)
	}
	log.Infof("Log Level %+v",log.GetLevel())
	customFormatter := new(log.TextFormatter)
	customFormatter.TimestampFormat = "2006-01-02 15:04:05:000"
	customFormatter.FullTimestamp = true
	log.SetFormatter(customFormatter)
	log.Info("Starting Go limiter")



	configObj := store.NewDefaultStoreConfig()
	{
		bytes, err := ioutil.ReadFile(configfileName)
		if err != nil {
			log.Error(err)
			return
		}
		err = yaml.Unmarshal(bytes, &configObj)
		if (err != nil) {
			panic(err)
		}
	}
	if(strings.TrimSpace(configObj.HostName)==""){
		hostname,err:=os.Hostname()
		if(err!=nil){
			log.Errorf("Not able to resolve hostname %+v",err)
			panic(err)
		}
		configObj.HostName=hostname
	}
	log.Infof("StoreConfig: %+v",configObj)

	store:=store.NewStore(configObj)

	http.NewGoHttpServer(configObj.HttpPort,configObj.HostName,store)




	var gracefulStop = make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGINT)
	sig := <-gracefulStop
	fmt.Printf("caught sig: %+v ", sig)
	store.Close()
	fmt.Println("Wait for 2 second to finish processing")
	time.Sleep(2*time.Second)
	os.Exit(0)

}



