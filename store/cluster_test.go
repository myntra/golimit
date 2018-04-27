package store

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestStoreCluster_Ctr_Limit_test(t *testing.T) {
	os.Remove("rateconfig.json")
	configStore1 := NewDefaultStoreConfig()
	configStore1.StatsDEnabled = false
	configStore1.UnsyncedTimeLimit = 1000
	configStore1.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore2 := NewDefaultStoreConfig()
	configStore2.StatsDEnabled = false
	configStore2.TChannelPort = "2346"
	configStore2.UnsyncedTimeLimit = 1000
	configStore2.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore3 := NewDefaultStoreConfig()
	configStore3.StatsDEnabled = false
	configStore3.TChannelPort = "2347"
	configStore3.UnsyncedTimeLimit = 1000
	configStore3.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore4 := NewDefaultStoreConfig()
	configStore4.StatsDEnabled = false
	configStore4.TChannelPort = "2348"
	configStore4.UnsyncedTimeLimit = 1000
	configStore4.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	store1 := NewStore(configStore1)
	store2 := NewStore(configStore2)
	store3 := NewStore(configStore3)
	store4 := NewStore(configStore4)

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

	configStore1 := NewDefaultStoreConfig()
	configStore1.StatsDEnabled = false
	configStore1.TChannelPort = "2355"
	configStore1.UnsyncedTimeLimit = 1000
	configStore1.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore2 := NewDefaultStoreConfig()
	configStore2.StatsDEnabled = false
	configStore2.TChannelPort = "2356"
	configStore2.UnsyncedTimeLimit = 1000
	configStore2.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore3 := NewDefaultStoreConfig()
	configStore3.StatsDEnabled = false
	configStore3.TChannelPort = "2357"
	configStore3.UnsyncedTimeLimit = 1000
	configStore3.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore4 := NewDefaultStoreConfig()
	configStore4.StatsDEnabled = false
	configStore4.TChannelPort = "2358"
	configStore4.UnsyncedTimeLimit = 1000
	configStore4.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	store1 := NewStore(configStore1)
	store2 := NewStore(configStore2)
	store3 := NewStore(configStore3)
	store4 := NewStore(configStore4)

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
	blocked = store1.Incr("two", 1, 10, 60, false)
	assert.True(t, blocked)

}

func TestStoreCluster_Node_Join(t *testing.T) {

	configStore1 := NewDefaultStoreConfig()
	configStore1.StatsDEnabled = false
	configStore1.TChannelPort = "2365"
	configStore1.UnsyncedTimeLimit = 1000
	configStore1.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore2 := NewDefaultStoreConfig()
	configStore2.StatsDEnabled = false
	configStore2.TChannelPort = "2366"
	configStore2.UnsyncedTimeLimit = 1000
	configStore2.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore3 := NewDefaultStoreConfig()
	configStore3.StatsDEnabled = false
	configStore3.TChannelPort = "2367"
	configStore3.UnsyncedTimeLimit = 1000
	configStore3.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore4 := NewDefaultStoreConfig()
	configStore4.StatsDEnabled = false
	configStore4.TChannelPort = "2368"
	configStore4.UnsyncedTimeLimit = 1000
	configStore4.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	store1 := NewStore(configStore1)
	time.Sleep(time.Second)
	nodes, _ := store1.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	store2 := NewStore(configStore2)
	time.Sleep(time.Second)
	nodes, _ = store2.ringpop.GetReachableMembers()
	assert.Equal(t, 2, len(nodes))
	store3 := NewStore(configStore3)
	time.Sleep(time.Second)
	nodes, _ = store3.ringpop.GetReachableMembers()
	assert.Equal(t, 3, len(nodes))
	store4 := NewStore(configStore4)
	time.Sleep(time.Second)
	nodes, _ = store4.ringpop.GetReachableMembers()
	assert.Equal(t, 4, len(store3.GetClusterInfo().Members))

}

func TestStoreCluster_GC(t *testing.T) {

	configStore1 := NewDefaultStoreConfig()
	configStore1.StatsDEnabled = true
	configStore1.TChannelPort = "2375"
	configStore1.UnsyncedTimeLimit = 1000
	configStore1.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	configStore2 := NewDefaultStoreConfig()
	configStore2.StatsDEnabled = true
	configStore2.TChannelPort = "2376"
	configStore2.UnsyncedTimeLimit = 1000
	configStore2.GcInterval = 1000
	configStore2.GcGrace = 1000
	configStore2.Seed = configStore1.HostName + ":" + configStore1.TChannelPort

	store1 := NewStore(configStore1)
	time.Sleep(time.Second)
	nodes, _ := store1.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))
	store2 := NewStore(configStore2)
	time.Sleep(time.Second)

	store1.Incr("one", 1, 1, 1, false)
	store2.Incr("one", 1, 1, 1, false)

	time.Sleep(time.Millisecond * 2500)

	e1 := store1.getKeyBucket("one").GetEntry("one")
	e2 := store2.getKeyBucket("one").GetEntry("one")
	assert.True(t, e1 != nil)
	assert.True(t, e2 == nil)

}
