package store

import (
	"github.com/myntra/golimit/store/clock"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/uber/ringpop-go"
	"github.com/uber/ringpop-go/discovery/statichosts"
	"github.com/uber/ringpop-go/swim"
	"github.com/uber/tchannel-go"
	"log"
	"os"
	"testing"
	"time"
)

func TestStoreCluster_Ctr_Limit_test(t *testing.T) {
	os.Remove("rateconfig.json")
	configStore1 := defaultOptions
	configStore1.unsyncedTimeLimit = 1000

	configStore2 := defaultOptions
	configStore2.tchannelport = "2346"
	configStore2.unsyncedTimeLimit = 1000
	configStore2.seed = hostname + ":" + configStore1.tchannelport

	configStore3 := defaultOptions
	configStore3.tchannelport = "2347"
	configStore3.unsyncedTimeLimit = 1000
	configStore3.seed = hostname + ":" + configStore1.tchannelport

	configStore4 := defaultOptions
	configStore4.tchannelport = "2348"
	configStore4.unsyncedTimeLimit = 1000
	configStore4.seed = hostname + ":" + configStore1.tchannelport
	_clock := &clock.UnRealClock{}
	store1 := NewStore(WithTChannelPort(configStore1.tchannelport),
		WithSeed(configStore1.seed), WithClock(_clock),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit), WithUnsyncedCtrLimit(10))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport),
		WithSeed(configStore2.seed), WithClock(_clock),
		WithUnsyncedTimeLimit(configStore2.unsyncedTimeLimit), WithUnsyncedCtrLimit(10))
	store3 := NewStore(WithTChannelPort(configStore3.tchannelport),
		WithSeed(configStore3.seed), WithClock(_clock),
		WithUnsyncedTimeLimit(configStore3.unsyncedTimeLimit), WithUnsyncedCtrLimit(10))
	store4 := NewStore(WithTChannelPort(configStore4.tchannelport),
		WithSeed(configStore4.seed), WithClock(_clock),
		WithUnsyncedTimeLimit(configStore4.unsyncedTimeLimit), WithUnsyncedCtrLimit(10))

	blocked := store1.Incr("one", 1, 10, 60, false)
	assert.False(t, blocked)
	blocked = store2.Incr("one", 1, 10, 60, false)
	assert.False(t, blocked)
	blocked = store3.Incr("one", 1, 10, 60, false)
	assert.False(t, blocked)
	blocked = store4.Incr("one", 1, 10, 60, false)
	assert.False(t, blocked)

	for i := 0; i < 9; i++ {
		blocked = store1.Incr("one", 1, 10, 60, false)
		assert.False(t, blocked)
	}
	blocked = store1.Incr("one", 1, 10, 60, false)
	assert.True(t, blocked)

	for i := 0; i < 100; i++ {
		blocked := store1.RateLimitGlobal("three", 1)
		assert.False(t, blocked)
	}

	store1.SetRateConfig("three", RateConfig{Limit: 10, Window: 60})
	time.Sleep(time.Second) //Sleep for server to sync config
	_clock.AddSeconds(1)

	for i := 0; i < 10; i++ {
		blocked = store2.RateLimitGlobal("three", 1)
		assert.False(t, blocked)
	}

	blocked = store2.RateLimitGlobal("three", 1)
	assert.True(t, blocked)

}

func TestStoreCluster_Timeout_Limit_test(t *testing.T) {

	configStore1 := defaultOptions
	configStore1.tchannelport = "2355"
	configStore1.unsyncedTimeLimit = 1000
	configStore1.seed = hostname + ":" + configStore1.tchannelport

	configStore2 := defaultOptions
	configStore2.tchannelport = "2356"
	configStore2.unsyncedTimeLimit = 1000
	configStore2.seed = hostname + ":" + configStore1.tchannelport

	configStore3 := defaultOptions
	configStore3.tchannelport = "2357"
	configStore3.unsyncedTimeLimit = 1000
	configStore3.seed = hostname + ":" + configStore1.tchannelport

	configStore4 := defaultOptions
	configStore4.tchannelport = "2358"
	configStore4.unsyncedTimeLimit = 1000
	configStore4.seed = hostname + ":" + configStore1.tchannelport
	_clock := &clock.UnRealClock{}
	store1 := NewStore(WithTChannelPort(configStore1.tchannelport),
		WithSeed(configStore1.seed), WithClock(_clock), WithUnsyncedTimeLimit(1000),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport),
		WithSeed(configStore2.seed), WithClock(_clock), WithUnsyncedTimeLimit(1000),
		WithUnsyncedTimeLimit(configStore2.unsyncedTimeLimit))
	store3 := NewStore(WithTChannelPort(configStore3.tchannelport),
		WithSeed(configStore3.seed), WithClock(_clock), WithUnsyncedTimeLimit(1000),
		WithUnsyncedTimeLimit(configStore3.unsyncedTimeLimit))
	store4 := NewStore(WithTChannelPort(configStore4.tchannelport),
		WithSeed(configStore4.seed), WithClock(_clock), WithUnsyncedTimeLimit(1000),
		WithUnsyncedTimeLimit(configStore4.unsyncedTimeLimit))

	blocked := store1.Incr("two", 1, 10, 60, false)
	assert.False(t, blocked)
	blocked = store2.Incr("two", 1, 10, 60, false)
	assert.False(t, blocked)
	blocked = store3.Incr("two", 1, 10, 60, false)
	assert.False(t, blocked)
	blocked = store4.Incr("two", 1, 10, 60, false)
	assert.False(t, blocked)

	for i := 0; i < 6; i++ {
		blocked = store1.Incr("two", 1, 10, 60, false)
		assert.False(t, blocked)
	}
	time.Sleep(1080 * time.Millisecond)
	logrus.Infof("%+v", store1)
	blocked = store1.Incr("two", 1, 10, 60, false)
	assert.True(t, blocked)

}

