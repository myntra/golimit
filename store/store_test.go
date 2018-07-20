package store

import (
	"github.com/myntra/golimit/store/clock"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	configStore1 := defaultOptions
	configStore1.tchannelport = "2381"
	configStore1.unsyncedTimeLimit = 1000
	configStore1.apiSecret = "helloworld"
	configStore1.seed = HOSTNAME + ":" + configStore1.tchannelport

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

	assert.False(t, store1.IsAuthorised("pingpong"))
	assert.True(t, store1.IsAuthorised("helloworld"))

	store1.Incr("one", 1, 1, 1, false)

}
