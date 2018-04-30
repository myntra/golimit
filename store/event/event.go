package event

import "sync"

type Event interface {
	GetRoute() string
}

type EventHandler interface {
	Handle(event Event)
}

type EventManager interface {
	GetPool(poolname string) *sync.Pool
	RegisterHandler(routekey string, handler EventHandler)
	UnRegisterHandler(routekey string, handler EventHandler)
	Publish(event Event) bool
}