func TestStoreCluster_Node_Join(t *testing.T) {

	configStore1 := defaultOptions
	configStore1.tchannelport = "2365"
	configStore1.unsyncedTimeLimit = 1000
	configStore1.seed = hostname + ":" + configStore1.tchannelport

	configStore2 := defaultOptions
	configStore2.tchannelport = "2366"
	configStore2.unsyncedTimeLimit = 1000
	configStore2.seed = hostname + ":" + configStore1.tchannelport

	configStore3 := defaultOptions
	configStore3.tchannelport = "2367"
	configStore3.unsyncedTimeLimit = 1000
	configStore3.seed = hostname + ":" + configStore1.tchannelport

	configStore4 := defaultOptions
	configStore4.tchannelport = "2368"
	configStore4.unsyncedTimeLimit = 1000
	configStore4.seed = hostname + ":" + configStore1.tchannelport

	_clock := &clock.UnRealClock{}
	store1 := NewStore(WithTChannelPort(configStore1.tchannelport), WithClock(_clock),
		WithSeed(configStore1.seed))
	time.Sleep(time.Second)
	nodes, _ := store1.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport), WithClock(_clock),
		WithSeed(configStore2.seed))
	time.Sleep(time.Second)
	nodes, _ = store2.ringpop.GetReachableMembers()
	assert.Equal(t, 2, len(nodes))
	store3 := NewStore(WithTChannelPort(configStore3.tchannelport), WithClock(_clock),
		WithSeed(configStore3.seed))
	time.Sleep(time.Second)
	nodes, _ = store3.ringpop.GetReachableMembers()
	assert.Equal(t, 3, len(nodes))
	store4 := NewStore(WithTChannelPort(configStore4.tchannelport), WithClock(_clock),
		WithSeed(configStore4.seed))
	time.Sleep(time.Second)
	nodes, _ = store4.ringpop.GetReachableMembers()
	assert.Equal(t, 4, len(store3.GetClusterInfo().Members))

}

func TestStoreCluster_GC(t *testing.T) {

	configStore1 := defaultOptions
	configStore1.tchannelport = "2375"
	configStore1.unsyncedTimeLimit = 1000
	configStore1.seed = hostname + ":" + configStore1.tchannelport

	configStore2 := defaultOptions
	configStore2.tchannelport = "2376"
	configStore2.unsyncedTimeLimit = 1000
	configStore2.seed = hostname + ":" + configStore1.tchannelport
	_clock := &clock.UnRealClock{}
	store1 := NewStore(WithTChannelPort(configStore1.tchannelport), WithClock(_clock),
		WithSeed(configStore1.seed),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit),
		WithGcGrace(1000),
		WithGcInterval(1000))
	time.Sleep(time.Second)
	nodes, _ := store1.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport), WithClock(_clock),
		WithSeed(configStore2.seed),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit),
		WithGcGrace(1000),
		WithGcInterval(1000))
	time.Sleep(time.Second)
	nodes, _ = store1.ringpop.GetReachableMembers()
	assert.Equal(t, 2, len(nodes))

	store1.Incr("one", 1, 1, 1, false)
	store2.Incr("one", 1, 1, 1, false)
	store1.Incr("two", 1, 1, 5, false)
	store2.Incr("three", 1, 1, 5, false)

	_clock.AddSeconds(3)
	time.Sleep(1080 * time.Millisecond)
	e1 := store1.getKeyBucket("one").GetEntry("one")
	e2 := store2.getKeyBucket("one").GetEntry("one")
	e3 := store2.getKeyBucket("two").GetEntry("two")
	e4 := store1.getKeyBucket("three").GetEntry("three")
	assert.True(t, e1 == nil)
	assert.True(t, e2 == nil)

	assert.False(t, e3 == nil)
	assert.False(t, e4 == nil)

	_clock.AddSeconds(4)
	time.Sleep(1080 * time.Millisecond)
	logrus.Infof("Checking Now")
	e3 = store2.getKeyBucket("two").GetEntry("two")
	e4 = store1.getKeyBucket("three").GetEntry("three")

	assert.True(t, e3 == nil)
	assert.True(t, e4 == nil)

}

func TestClusterWithRingPop(t *testing.T) {

	tchannel, err := tchannel.NewChannel("testc", nil)
	if err != nil {
		log.Fatalf("Cluster: could not listen on given hostport: %v", err)
	}
	ringpop, err := ringpop.New("testc",
		ringpop.Channel(tchannel),
		ringpop.Address("localhost:7772"))
	if err := tchannel.ListenAndServe("localhost:7772"); err != nil {
		log.Fatalf("Cluster: could not listen on given hostport: %v", err)
	}

	opts := new(swim.BootstrapOptions)

	opts.DiscoverProvider = statichosts.New("localhost:7772")

	if _, err := ringpop.Bootstrap(opts); err != nil {
		log.Fatalf("Cluster: ringpop bootstrap failed: %v", err)
	}

	store := NewStore(
		WithHostName("localhost"),
		WithRingpop(ringpop),
		WithTChannel(tchannel),
		WithHostName("localhost"),
		WithClusterName("testc"),
		WithTChannelPort("7772"))

	time.Sleep(time.Second)
	nodes, _ := store.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	store.Incr("one", 1, 1, 10, false)
}
