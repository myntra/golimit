package bucket

import (
	log "github.com/sirupsen/logrus"
	"sync"
	"time"
)

type KeyBucket struct {
	sync.RWMutex
	lookup map[string]*KeyEntry
}

func NewKeyBucket() *KeyBucket {
	b := &KeyBucket{}
	b.lookup = make(map[string]*KeyEntry)
	return b
}

func (b *KeyBucket) GetEntry(key string) *KeyEntry {
	b.RLock()
	e := b.lookup[key]
	b.RUnlock()
	return e
}

func (b *KeyBucket) Incr(key string, count int32, limit int32, window int32) (bool, int64, bool) {

	log.Debugf("Incr: incrementing key, count, limit, window %s, %d, %d, %d %t", key, count, limit)
	t := time.Now().UnixNano()
	e := b.GetEntry(key)
	if e == nil {
		log.Debugf("Incr: entry not found %s", key)
		b.Lock()
		e = b.lookup[key]
		if e == nil {
			log.Debugf("Incr: Double check entry not found %s", key)
			e = NewEntry(0, genExpiry(t, window))
			b.lookup[key] = e
		}
		b.Unlock()
	}
	e.Lock()
	if e.Expired() {
		log.Debugf("Incr: Found entry expired, reiniting %s", key)
		e.ReInit(0, genExpiry(t, window))
	}

	previousTotal := e.Count()
	newTotal := previousTotal
	log.Debugf("Incr: Previous count, Key %s : count %d ", key, previousTotal)
	if previousTotal <= limit {
		newTotal = e.Incr(count)
	}
	log.Debugf("Incr: New count, Key %s : count %d ", key, newTotal)
	log.Debugf("Incr: Incremented count, Key %s : count %d", key, previousTotal)
	e.Unlock()
	log.Debugf("Incr: Incremented count, Total %d : Limit %d", newTotal, limit)
	if newTotal <= limit {
		log.Debugf("Incr: returning ALLOWED, Key %s : count %d", key, newTotal)
		return false, e.Expiry(), false
	} else {
		log.Debugf("Incr: returning BLOCKED, Key %s : count %d", key, newTotal)
		return true, e.Expiry(), previousTotal < newTotal
	}
}

func (b *KeyBucket) Sync(key string, count int32, expiry int64) {
	if time.Now().UnixNano() > expiry {
		return
	}
	e := b.GetEntry(key)
	if e == nil {
		b.Lock()
		e = b.lookup[key]
		if e == nil {
			e = NewEntry(0, expiry)
			b.lookup[key] = e
		}
		b.Unlock()
	}
	e.Lock()
	if e.Expired() {
		e.ReInit(0, expiry)
	}
	e.Sync(count)
	e.Unlock()
}

/**
ttl in seconds,
return expiry in Nano
*/
func genExpiry(curTime int64, ttl int32) int64 {
	if ttl <= 0 {
		return curTime
	}
	ttlNano := time.Duration(ttl) * time.Second
	return curTime + int64(ttlNano) - (curTime % int64(ttlNano))
}

func (b *KeyBucket) Lookup() map[string]*KeyEntry {
	return b.lookup
}
