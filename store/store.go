package store

import (
	"encoding/json"
	"github.com/myntra/golimit/gen-go/com"
	"github.com/myntra/golimit/store/bucket"
	"github.com/myntra/golimit/store/event"
	log "github.com/sirupsen/logrus"
	"github.com/spaolacci/murmur3"
	"github.com/uber/ringpop-go"
	"github.com/uber/tchannel-go"
	"gopkg.in/alexcesaro/statsd.v2"
	"io/ioutil"
	"math"
	"os"
	"sync"
	"time"
)

type StoreConfig struct {
	ClusterName       string  `yaml:clustername,json:"clustername"`
	NodeId            string  `yaml:nodeid,json:"nodeid"`
	HostName          string  `yaml:hostname,json:"hostname"`
	TChannelPort      string  `yaml:tchannelport,json:"tchannelport"`
	Seed              string  `yaml:seednodes,json:"seednodes"`
	SyncBuffer        int     `yaml:syncbuffer,json:"syncbuffer"`
	Buckets           int     `yaml:buckets,json:"buckets"`
	StatsDEnabled     bool    `yaml:statsdenabled,json:"statsdenabled"`
	HttpPort          int     `yaml:httpport,json:"httpport"`
	UnsyncedCtrLimit  int32   `yaml:unsyncedctrlimit,json:"unsyncedctrlimit"`
	UnsyncedTimeLimit int     `yaml:unsyncedtimelimit,json:"unsyncedtimelimit"`
	StatsDHostPort    string  `yaml:statsdhostport,json:"statsdhostport"`
	StatsDSampleRate  float32 `yaml:statsdsamplerate,json:"statsdsamplerate"`
	StatsDBucket      string  `yaml:statsdbucket,json:"statsdbucket"`
	GcInterval        int     `yaml:gcinterval,json:"gcinterval"`
	ApiSecret         string  `yaml:apisecret,json:"apisecret"`
	GcGrace           int     `yaml:gcgrace,json:"gcgrace"`
}

func NewDefaultStoreConfig() StoreConfig {
	hostname := "UNKNOWN"
	hostname, _ = os.Hostname()
	storeCfg := StoreConfig{NodeId: hostname, Buckets: 1000, ClusterName: "golimittest", HostName: hostname, SyncBuffer: 100000,
		TChannelPort: "2345", Seed: "127.0.0.1:2345", HttpPort: 5000, UnsyncedCtrLimit: 10, UnsyncedTimeLimit: 30000, ApiSecret: "test",
		StatsDSampleRate: 0.01, GcInterval: 1800000, GcGrace: 1800000, StatsDEnabled: true, StatsDBucket: "golimittest", StatsDHostPort: "127.0.0.1:8125"}
	return storeCfg
}

type RateConfig struct {
	Window       int32 //in seconds
	Limit        int32
	PeakAveraged bool
}

type Store struct {
	sync.RWMutex
	config      *StoreConfig
	keyBucket   []*bucket.KeyBucket
	rateConfig  map[string]*RateConfig
	bucketMask  uint32
	syncChannel chan (*com.SyncCommand)
	tchannel    *tchannel.Channel
	ringpop     *ringpop.Ringpop
	nodeId      string
	syncPool    *sync.Pool
	statsd      *statsd.Client
}

func NewStore(config StoreConfig) *Store {
	log.Infof("Store: Config %+v", config)
	store := &Store{nodeId: config.NodeId,
		config:     &config,
		keyBucket:  make([]*bucket.KeyBucket, config.Buckets),
		rateConfig: make(map[string]*RateConfig),
		bucketMask: uint32(config.Buckets) - 1}
	store.nodeId = config.NodeId
	for i := 0; i < config.Buckets; i++ {
		store.keyBucket[i] = bucket.NewKeyBucket()
	}
	store.syncPool = &sync.Pool{
		New: func() interface{} {
			return &com.SyncCommand{}
		},
	}
	store.syncChannel = make(chan (*com.SyncCommand), store.config.SyncBuffer)
	store.initRingPop()
	store.registerClusterNode()
	store.LoadRateConfig()
	if store.config.StatsDEnabled {
		event.GetMgrInstance().RegisterHandler(event.KEYEVENT, store)
		statsd, err := statsd.New(statsd.Address(store.config.StatsDHostPort), statsd.SampleRate(store.config.StatsDSampleRate))
		if err == nil {
			store.statsd = statsd
		} else {

			log.Infof("Node: Error initialising statsd %s", err)
			store.config.StatsDEnabled = false
		}
	}
	store.startSyncWorker()
	store.gc()
	return store
}

func (s *Store) Handle(e event.Event) {
	kevent, ok := e.(*event.KeyEvent)
	if ok {
		log.Debugf("Sending stats %+v", e)
		if kevent.Allowed {
			s.statsd.Count(s.config.StatsDBucket+".golimit."+kevent.Key+".allowed", kevent.Count)
		} else {
			s.statsd.Count(s.config.StatsDBucket+".golimit."+kevent.Key+".blocked", kevent.Count)
		}
	}

}

