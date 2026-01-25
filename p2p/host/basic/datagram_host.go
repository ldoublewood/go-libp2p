package basichost

import (
	"context"
	"errors"
	"sync"

	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

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

func (dh *datagramHost) SetDatagramHandler(handler network.DatagramHandler) {
	dh.handlerMu.Lock()
	dh.handler = handler
	dh.handlerMu.Unlock()
	
	// Set handler on all existing datagram-capable connections
	for _, conn := range dh.Network().Conns() {
		if dcConn, ok := conn.(network.DatagramCapableConn); ok && dcConn.SupportsDatagrams() {
			if dgConn := dcConn.AsDatagramConn(); dgConn != nil {
				dgConn.SetDatagramHandler(handler)
			}
		}
	}
}

func (dh *datagramHost) SendDatagram(ctx context.Context, p peer.ID, data []byte) error {
	// Try to find an existing datagram-capable connection
	if dgConn := dh.GetDatagramConn(p); dgConn != nil {
		return dgConn.SendDatagram(ctx, data)
	}
	
	// No existing connection, try to establish one
	conn, err := dh.Network().DialPeer(ctx, p)
	if err != nil {
		return err
	}
	
	if dcConn, ok := conn.(network.DatagramCapableConn); ok && dcConn.SupportsDatagrams() {
		dgConn := dcConn.AsDatagramConn()
		if dgConn != nil {
			// Set the global handler on the new connection
			dh.handlerMu.RLock()
			handler := dh.handler
			dh.handlerMu.RUnlock()
			
			if handler != nil {
				dgConn.SetDatagramHandler(handler)
			}
			
			return dgConn.SendDatagram(ctx, data)
		}
	}
	
	return ErrDatagramNotSupported
}

func (dh *datagramHost) GetDatagramConn(p peer.ID) network.DatagramConn {
	conns := dh.Network().ConnsToPeer(p)
	for _, conn := range conns {
		if dcConn, ok := conn.(network.DatagramCapableConn); ok && dcConn.SupportsDatagrams() {
			if dgConn := dcConn.AsDatagramConn(); dgConn != nil {
				return dgConn
			}
		}
	}
	return nil
}

var ErrDatagramNotSupported = errors.New("datagram not supported by any available connection")