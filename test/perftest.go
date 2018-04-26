package main

import (

)
import (
	"time"
	"net/http"
	"math/rand"
	"io/ioutil"
	"strconv"
	"gopkg.in/alexcesaro/statsd.v2"

	"bytes"
	"github.com/sirupsen/logrus"
	"fmt"
)

func main()  {

	threads:=10
	keysCount:=10
	s,_:=statsd.New(statsd.Address("127.0.0.1:8125"),statsd.SampleRate(0.01))
	servers:=[]string{"127.0.0.1:8080","127.0.0.1:8081","127.0.0.1:8082"}
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport := *defaultTransportPointer // dereference it to get a copy of the struct that the pointer points to
	defaultTransport.MaxIdleConns = 1000
	defaultTransport.MaxIdleConnsPerHost = 1000

	http.DefaultClient = &http.Client{Transport: &defaultTransport}

	for i:=0;i<threads;i++{
		go func(){
			for{
				r:=rand.Intn(keysCount)
				t:=s.NewTiming()
				rs:=rand.Intn(len(servers))
				buf := bytes.NewBufferString("")
				resp,err:=http.Post("http://"+servers[rs]+"/incr?K=gomykey"+strconv.Itoa(r)+"&C=1&W=60&T=2000&P=1","application/json",buf)
				if(err==nil){
					ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					s.Count("success",1)
				}else{
					logrus.Error(err)
					s.Count("fail",1)
				}
				t.Send("gogoncr")

			}

		}()
	}


	time.Sleep(5*time.Minute)

}



