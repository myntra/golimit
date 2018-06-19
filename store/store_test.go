package store

import (
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
	"github.com/myntra/golimit/store/clock"
)

func TestNewStore(t *testing.T) {
	configStore1 := defaultOptions
	configStore1.tchannelport = "2375"
	configStore1.unsyncedTimeLimit = 1000
	configStore1.apiSecret="helloworld"
	configStore1.seed = hostname + ":" + configStore1.tchannelport

	_clock := &clock.UnRealClock{}
	store1 := NewStore(WithTChannelPort(configStore1.tchannelport), WithClock(_clock),
		WithSeed(configStore1.seed),
		WithUnsyncedTimeLimit(configStore1.unsyncedTimeLimit),
		WithGcGrace(1000),
		WithGcInterval(1000),
		WithSyncBuffer(100),
		WithBucketSize(100),
		WithApiSecret(configStore1.apiSecret))
	time.Sleep(time.Second)
	nodes, _ := store1.ringpop.GetReachableMembers()
	assert.Equal(t, 1, len(nodes))

	assert.False(t,store1.IsAuthorised("pingpong"))
	assert.True(t,store1.IsAuthorised("helloworld"))


	store1.Incr("one", 1, 1, 1, false)


}

