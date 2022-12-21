package grpcboot

import (
	"context"
	"sync"

	"google.golang.org/grpc"
)

type StreamManager[T any] struct {
	streams []grpc.ServerStream
	lock    sync.Mutex
	Context context.Context
}

func NewStreamManager[T any](ctx context.Context) *StreamManager[T] {
	return &StreamManager[T]{
		lock:    sync.Mutex{},
		Context: ctx,
	}
}

func (m *StreamManager[T]) Add(stream grpc.ServerStream) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.streams = append(m.streams, stream)
}

func (m *StreamManager[T]) Send(value T) {
	m.lock.Lock()
	defer m.lock.Unlock()
	for _, s := range m.streams {
		s.SendMsg(value)
	}
}

func (m *StreamManager[T]) Wait() {
	<-m.Context.Done()
}
