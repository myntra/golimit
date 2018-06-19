package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/alexcesaro/statsd.v2"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func main() {

	var mode string
	flag.StringVar(&mode, "mode", "tcp", "mode tcp or unix")
	flag.Parse()

	if mode == "tcp" {
		tcpPerfTest()
	} else if mode == "sock" {
		socksPerfTest()
	}

}

func tcpPerfTest() {
	threads := 40
	keysCount := 10
	s, _ := statsd.New(statsd.Address("127.0.0.1:8125"), statsd.SampleRate(0.01))
	servers := []string{"127.0.0.1:8080", "127.0.0.1:8081", "127.0.0.1:8082"}
	defaultRoundTripper := http.DefaultTransport
	defaultTransportPointer, ok := defaultRoundTripper.(*http.Transport)
	if !ok {
		panic(fmt.Sprintf("defaultRoundTripper not an *http.Transport"))
	}
	defaultTransport := *defaultTransportPointer // dereference it to get a copy of the struct that the pointer points to
	defaultTransport.MaxIdleConns = 1000
	defaultTransport.MaxIdleConnsPerHost = 1000

	http.DefaultClient = &http.Client{Transport: &defaultTransport}

	count := int32(0)
	block := int32(0)
	allowed := int32(0)

	for i := 0; i < threads; i++ {
		go func() {
			for {
				r := rand.Intn(keysCount)
				t := s.NewTiming()
				rs := rand.Intn(len(servers))
				buf := bytes.NewBufferString("")
				resp, err := http.Post("http://"+servers[rs]+"/incr?K=gomykey"+strconv.Itoa(r)+"&C=1&W=60&T=2000&P=1", "application/json", buf)
				if err == nil {
					b, _ := ioutil.ReadAll(resp.Body)
					if strings.Contains(string(b), "true") {
						atomic.AddInt32(&block, 1)
					} else {
						atomic.AddInt32(&allowed, 1)
					}
					resp.Body.Close()
					s.Count("success", 1)
				} else {
					logrus.Error(err)
					s.Count("fail", 1)
				}
				t.Send("gogoncr")
				atomic.AddInt32(&count, 1)

			}

		}()
	}

	time.Sleep(5 * time.Minute)
	fmt.Printf("total : %d\n", count)
	fmt.Printf("blocked : %d\n", block)
	fmt.Printf("allowed : %d\n", allowed)
}

func socksPerfTest() {
	threads := 40
	keysCount := 10
	s, _ := statsd.New(statsd.Address("127.0.0.1:8125"), statsd.SampleRate(0.01))

	client1 := &http.Client{Transport: &http.Transport{
		IdleConnTimeout:     time.Second * 90,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/tmp/golimit1.sock")
		}}}

	client2 := &http.Client{Transport: &http.Transport{
		IdleConnTimeout:     time.Second * 90,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/tmp/golimit2.sock")
		}}}
	client3 := &http.Client{Transport: &http.Transport{
		IdleConnTimeout:     time.Second * 90,
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 1000,
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return net.Dial("unix", "/tmp/golimit.sock")
		}}}

	servers := []*http.Client{client1, client2, client3}

	count := int32(0)
	block := int32(0)
	allowed := int32(0)

	for i := 0; i < threads; i++ {
		go func() {
			for {
				r := rand.Intn(keysCount)
				t := s.NewTiming()
				rs := rand.Intn(len(servers))
				buf := bytes.NewBufferString("")
				resp, err := servers[rs].Post("http://golimit/incr?K=gomykey"+strconv.Itoa(r)+"&C=1&W=60&T=2000&P=1", "application/json", buf)
				if err == nil {
					b, _ := ioutil.ReadAll(resp.Body)
					if strings.Contains(string(b), "true") {
						atomic.AddInt32(&block, 1)
					} else {
						atomic.AddInt32(&allowed, 1)
					}
					resp.Body.Close()
					s.Count("success", 1)
				} else {
					logrus.Error(err)
					s.Count("fail", 1)
				}
				t.Send("gogoncr")
				atomic.AddInt32(&count, 1)
			}

		}()
	}

	time.Sleep(5 * time.Minute)
	fmt.Printf("total : %d\n", count)
	fmt.Printf("blocked : %d\n", block)
	fmt.Printf("allowed : %d\n", allowed)

}
