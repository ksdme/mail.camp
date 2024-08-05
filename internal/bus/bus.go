package bus

import (
	"sync"
)

// A broadcasting signal bus. It also has the additional functionality
// of always cleaning up after itself. You can also emit messages to
// topics that don't exist yet. Those messages will be however be
// drained.
type SignalBus[T any] struct {
	channels map[int64][]chan T
	lock     sync.Mutex
}

func NewSignalBus[T any]() SignalBus[T] {
	return SignalBus[T]{
		channels: make(map[int64][]chan T),
		lock:     sync.Mutex{},
	}
}

// Emit a message on a topic with a message.
func (s *SignalBus[T]) Emit(topic int64, message T) {
	channels := func() []chan T {
		s.lock.Lock()
		defer s.lock.Unlock()

		if channels, ok := s.channels[topic]; ok {
			delete(s.channels, topic)
			return channels
		}
		return nil
	}()

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
func (s *SignalBus[T]) Wait(topic int64) (T, bool) {
	channel := make(chan T)

	func() {
		s.lock.Lock()
		defer s.lock.Unlock()

		if channels, ok := s.channels[topic]; ok {
			s.channels[topic] = append(channels, channel)
		} else {
			s.channels[topic] = []chan T{channel}
		}
	}()

	if value, ok := <-channel; ok {
		return value, false
	} else {
		var zero T
		return zero, true
	}
}

// Clean up a topic on the bus. All pending waits will resolve
// with a done flag.
func (s *SignalBus[T]) CleanUp(topic int64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if channels, ok := s.channels[topic]; ok {
		for _, channel := range channels {
			close(channel)
		}
		delete(s.channels, topic)
	}
}
