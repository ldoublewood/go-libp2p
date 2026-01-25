package libp2pwebrtc

import (
	"context"
	"errors"
	"sync"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"

	ma "github.com/multiformats/go-multiaddr"
	"github.com/pion/webrtc/v4"
)

// webrtcDatagramConn implements network.DatagramConn for WebRTC connections
type webrtcDatagramConn struct {
	pc              *webrtc.PeerConnection
	datagramChannel *webrtc.DataChannel
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

func newWebRTCDatagramConn(pc *webrtc.PeerConnection, localPeer peer.ID, remotePeerID peer.ID, remotePubKey ic.PubKey, localAddr, remoteAddr ma.Multiaddr) (*webrtcDatagramConn, error) {
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create a dedicated datagram channel
	datagramChannel, err := pc.CreateDataChannel("libp2p-datagram", &webrtc.DataChannelInit{
		Ordered: &[]bool{false}[0], // Unordered for datagram semantics
	})
	if err != nil {
		cancel()
		return nil, err
	}
	
	dc := &webrtcDatagramConn{
		pc:              pc,
		datagramChannel: datagramChannel,
		localPeer:       localPeer,
		remotePeerID:    remotePeerID,
		remotePubKey:    remotePubKey,
		localMultiaddr:  localAddr,
		remoteMultiaddr: remoteAddr,
		ctx:             ctx,
		cancel:          cancel,
	}
	
	// Set up message handler
	datagramChannel.OnMessage(func(msg webrtc.DataChannelMessage) {
		dc.handlerMu.RLock()
		handler := dc.handler
		dc.handlerMu.RUnlock()
		
		if handler != nil {
			go handler(msg.Data, dc.remotePeerID, dc.localMultiaddr, dc.remoteMultiaddr)
		}
	})
	
	return dc, nil
}

func (dc *webrtcDatagramConn) SendDatagram(ctx context.Context, data []byte) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-dc.ctx.Done():
		return dc.ctx.Err()
	default:
	}
	
	if dc.datagramChannel.ReadyState() != webrtc.DataChannelStateOpen {
		return ErrDataChannelNotOpen
	}
	
	return dc.datagramChannel.Send(data)
}

func (dc *webrtcDatagramConn) ReceiveDatagram(ctx context.Context) ([]byte, error) {
	// WebRTC uses event-driven model, so direct receive is not supported
	// Users should use SetDatagramHandler instead
	return nil, ErrReceiveNotSupported
}

func (dc *webrtcDatagramConn) SetDatagramHandler(handler network.DatagramHandler) {
	dc.handlerMu.Lock()
	dc.handler = handler
	dc.handlerMu.Unlock()
}

func (dc *webrtcDatagramConn) DatagramMTU() int {
	// WebRTC data channel MTU is typically 16KB, but we use a conservative value
	return 1200
}

func (dc *webrtcDatagramConn) Close() error {
	dc.cancel()
	if dc.datagramChannel != nil {
		return dc.datagramChannel.Close()
	}
	return nil
}

func (dc *webrtcDatagramConn) LocalPeer() peer.ID {
	return dc.localPeer
}

func (dc *webrtcDatagramConn) RemotePeer() peer.ID {
	return dc.remotePeerID
}

func (dc *webrtcDatagramConn) LocalMultiaddr() ma.Multiaddr {
	return dc.localMultiaddr
}

func (dc *webrtcDatagramConn) RemoteMultiaddr() ma.Multiaddr {
	return dc.remoteMultiaddr
}

var (
	ErrDataChannelNotOpen   = errors.New("data channel not open")
	ErrReceiveNotSupported  = errors.New("direct receive not supported, use SetDatagramHandler")
)