func (s *Store) startSyncWorker() {
	go func() {
		log.Infof("Store: Sync Worker started")
		m := make(map[string]*com.SyncCommand)
		send := false
		close := false
		timer := time.NewTimer(time.Millisecond * time.Duration(s.config.UnsyncedTimeLimit))
		for {
			send = false
			close = false
			select {
			case sCmd, ok := <-s.syncChannel:
				if ok {
					key := sCmd.Key
					send = sCmd.Force
					log.Debugf("Store: Sync Worker Got to Sync")
					if m[sCmd.Key] == nil {
						log.Debugf("Store: Sync Worker Key Is New Key: %s", sCmd.Key)
						m[sCmd.Key] = sCmd
					} else {
						log.Debugf("Store: Sync Worker Key Is Old Key: %s", sCmd.Key)
						if sCmd.Expiry == m[sCmd.Key].Expiry {
							log.Debugf("Store: Sync Worker Expiry is Same Expiry: %d", sCmd.Expiry)
							m[sCmd.Key].Count += sCmd.Count
							s.syncPool.Put(sCmd)
						} else {
							log.Debugf("Store: Sync Worker Expiry is Different Expiry: %d", sCmd.Expiry)
							s.syncPool.Put(m[sCmd.Key])
							m[sCmd.Key] = sCmd
						}
					}
					if m[key].Count > s.config.UnsyncedCtrLimit || send {
						log.Debugf("Store: Sync Worker Unsynced Limit reached")
						send = true
					}
				} else {
					log.Infof("Store: Sync Worker Channel Closed")
					send = true
					close = true
				}

			case <-timer.C:
				log.Debugf("Store: Sync Worker Unsynced timeout happened")
				send = true
			}
			if send && len(m) > 0 {
				slice := make([]*com.SyncCommand, len(m))
				i := 0
				for k, v := range m {
					slice[i] = v
					i++
					delete(m, k)
				}
				log.Debugf("Store: Sync Worker sending to sysc %+v", slice)
				s.sendSyncCommandToNodes(slice)
				for _, v := range slice {
					s.syncPool.Put(v)
				}
			}
			timer.Stop()
			timer.Reset(time.Millisecond * time.Duration(s.config.UnsyncedTimeLimit))
			if close {
				log.Debugf("Store: Sync Worker Closing")
				break
			}
		}
	}()
}
func (s *Store) SetRateConfig(key string, rateConfig RateConfig) {
	s.rateConfig[key] = &rateConfig
	s.SaveRateConfig()
	s.sendSyncRateConfigToNodes(key, &rateConfig)
}

func (s *Store) GetRateConfig(key string) *RateConfig {
	return s.rateConfig[key]
}

func (s *Store) GetRateConfigAll() map[string]*RateConfig {
	return s.rateConfig
}

func (s *Store) getKeyBucket(key string) *bucket.KeyBucket {
	return s.keyBucket[murmur3.Sum32([]byte(key))&s.bucketMask]
}

func (s *Store) RateLimitGlobal(key string, count int32) bool {
	//If not ratelimit config defined Allow
	rateConfig := s.rateConfig[key]
	blocked := false
	if rateConfig != nil {
		blocked = s.Incr(key, count, rateConfig.Limit, rateConfig.Window, rateConfig.PeakAveraged)
	}
	return blocked
}

func (s *Store) Incr(key string, count int32, threshold int32, window int32, peakaveraged bool) bool {
	log.Debugf("Store: incrementing (key, count, limit, window,peakaverageder) (%s, %d, %d, %d, %t)", key, count, threshold, window, peakaveraged)
	if peakaveraged && window != 1 {
		threshold = int32(math.Ceil(float64(threshold) / float64(window)))
		window = 1
	}
	blocked, expires, dosync := s.getKeyBucket(key).Incr(key, count, threshold, window)
	log.Debugf("Store: Increment result blocked= %t , expires = %d, dosync:: %t", blocked, expires, dosync)
	if !blocked || dosync {
		sCmd := s.syncPool.Get().(*com.SyncCommand)
		sCmd.Key = key
		sCmd.Count = count
		sCmd.Expiry = expires
		sCmd.Force = dosync
		s.syncChannel <- sCmd
	}
	if s.config.StatsDEnabled {
		e := event.GetMgrInstance().GetPool(event.KEYEVENT).Get().(*event.KeyEvent)
		e.Count = count
		e.Key = key
		e.Allowed = !blocked
		e.Time = time.Now().UnixNano()
		event.GetMgrInstance().Publish(e)
	}

	return blocked
}

func (s *Store) gc() {
	go func() {
		for {
			time.Sleep(time.Duration(s.config.GcInterval) * time.Millisecond)
			log.Debugf("Store: GC stated")
			for _, b := range s.keyBucket {
				b.Lock()
				for k, e := range b.Lookup() {
					log.Debugf("Store: GC checking key %s expiry %d", k, e.Expiry())
					if e.Expiry()+(time.Duration(s.config.GcGrace)*time.Millisecond).Nanoseconds() < time.Now().UnixNano() {
						e.Lock()
						if e.Expired() {
							delete(b.Lookup(), k)
							log.Debugf("Store: GC deleting key %s", k)
						}
						e.Unlock()
					}
				}
				b.Unlock()
			}
			log.Debugf("Store: GC done")
		}
	}()
}

func (s *Store) Close() {
	s.ringpop.Destroy()
	s.tchannel.Close()

}

func (s *Store) IsAuthorised(secret string) bool {
	return secret == s.config.ApiSecret
}

func (s *Store) StatsDClient() *statsd.Client {
	return s.statsd
}

func (s *Store) SaveRateConfig() {
	json, _ := json.Marshal(s.rateConfig)
	ioutil.WriteFile("rateconfig.json", json, 0644)
}

func (s *Store) LoadRateConfig() {
	bytes, err := ioutil.ReadFile("rateconfig.json")
	if err == nil {
		json.Unmarshal(bytes, &s.rateConfig)
	}
}
