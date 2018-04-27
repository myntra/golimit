package event

import (
	log "github.com/sirupsen/logrus"
	"sync"
)

const (
	KEYEVENT = "KEYEVENT"
)

type KeyEvent struct {
	Key     string
	Count   int32
	Time    int64
	Allowed bool
}

func (e KeyEvent) GetRoute() string {
	return KEYEVENT
}

type goLimitEventManager struct {
	sync.RWMutex
	workerCount int
	handlers    map[string][]EventHandler
	eventChan   chan (Event)
	objPools    map[string]*sync.Pool
	poolLock    sync.RWMutex
}

var instance *goLimitEventManager
var once sync.Once

func GetMgrInstance() EventManager {
	once.Do(func() {
		instance = &goLimitEventManager{workerCount: 1000, handlers: make(map[string][]EventHandler),
			eventChan: make(chan (Event), 100000)}
		instance.objPools = make(map[string]*sync.Pool)
		instance.poolLock = sync.RWMutex{}
		instance.startEventWorkers()

	})
	return instance
}

func (em *goLimitEventManager) GetPool(poolname string) *sync.Pool {
	em.poolLock.RLock()
	pool := em.objPools[poolname]
	em.poolLock.RUnlock()
	if pool == nil {
		em.poolLock.Lock()
		pool = em.createNewPool(poolname)
		em.objPools[poolname] = pool
		em.poolLock.Unlock()
	}
	return pool
}

func (em *goLimitEventManager) createNewPool(poolname string) *sync.Pool {
	switch poolname {
	case KEYEVENT:

		return &sync.Pool{
			New: func() interface{} {
				return &KeyEvent{}
			},
		}

	default:
		log.Errorf("pool (%s) definitation not registered", poolname)
	}
	return nil

}

func (em *goLimitEventManager) RegisterHandler(routekey string, handler EventHandler) {
	em.Lock()
	defer em.Unlock()

	em.handlers[routekey] = append(em.handlers[routekey], handler)
}

func (em *goLimitEventManager) UnRegisterHandler(routekey string, handler EventHandler) {
	em.Lock()
	defer em.Unlock()

	for i, v := range em.handlers[routekey] {
		if v == handler {
			em.handlers[routekey] = append(em.handlers[routekey][:i], em.handlers[routekey][i+1:]...)
			break
		}
	}

}

func (em *goLimitEventManager) startEventWorkers() {
	for i := 0; i < em.workerCount; i++ {
		go func() {
			for e := range em.eventChan {
				em.RLock()
				handlers := em.handlers[e.GetRoute()]
				em.RUnlock()
				for _, h := range handlers {
					h.Handle(e)
				}
				em.GetPool(e.GetRoute()).Put(e)

			}
		}()
	}
}

func (em *goLimitEventManager) Publish(event Event) {

	select {
	case em.eventChan <- event:
	default:
		log.Errorf("Event Channel Full, discarding event")
	}

}
