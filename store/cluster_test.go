package store

import (
	"github.com/myntra/golimit/store/bucket"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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

	store1 := NewStore(WithTChannelPort(configStore1.tchannelport),
		WithSeed(configStore1.seed),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport),
		WithSeed(configStore2.seed),
		WithUnsyncedTimeLimit(configStore2.unsyncedTimeLimit))
	store3 := NewStore(WithTChannelPort(configStore3.tchannelport),
		WithSeed(configStore3.seed),
		WithUnsyncedTimeLimit(configStore3.unsyncedTimeLimit))
	store4 := NewStore(WithTChannelPort(configStore4.tchannelport),
		WithSeed(configStore4.seed),
		WithUnsyncedTimeLimit(configStore4.unsyncedTimeLimit))

	ti := time.Now().UnixNano()
	expiry := bucket.GenExpiry(ti, 60)
	if expiry-ti < (time.Second * 60).Nanoseconds() {
		time.Sleep(time.Nanosecond * time.Duration(expiry-ti))
	}

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

	time.Sleep(time.Second)

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

	store1 := NewStore(WithTChannelPort(configStore1.tchannelport),
		WithSeed(configStore1.seed),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport),
		WithSeed(configStore2.seed),
		WithUnsyncedTimeLimit(configStore2.unsyncedTimeLimit))
	store3 := NewStore(WithTChannelPort(configStore3.tchannelport),
		WithSeed(configStore3.seed),
		WithUnsyncedTimeLimit(configStore3.unsyncedTimeLimit))
	store4 := NewStore(WithTChannelPort(configStore4.tchannelport),
		WithSeed(configStore4.seed),
		WithUnsyncedTimeLimit(configStore4.unsyncedTimeLimit))

	ti := time.Now().UnixNano()
	expiry := bucket.GenExpiry(ti, 60)
	if expiry-ti < (time.Second * 60).Nanoseconds() {
		time.Sleep(time.Nanosecond * time.Duration(expiry-ti))
	}

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

	store1 := NewStore(WithTChannelPort(configStore1.tchannelport),
		WithSeed(configStore1.seed))
	time.Sleep(time.Second)
	nodes, _ := store1.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport),
		WithSeed(configStore2.seed))
	time.Sleep(time.Second)
	nodes, _ = store2.ringpop.GetReachableMembers()
	assert.Equal(t, 2, len(nodes))
	store3 := NewStore(WithTChannelPort(configStore3.tchannelport),
		WithSeed(configStore3.seed))
	time.Sleep(time.Second)
	nodes, _ = store3.ringpop.GetReachableMembers()
	assert.Equal(t, 3, len(nodes))
	store4 := NewStore(WithTChannelPort(configStore4.tchannelport),
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

	store1 := NewStore(WithTChannelPort(configStore1.tchannelport),
		WithSeed(configStore1.seed),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit),
		WithGcGrace(1000),
		WithGcInterval(1000))
	time.Sleep(time.Second)
	nodes, _ := store1.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	store2 := NewStore(WithTChannelPort(configStore2.tchannelport),
		WithSeed(configStore2.seed),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit),
		WithGcGrace(1000),
		WithGcInterval(1000))
	time.Sleep(time.Second)
	nodes, _ = store1.ringpop.GetReachableMembers()
	assert.Equal(t, 2, len(nodes))

	ti := time.Now().UnixNano()
	expiry := bucket.GenExpiry(ti, 10)
	if expiry-ti < (time.Second * 10).Nanoseconds() {
		time.Sleep(time.Nanosecond * time.Duration(expiry-ti))
	}

	store1.Incr("one", 1, 1, 1, false)
	store2.Incr("one", 1, 1, 1, false)
	store1.Incr("two", 1, 1, 5, false)
	store2.Incr("three", 1, 1, 5, false)

	time.Sleep(time.Millisecond * 2500)

	e1 := store1.getKeyBucket("one").GetEntry("one")
	e2 := store2.getKeyBucket("one").GetEntry("one")
	e3 := store2.getKeyBucket("two").GetEntry("two")
	e4 := store1.getKeyBucket("three").GetEntry("three")
	assert.True(t, e1 == nil)
	assert.True(t, e2 == nil)

	assert.False(t, e3 == nil)
	assert.False(t, e4 == nil)

	time.Sleep(time.Millisecond * 10000)
	logrus.Infof("Checking Now")
	e3 = store2.getKeyBucket("two").GetEntry("two")
	e4 = store1.getKeyBucket("three").GetEntry("three")

	assert.True(t, e3 == nil)
	assert.True(t, e4 == nil)

}
