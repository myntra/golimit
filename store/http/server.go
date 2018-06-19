package http

import (
	"encoding/json"
	"github.com/myntra/golimit/store"
	"github.com/pressly/chi"
	log "github.com/sirupsen/logrus"
	"net"
	"net/http"
	"os"
	"runtime/pprof"
	"strconv"
	"strings"
)

type HttpServer struct {
	port          int
	hostname      string
	unixSocksFile string
	router        *chi.Mux
	store         *store.Store
}

func NewGoHttpServer(port int, hostname string, store *store.Store) *HttpServer {
	server := &HttpServer{port: port, hostname: hostname, store: store}
	server.router = chi.NewRouter()
	server.registerHttpHandlers()
	go http.ListenAndServe(":"+strconv.Itoa(port), server.router)
	log.Infof("http server started on port %d", port)
	return server
}

func NewGoHttpServerOnUnixSocket(sockFile string, store *store.Store) *HttpServer {
	server := &HttpServer{unixSocksFile: sockFile, store: store}
	server.router = chi.NewRouter()
	server.registerHttpHandlers()
	listener, err := net.Listen("unix", sockFile)
	if err != nil {
		log.Error(err)
		return nil
	}
	go http.Serve(listener, server.router)
	log.Infof("http server started on socket %s", sockFile)
	return server
}

func (s *HttpServer) registerHttpHandlers() {

	s.router.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("pong"))
	})

	s.router.Post("/incr", func(w http.ResponseWriter, r *http.Request) {
		var countStr, windowStr, thresholdStr, peakBust string

		key := r.URL.Query().Get("K")
		if r.URL.Query().Get("C") != "" {
			countStr = r.URL.Query().Get("C")
		} else {
			countStr = "1"
		}

		thresholdStr = r.URL.Query().Get("T")
		windowStr = r.URL.Query().Get("W")
		peakBust = r.URL.Query().Get("P")

		count, err := strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Count should be numeric", 400)
			return
		}

		threshold, err := strconv.Atoi(thresholdStr)
		if err != nil {
			http.Error(w, "Threshold should be numeric", 400)
			return
		}
		window, err := strconv.Atoi(windowStr)
		if err != nil {
			http.Error(w, "Window should be numeric", 400)
			return
		}
		peakaveraged, err := strconv.Atoi(peakBust)
		if err != nil {
			peakaveraged = 0
		}

		ret := s.store.Incr(key, int32(count), int32(threshold), int32(window), peakaveraged > 0)
		w.Header().Set("Content-Type", "application/json")
		w.Write((serialize(struct{ Block bool }{Block: ret})))

	})

	s.router.Post("/ratelimit", func(w http.ResponseWriter, r *http.Request) {
		var countStr string
		key := r.URL.Query().Get("K")
		if r.URL.Query().Get("C") != "" {
			countStr = r.URL.Query().Get("C")
		} else {
			countStr = "1"
		}

		count, err := strconv.Atoi(countStr)
		if err != nil {
			http.Error(w, "Count should be numeric", 400)
			return
		}
		ret := s.store.RateLimitGlobal(key, int32(count))
		w.Header().Set("Content-Type", "application/json")
		w.Write((serialize(struct{ Block bool }{Block: ret})))

	})
	s.router.Get("/rateall", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write((serialize(s.store.GetRateConfigAll())))
	})

	s.router.Get("/rate", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("K")
		if s.store.GetRateConfig(key) == nil {
			http.Error(w, "Key Not found", 404)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write((serialize(s.store.GetRateConfig(key))))
	})

	s.router.Get("/clusterinfo", func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Content-Type", "application/json")
		w.Write((serialize(s.store.GetClusterInfo())))
	})

	s.router.Put("/rate", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		secret := r.Header.Get("apisecret")
		if !s.store.IsAuthorised(secret) {
			http.Error(w, "Invalid Api Secret", 403)
			return
		}
		rate := struct {
			Key          string
			Window       int
			Limit        int
			PeakAveraged bool
		}{}
		decoder.Decode(&rate)
		log.Info("RateConfig Update request %+v", rate)
		if strings.TrimSpace(rate.Key) == "" || rate.Window < 1 || rate.Limit < 1 {
			http.Error(w, "Invalid Rate Config", 400)
			return
		}

		s.store.SetRateConfig(rate.Key, store.RateConfig{Limit: int32(rate.Limit), Window: int32(rate.Window),
			PeakAveraged: rate.PeakAveraged})

		w.Header().Set("Content-Type", "application/json")
		w.Write(serialize(struct{ Success bool }{Success: true}))
	})

	s.router.Get("/profilecpuenable", func(w http.ResponseWriter, r *http.Request) {

		secret := r.Header.Get("apisecret")
		if !s.store.IsAuthorised(secret) {
			http.Error(w, "Invalid Api Secret", 403)
			return
		}
		f, err := os.Create("golimitV3_cpu.pprof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		w.Header().Set("Content-Type", "application/json")
		w.Write(serialize(struct{ Success bool }{Success: true}))
	})
	s.router.Get("/profilecpudisable", func(w http.ResponseWriter, r *http.Request) {

		secret := r.Header.Get("apisecret")
		if !s.store.IsAuthorised(secret) {
			http.Error(w, "Invalid Api Secret", 403)
			return
		}
		pprof.StopCPUProfile()
		w.Header().Set("Content-Type", "application/json")
		w.Write(serialize(struct{ Success bool }{Success: true}))
	})

	s.router.Get("/memprofile", func(w http.ResponseWriter, r *http.Request) {

		secret := r.Header.Get("apisecret")
		if !s.store.IsAuthorised(secret) {
			http.Error(w, "Invalid Api Secret", 403)
			return
		}
		f, err := os.Create("golimitV3_mem.pprof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.WriteHeapProfile(f)
		f.Close()
		w.Header().Set("Content-Type", "application/json")
		w.Write(serialize(struct{ Success bool }{Success: true}))
	})

}

func serialize(obj interface{}) []byte {

	if str, err := json.Marshal(obj); err != nil {
		log.Errorf("Error serializing +%v", obj)
		return nil
	} else {
		return str
	}

}
