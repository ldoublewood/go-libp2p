package network

import (
	"context"

	"github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// DatagramHandler is the type of function used to handle incoming datagrams.
type DatagramHandler func(data []byte, from peer.ID, localAddr, remoteAddr ma.Multiaddr)

// DatagramConn represents a datagram-capable connection that can send and receive
// unreliable packets directly without stream multiplexing.
type DatagramConn interface {
	// SendDatagram sends a datagram to the remote peer.
	// The data is sent unreliably and may be lost, duplicated, or reordered.
	SendDatagram(ctx context.Context, data []byte) error

	// ReceiveDatagram receives a datagram from the remote peer.
	// This is a blocking call that returns when a datagram is available.
	ReceiveDatagram(ctx context.Context) ([]byte, error)

	// SetDatagramHandler sets a handler for incoming datagrams.
	// When set, ReceiveDatagram should not be used.
	SetDatagramHandler(handler DatagramHandler)

	// DatagramMTU returns the maximum transmission unit for datagrams.
	DatagramMTU() int

	// Close closes the datagram connection.
	Close() error

	// Connection information
	LocalPeer() peer.ID
	RemotePeer() peer.ID
	LocalMultiaddr() ma.Multiaddr
	RemoteMultiaddr() ma.Multiaddr
}

//// DatagramCapableConn extends Conn with datagram capabilities.
//// This interface represents a connection that supports both regular streams
//// and datagram transmission.
//type DatagramCapableConn interface {
//	Conn
//
//	// AsDatagramConn returns a DatagramConn if the underlying transport supports datagrams.
//	// Returns nil if datagrams are not supported.
//	AsDatagramConn() DatagramConn
//
//	// SupportsDatagrams returns true if the connection supports datagram transmission.
//	SupportsDatagrams() bool
//}

// DatagramNetwork extends Network with datagram capabilities.
type DatagramNetwork interface {
	Network

	// SetDatagramHandler sets the global handler for incoming datagrams.
	SetDatagramHandler(DatagramHandler) error

	// SendDatagram sends a datagram to a specific peer.
	// If no datagram-capable connection exists, it will attempt to create one.
	SendDatagram(ctx context.Context, p peer.ID, data []byte) error

	// GetDatagramConn returns a datagram connection to the specified peer.
	// Returns nil if no datagram-capable connection exists.
	GetDatagramConn(p peer.ID) (DatagramConn, error)

	StatDatagramConn(p peer.ID) (total int, datagramTotal int, err error)
}

// DatagramTransport represents a transport that supports datagram transmission.
// This is a minimal interface that can be implemented by transports that support datagrams.
type DatagramTransport interface {
	// SupportsDatagrams returns true if this transport supports datagram transmission.
	SupportsDatagrams() bool
}
