package libp2pquic

import (
	"context"
	"sync"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/quic-go/quic-go"
)

// datagramConn implements network.DatagramConn for QUIC connections
type datagramConn struct {
	quicConn        *quic.Conn
	localPeer       peer.ID
	remotePeerID    peer.ID
	remotePubKey    ic.PubKey
	localMultiaddr  ma.Multiaddr
	remoteMultiaddr ma.Multiaddr

	handlerMu sync.RWMutex
	handler   network.DatagramHandler

	ctx    context.Context
	cancel context.CancelFunc
}

func newDatagramConn(qconn *quic.Conn, localPeer peer.ID, remotePeerID peer.ID, remotePubKey ic.PubKey, localAddr, remoteAddr ma.Multiaddr) *datagramConn {
	ctx, cancel := context.WithCancel(context.Background())
	dc := &datagramConn{
		quicConn:        qconn,
		localPeer:       localPeer,
		remotePeerID:    remotePeerID,
		remotePubKey:    remotePubKey,
		localMultiaddr:  localAddr,
		remoteMultiaddr: remoteAddr,
		ctx:             ctx,
		cancel:          cancel,
	}

	// Start the receive loop if handler is set
	go dc.receiveLoop()

	return dc
}

func (dc *datagramConn) SendDatagram(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-dc.ctx.Done():
		return dc.ctx.Err()
	default:
	}

	return dc.quicConn.SendDatagram(data)
}

func (dc *datagramConn) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-dc.ctx.Done():
		return nil, dc.ctx.Err()
	default:
	}

	return dc.quicConn.ReceiveDatagram(ctx)
}

func (dc *datagramConn) SetDatagramHandler(handler network.DatagramHandler) {
	dc.handlerMu.Lock()
	dc.handler = handler
	dc.handlerMu.Unlock()
}

func (dc *datagramConn) DatagramMTU() int {
	// QUIC datagram MTU is typically around 1200 bytes to avoid fragmentation
	// This should be dynamically determined based on path MTU discovery
	return 4200
}

func (dc *datagramConn) Close() error {
	dc.cancel()
	return nil
}

func (dc *datagramConn) LocalPeer() peer.ID {
	return dc.localPeer
}

func (dc *datagramConn) RemotePeer() peer.ID {
	return dc.remotePeerID
}

func (dc *datagramConn) LocalMultiaddr() ma.Multiaddr {
	return dc.localMultiaddr
}

func (dc *datagramConn) RemoteMultiaddr() ma.Multiaddr {
	return dc.remoteMultiaddr
}

func (dc *datagramConn) receiveLoop() {
	log.Info("starting datagram receiveLoop", "localPeer", dc.localPeer, "remotePeer", dc.remotePeerID, "local", dc.localMultiaddr, "remote", dc.remoteMultiaddr)

	for {
		select {
		case <-dc.ctx.Done():
			log.Info("receiveLoop context cancelled, exiting", "localPeer", dc.localPeer, "remotePeer", dc.remotePeerID)
			return
		default:
		}

		log.Debug("waiting for datagram", "localPeer", dc.localPeer, "remotePeer", dc.remotePeerID)
		data, err := dc.quicConn.ReceiveDatagram(dc.ctx)
		if err != nil {
			if dc.ctx.Err() != nil {
				log.Info("exiting datagram conn receiveLoop", "err", dc.ctx.Err(), "localPeer", dc.localPeer, "remotePeer", dc.remotePeerID)
				return // Context cancelled, normal shutdown
			}
			log.Warn("Error receiving datagram", "err", err, "localPeer", dc.localPeer, "remotePeer", dc.remotePeerID, "local", dc.localMultiaddr, "remote", dc.remoteMultiaddr)
			continue
		}

		log.Info("received datagram", "dataLen", len(data), "localPeer", dc.localPeer, "remotePeer", dc.remotePeerID, "local", dc.localMultiaddr, "remote", dc.remoteMultiaddr)

		// Log first few bytes of data for debugging (be careful with sensitive data)
		if len(data) > 0 {
			previewLen := 32
			if len(data) < previewLen {
				previewLen = len(data)
			}
			log.Debug("datagram data preview", "dataPreview", data[:previewLen], "totalLen", len(data))
		}

		dc.handlerMu.RLock()
		handler := dc.handler
		dc.handlerMu.RUnlock()

		if handler != nil {
			log.Debug("dispatching datagram to handler", "dataLen", len(data), "remotePeer", dc.remotePeerID)
			go func() {
				defer func() {
					if r := recover(); r != nil {
						log.Error("datagram handler panicked", "panic", r, "remotePeer", dc.remotePeerID, "dataLen", len(data))
					}
				}()
				handler(data, dc.remotePeerID, dc.localMultiaddr, dc.remoteMultiaddr)
				log.Debug("datagram handler completed", "dataLen", len(data), "remotePeer", dc.remotePeerID)
			}()
		} else {
			log.Warn("received datagram but handler is not set", "dataLen", len(data), "remote", dc.remoteMultiaddr, "local", dc.localMultiaddr, "remotePeer", dc.remotePeerID)
		}
	}
}
