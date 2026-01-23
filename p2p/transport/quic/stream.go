package libp2pquic

import (
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"

	"github.com/quic-go/quic-go"
)

const (
	reset quic.StreamErrorCode = 0
)

type stream struct {
	quic.Stream
	conn     *conn
	id       string
	protocol protocol.ID
}

var _ network.MuxedStream = &stream{}

func (s *stream) Read(b []byte) (n int, err error) {
	n, err = s.Stream.Read(b)
	if err != nil && errors.Is(err, &quic.StreamError{}) {
		err = network.ErrReset
	}
	return n, err
}

func (s *stream) Write(b []byte) (n int, err error) {
	n, err = s.Stream.Write(b)
	if err != nil && errors.Is(err, &quic.StreamError{}) {
		err = network.ErrReset
	}
	return n, err
}

func (s *stream) Reset() error {
	s.Stream.CancelRead(reset)
	s.Stream.CancelWrite(reset)
	return nil
}

func (s *stream) Close() error {
	s.Stream.CancelRead(reset)
	return s.Stream.Close()
}

func (s *stream) CloseRead() error {
	s.Stream.CancelRead(reset)
	return nil
}

func (s *stream) CloseWrite() error {
	return s.Stream.Close()
}

// ID returns an identifier that uniquely identifies this Stream within this
// host, during this run. Stream IDs may repeat across restarts.
func (s *stream) ID() string {
	if s.id == "" {
		s.id = fmt.Sprintf("quic-%d", s.Stream.StreamID())
	}
	return s.id
}

// Protocol returns the protocol ID for this stream.
func (s *stream) Protocol() protocol.ID {
	return s.protocol
}

// SetProtocol sets the protocol ID for this stream.
func (s *stream) SetProtocol(id protocol.ID) error {
	s.protocol = id
	return nil
}

// Stat returns metadata pertaining to this stream.
func (s *stream) Stat() network.Stats {
	return network.Stats{
		Direction: network.DirUnknown, // QUIC doesn't distinguish direction at stream level
	}
}

// Conn returns the connection this stream is part of.
func (s *stream) Conn() network.Conn {
	return s.conn
}

// Scope returns the user's view of this stream's resource scope.
func (s *stream) Scope() network.StreamScope {
	// Return a no-op scope for now
	return &noopStreamScope{}
}

// noopStreamScope is a no-op implementation of StreamScope
type noopStreamScope struct{}

func (n *noopStreamScope) SetService(svc string) error { return nil }
func (n *noopStreamScope) Service() string             { return "" }
func (n *noopStreamScope) SetProtocol(proto protocol.ID) error { return nil }
func (n *noopStreamScope) Protocol() protocol.ID       { return "" }
func (n *noopStreamScope) SetPeer(p peer.ID) error     { return nil }
func (n *noopStreamScope) Peer() peer.ID               { return "" }
func (n *noopStreamScope) Done()                       {}
func (n *noopStreamScope) Release()                    {}
func (n *noopStreamScope) Stat() network.ScopeStat     { return network.ScopeStat{} }
func (n *noopStreamScope) ReserveMemory(size int, prio uint8) error { return nil }
func (n *noopStreamScope) ReleaseMemory(size int) {}
func (n *noopStreamScope) BeginSpan() (network.ResourceScopeSpan, error) { 
	return &noopResourceScopeSpan{}, nil 
}

// noopResourceScopeSpan is a no-op implementation of ResourceScopeSpan
type noopResourceScopeSpan struct{}

func (n *noopResourceScopeSpan) ReserveMemory(size int, prio uint8) error { return nil }
func (n *noopResourceScopeSpan) ReleaseMemory(size int) {}
func (n *noopResourceScopeSpan) Stat() network.ScopeStat { return network.ScopeStat{} }
func (n *noopResourceScopeSpan) BeginSpan() (network.ResourceScopeSpan, error) { 
	return &noopResourceScopeSpan{}, nil 
}
func (n *noopResourceScopeSpan) Done() {}
