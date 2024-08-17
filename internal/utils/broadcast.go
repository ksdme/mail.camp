package utils

import (
	"sync"
)

// A broadcasting event bus. It also has the additional functionality
// of always cleaning up after itself. You can also emit messages to
// topics that don't exist yet. Those messages will however be drained away.
type BroadcastBus[T any] struct {
	channels map[int64][]chan T
	lock     sync.Mutex
}

func NewBroadcastBus[T any]() BroadcastBus[T] {
	return BroadcastBus[T]{
		channels: make(map[int64][]chan T),
		lock:     sync.Mutex{},
	}
}

// Emit a message on a topic with a message.
func (b *BroadcastBus[T]) Emit(subject int64, message T) {
	b.lock.Lock()
	channels, ok := b.channels[subject]
	if ok {
		delete(b.channels, subject)
	}
	b.lock.Unlock()

	for _, channel := range channels {
		select {
		case channel <- message:
			close(channel)
		default:
		}
	}
}

// Wait for a message on the topic. Returns the message and a bool
// flag that indicates if the wait was aborted. This usually happens
// when the topic is being cleaned up.
func (b *BroadcastBus[T]) Wait(subject int64) (T, bool) {
	b.lock.Lock()
	channel := make(chan T)
	channels, ok := b.channels[subject]
	if ok {
		b.channels[subject] = append(channels, channel)
	} else {
		b.channels[subject] = []chan T{channel}
	}
	b.lock.Unlock()

	if value, ok := <-channel; ok {
		return value, false
	} else {
		var zero T
		return zero, true
	}
}

// Clean up a topic on the bus. All pending waits will resolve
// with a done flag.
func (b *BroadcastBus[T]) CleanUp(subject int64) {
	b.lock.Lock()
	defer b.lock.Unlock()

	if channels, ok := b.channels[subject]; ok {
		for _, channel := range channels {
			close(channel)
		}
		delete(b.channels, subject)
	}
}
