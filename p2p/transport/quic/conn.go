package libp2pquic

import (
	"context"
	"fmt"
	"time"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	tpt "github.com/libp2p/go-libp2p/core/transport"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/quic-go/quic-go"
)

type conn struct {
	quicConn  quic.Connection
	transport *transport
	scope     network.ConnManagementScope

	localPeer      peer.ID
	localMultiaddr ma.Multiaddr

	remotePeerID    peer.ID
	remotePubKey    ic.PubKey
	remoteMultiaddr ma.Multiaddr
}

var _ tpt.CapableConn = &conn{}
var _ network.DatagramConn = &conn{}

// Close closes the connection.
// It must be called even if the peer closed the connection in order for
// garbage collection to properly work in this package.
func (c *conn) Close() error {
	return c.closeWithError(0, "")
}

func (c *conn) closeWithError(errCode quic.ApplicationErrorCode, errString string) error {
	c.transport.removeConn(c.quicConn)
	err := c.quicConn.CloseWithError(errCode, errString)
	c.scope.Done()
	return err
}

// IsClosed returns whether a connection is fully closed.
func (c *conn) IsClosed() bool {
	return c.quicConn.Context().Err() != nil
}

func (c *conn) allowWindowIncrease(size uint64) bool {
	return c.scope.ReserveMemory(int(size), network.ReservationPriorityMedium) == nil
}

// OpenStream creates a new stream.
func (c *conn) OpenStream(ctx context.Context) (network.MuxedStream, error) {
	qstr, err := c.quicConn.OpenStreamSync(ctx)
	return &stream{Stream: qstr}, err
}

// AcceptStream accepts a stream opened by the other side.
func (c *conn) AcceptStream() (network.MuxedStream, error) {
	qstr, err := c.quicConn.AcceptStream(context.Background())
	return &stream{Stream: qstr}, err
}

// LocalPeer returns our peer ID
func (c *conn) LocalPeer() peer.ID { return c.localPeer }

// RemotePeer returns the peer ID of the remote peer.
func (c *conn) RemotePeer() peer.ID { return c.remotePeerID }

// RemotePublicKey returns the public key of the remote peer.
func (c *conn) RemotePublicKey() ic.PubKey { return c.remotePubKey }

// LocalMultiaddr returns the local Multiaddr associated
func (c *conn) LocalMultiaddr() ma.Multiaddr { return c.localMultiaddr }

// RemoteMultiaddr returns the remote Multiaddr associated
func (c *conn) RemoteMultiaddr() ma.Multiaddr { return c.remoteMultiaddr }

func (c *conn) Transport() tpt.Transport { return c.transport }

func (c *conn) Scope() network.ConnScope { return c.scope }

// ConnState is the state of security connection.
func (c *conn) ConnState() network.ConnectionState {
	t := "quic-v1"
	if _, err := c.LocalMultiaddr().ValueForProtocol(ma.P_QUIC); err == nil {
		t = "quic"
	}
	return network.ConnectionState{Transport: t}
}

// ID returns an identifier that uniquely identifies this Conn within this
// host, during this run. Connection IDs may repeat across restarts.
func (c *conn) ID() string {
	return fmt.Sprintf("%p", c)
}

// NewStream constructs a new Stream over this conn.
func (c *conn) NewStream(ctx context.Context) (network.Stream, error) {
	qstr, err := c.quicConn.OpenStreamSync(ctx)
	if err != nil {
		return nil, err
	}
	return &stream{
		Stream: qstr,
		conn:   c,
	}, nil
}

// GetStreams returns all open streams over this conn.
func (c *conn) GetStreams() []network.Stream {
	// QUIC doesn't provide a direct way to enumerate all streams
	// This is a limitation of the QUIC-go library
	// For now, return an empty slice
	return []network.Stream{}
}

// Stat stores metadata pertaining to this conn.
func (c *conn) Stat() network.ConnStats {
	return network.ConnStats{
		Stats: network.Stats{
			Direction: network.DirOutbound, // This should be set properly based on connection direction
			Opened:    time.Now(),          // This should be set when connection is established
			Limited:   false,
			Extra:     make(map[interface{}]interface{}),
		},
		NumStreams: 0, // QUIC doesn't provide easy access to stream count
	}
}

// SendDatagram sends a datagram message over the QUIC connection.
func (c *conn) SendDatagram(data []byte) error {
	if !c.SupportsDatagrams() {
		return fmt.Errorf("connection does not support datagrams")
	}
	return c.quicConn.SendDatagram(data)
}

// ReceiveDatagram receives a datagram message from the QUIC connection.
func (c *conn) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	if !c.SupportsDatagrams() {
		return nil, fmt.Errorf("connection does not support datagrams")
	}
	return c.quicConn.ReceiveDatagram(ctx)
}

// SupportsDatagrams returns true if the QUIC connection supports datagram transmission.
func (c *conn) SupportsDatagrams() bool {
	// QUIC connections support datagrams if both peers negotiated the extension
	return c.quicConn.ConnectionState().SupportsDatagrams
}
