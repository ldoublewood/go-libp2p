package basichost

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

// datagramHost extends BasicHost with datagram capabilities
type datagramHost struct {
	*BasicHost

	handlerMu sync.RWMutex
	handler   network.DatagramHandler
}

// NewDatagramHost creates a new datagram-capable host
func NewDatagramHost(bh *BasicHost) network.DatagramNetwork {
	return &datagramHost{
		BasicHost: bh,
	}
}

// Network interface methods - delegate to BasicHost's Network
func (dh *datagramHost) SetStreamHandler(handler network.StreamHandler) {
	dh.BasicHost.Network().SetStreamHandler(handler)
}

func (dh *datagramHost) NewStream(ctx context.Context, p peer.ID) (network.Stream, error) {
	return dh.BasicHost.Network().NewStream(ctx, p)
}

func (dh *datagramHost) Listen(addrs ...ma.Multiaddr) error {
	return dh.BasicHost.Network().Listen(addrs...)
}

func (dh *datagramHost) ListenAddresses() []ma.Multiaddr {
	return dh.BasicHost.Network().ListenAddresses()
}

func (dh *datagramHost) InterfaceListenAddresses() ([]ma.Multiaddr, error) {
	return dh.BasicHost.Network().InterfaceListenAddresses()
}

func (dh *datagramHost) ResourceManager() network.ResourceManager {
	return dh.BasicHost.Network().ResourceManager()
}

func (dh *datagramHost) Close() error {
	return dh.BasicHost.Network().Close()
}

// Dialer interface methods - delegate to BasicHost's Network
func (dh *datagramHost) Peerstore() peerstore.Peerstore {
	return dh.BasicHost.Network().Peerstore()
}

func (dh *datagramHost) LocalPeer() peer.ID {
	return dh.BasicHost.Network().LocalPeer()
}

func (dh *datagramHost) DialPeer(ctx context.Context, p peer.ID) (network.Conn, error) {
	return dh.BasicHost.Network().DialPeer(ctx, p)
}

func (dh *datagramHost) ClosePeer(p peer.ID) error {
	return dh.BasicHost.Network().ClosePeer(p)
}

func (dh *datagramHost) Connectedness(p peer.ID) network.Connectedness {
	return dh.BasicHost.Network().Connectedness(p)
}

func (dh *datagramHost) Peers() []peer.ID {
	return dh.BasicHost.Network().Peers()
}

func (dh *datagramHost) Conns() []network.Conn {
	return dh.BasicHost.Network().Conns()
}

func (dh *datagramHost) ConnsToPeer(p peer.ID) []network.Conn {
	return dh.BasicHost.Network().ConnsToPeer(p)
}

func (dh *datagramHost) Notify(notifiee network.Notifiee) {
	dh.BasicHost.Network().Notify(notifiee)
}

func (dh *datagramHost) StopNotify(notifiee network.Notifiee) {
	dh.BasicHost.Network().StopNotify(notifiee)
}

func (dh *datagramHost) CanDial(p peer.ID, addr ma.Multiaddr) bool {
	return dh.BasicHost.Network().CanDial(p, addr)
}

func (dh *datagramHost) SetDatagramHandler(handler network.DatagramHandler) error {
	dh.handlerMu.Lock()
	dh.handler = handler
	dh.handlerMu.Unlock()

	// Set handler on all existing datagram-capable connections
	for _, conn := range dh.Network().Conns() {
		dcConn, err := conn.AsDatagramConn()
		if err != nil {
			return fmt.Errorf("cannot create DatagramConn from %v: %w", conn.RemotePeer(), err)
		}
		if dcConn != nil {
			dcConn.SetDatagramHandler(handler)
		}
	}
	return nil
}

func (dh *datagramHost) SendDatagram(ctx context.Context, p peer.ID, data []byte) error {
	// Try to find an existing datagram-capable connection
	dgConn, err := dh.GetDatagramConn(p)
	if err != nil {
		return fmt.Errorf("cannot get DatagramConn from %v: %w", p, err)
	}
	if dgConn != nil {
		return dgConn.SendDatagram(ctx, data)
	}

	// No existing connection, try to establish one
	conn, err := dh.Network().DialPeer(ctx, p)
	if err != nil {
		return fmt.Errorf("cannot dial Datagram from %v: %w", p, err)
	}
	dcConn, err := conn.AsDatagramConn()
	if err != nil {
		return fmt.Errorf("cannot create DatagramConn from %v: %w", conn.RemotePeer(), err)
	}

	//conn newly dialed does not support datagram, try the whole host again
	if dcConn == nil {
		dgConn, err := dh.GetDatagramConn(p)
		if err != nil {
			return fmt.Errorf("cannot get DatagramConn from %v: %w", p, err)
		}
		if dgConn != nil {
			return dgConn.SendDatagram(ctx, data)
		}
	}
	return ErrDatagramNotSupported
}

func (dh *datagramHost) StatDatagramConn(p peer.ID) (total int, datagramTotal int, err error) {
	conns := dh.Network().ConnsToPeer(p)
	total = len(conns)
	datagramTotal = 0
	for _, conn := range conns {
		dcConn, err := conn.AsDatagramConn()
		if err != nil {
			return total, datagramTotal, fmt.Errorf("cannot create DatagramConn from %v: %w", conn.RemotePeer(), err)
		}
		if dcConn == nil {
			continue
		}
		datagramTotal++
	}
	return total, datagramTotal, nil
}
func (dh *datagramHost) GetDatagramConn(p peer.ID) (network.DatagramConn, error) {
	conns := dh.Network().ConnsToPeer(p)
	for _, conn := range conns {

		dcConn, err := conn.AsDatagramConn()
		if err != nil {
			return nil, fmt.Errorf("cannot create DatagramConn from %v: %w", conn.RemotePeer(), err)
		}
		if dcConn == nil {
			continue
		}
		return dcConn, nil
	}
	return nil, nil
}

var ErrDatagramNotSupported = errors.New("datagram not supported by any available connection")
