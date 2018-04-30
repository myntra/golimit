package store

import (
	"encoding/json"
	"github.com/myntra/golimit/gen-go/com"
	"github.com/myntra/golimit/store/bucket"
	"github.com/myntra/golimit/store/clock"
	"github.com/myntra/golimit/store/event"
	log "github.com/sirupsen/logrus"
	"github.com/spaolacci/murmur3"
	"github.com/uber/ringpop-go"
	"github.com/uber/tchannel-go"
	"gopkg.in/alexcesaro/statsd.v2"
	"io/ioutil"
	"math"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

var hostname string

func init() {
	hostname, _ = os.Hostname()
}

type options struct {
	clusterName       string
	nodeId            string
	tchannelport      string
	seed              string
	syncBuffer        int
	buckets           int
	statsDEnabled     bool
	httpPort          string
	unsyncedCtrLimit  int32
	unsyncedTimeLimit int
	statsDHostPort    string
	statsDSampleRate  float32
	statsDBucket      string
	gcInterval        int
	apiSecret         string
	gcGrace           int
	tchannel          *tchannel.Channel
	ringpop           *ringpop.Ringpop
	clock             clock.Clock
}

var defaultOptions = options{
	clusterName:       "golimit",
	nodeId:            "golimit" + strconv.Itoa(rand.Int()),
	tchannelport:      "2479",
	seed:              hostname + ":2479",
	syncBuffer:        100000,
	buckets:           1000,
	statsDEnabled:     false,
	statsDHostPort:    "",
	statsDSampleRate:  .0001,
	httpPort:          "7289",
	unsyncedCtrLimit:  10,
	unsyncedTimeLimit: 30000,
	apiSecret:         "pingpong",
	statsDBucket:      "golimit",
	gcInterval:        1800000,
	gcGrace:           1800000,
	tchannel:          nil,
	ringpop:           nil,
	clock:             &clock.RealClock{},
}

type Option func(*options)

func WithRingpop(ringpop *ringpop.Ringpop) Option {
	return func(o *options) {
		o.ringpop = ringpop
	}
}

func WithTChannel(tchannel *tchannel.Channel) Option {
	return func(o *options) {
		o.tchannel = tchannel
	}
}

func WithClock(clock clock.Clock) Option {
	return func(o *options) {
		o.clock = clock
	}
}

func WithApiSecret(secret string) Option {
	return func(o *options) {
		o.apiSecret = secret
	}
}

func WithGcGrace(gcgrace int) Option {
	return func(o *options) {
		o.gcGrace = gcgrace
	}
}

func WithGcInterval(gcinterval int) Option {
	return func(o *options) {
		o.gcInterval = gcinterval
	}
}

func WithStatsDBucket(bucket string) Option {
	return func(o *options) {
		o.statsDBucket = bucket
	}
}

func WithStatsDSampleRate(rate float32) Option {
	return func(o *options) {
		o.statsDSampleRate = rate
	}
}

func WithStatsDHostPort(hostport string) Option {
	return func(o *options) {
		o.statsDHostPort = hostport
	}
}

func WithUnsyncedTimeLimit(limit int) Option {
	return func(o *options) {
		o.unsyncedTimeLimit = limit
	}
}

func WithUnsyncedCtrLimit(limit int32) Option {
	return func(o *options) {
		o.unsyncedCtrLimit = limit
	}
}

func WithHttpPort(port string) Option {
	return func(o *options) {
		o.httpPort = port
	}
}

func WithStatDEnabled(enable bool) Option {
	return func(o *options) {
		o.statsDEnabled = enable
	}
}

func WithBucketSize(buckets int) Option {
	return func(o *options) {
		o.buckets = buckets
	}
}

func WithSyncBuffer(bufferSize int) Option {
	return func(o *options) {
		o.syncBuffer = bufferSize
	}
}

func WithSeed(seeds string) Option {
	return func(o *options) {
		o.seed = seeds
	}
}

func WithTChannelPort(port string) Option {
	return func(o *options) {
		o.tchannelport = port
		o.nodeId = hostname + ":" + o.tchannelport
	}
}

func WithHostName(hostName string) Option {
	return func(o *options) {
		hostname = hostName
		o.nodeId = hostname + ":" + o.tchannelport
	}
}

func WithClusterName(name string) Option {
	return func(o *options) {
		o.clusterName = name
		o.nodeId = hostname + ":" + o.tchannelport
	}
}

func WithNodeId(nodeId string) Option {
	return func(o *options) {
		o.nodeId = nodeId
	}
}

type RateConfig struct {
	Window       int32 //in seconds
	Limit        int32
	PeakAveraged bool
}

type Store struct {
	sync.RWMutex
	opts        options
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

func NewStore(opt ...Option) *Store {

	opts := defaultOptions
	for _, o := range opt {
		o(&opts)
	}

	log.Infof("Store: Options %+v", opts)

	store := &Store{nodeId: opts.nodeId,
		opts:       opts,
		keyBucket:  make([]*bucket.KeyBucket, opts.buckets),
		rateConfig: make(map[string]*RateConfig),
		bucketMask: uint32(opts.buckets) - 1}
	store.nodeId = opts.nodeId
	for i := 0; i < opts.buckets; i++ {
		store.keyBucket[i] = bucket.NewKeyBucket(opts.clock)
	}
	store.syncPool = &sync.Pool{
		New: func() interface{} {
			return &com.SyncCommand{}
		},
	}
	store.syncChannel = make(chan (*com.SyncCommand), store.opts.syncBuffer)
	store.initRingPop()
	store.registerClusterNode()
	store.LoadRateConfig()
	if store.opts.statsDEnabled {
		event.GetMgrInstance().RegisterHandler(event.KEYEVENT, store)
		statsd, err := statsd.New(statsd.Address(store.opts.statsDHostPort), statsd.SampleRate(store.opts.statsDSampleRate))
		if err == nil {
			store.statsd = statsd
		} else {

			log.Infof("Node: Error initialising statsd %s", err)
			store.opts.statsDEnabled = false
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
			s.statsd.Count(s.opts.statsDBucket+".golimit."+kevent.Key+".allowed", kevent.Count)
		} else {
			s.statsd.Count(s.opts.statsDBucket+".golimit."+kevent.Key+".blocked", kevent.Count)
		}
	}

}

func (s *Store) startSyncWorker() {
	go func() {
		log.Infof("Store: Sync Worker started")
		m := make(map[string]*com.SyncCommand)
		send := false
		close := false
		timer := time.NewTimer(time.Millisecond * time.Duration(s.opts.unsyncedTimeLimit))
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
					if m[key].Count > s.opts.unsyncedCtrLimit || send {
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
			timer.Reset(time.Millisecond * time.Duration(s.opts.unsyncedTimeLimit))
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
	//If not ratelimit opts defined Allow
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
	if s.opts.statsDEnabled {
		e := event.GetMgrInstance().GetPool(event.KEYEVENT).Get().(*event.KeyEvent)
		e.Count = count
		e.Key = key
		e.Allowed = !blocked
		e.Time = s.opts.clock.Now().UnixNano()
		event.GetMgrInstance().Publish(e)
	}

	return blocked
}

func (s *Store) gc() {
	go func() {
		for {
			time.Sleep(time.Duration(s.opts.gcInterval) * time.Millisecond)
			log.Debugf(s.nodeId + " Store: GC stated")
			for _, b := range s.keyBucket {
				b.Lock()
				for k, e := range b.Lookup() {
					log.Debugf(s.nodeId+" Store: GC checking key %s expiry %d", k, e.Expiry())
					if e.Expiry()+(time.Duration(s.opts.gcGrace)*time.Millisecond).Nanoseconds() < s.opts.clock.Now().UnixNano() {
						e.Lock()
						if e.Expired() {
							delete(b.Lookup(), k)
							log.Debugf(s.nodeId+" Store: GC deleting key %s", k)
						}
						e.Unlock()
					}
				}
				b.Unlock()
			}
			log.Debugf(s.nodeId + " Store: GC done")
		}
	}()
}

func (s *Store) Close() {
	s.ringpop.Destroy()
	s.tchannel.Close()

}

func (s *Store) IsAuthorised(secret string) bool {
	return secret == s.opts.apiSecret
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
