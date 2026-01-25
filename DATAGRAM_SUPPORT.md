# Datagram Support in libp2p

This document describes the datagram support extension for libp2p, which enables unreliable packet transmission suitable for applications like VPN, gaming, and real-time communication.

## Overview

Traditional libp2p focuses on reliable stream-based communication. However, certain applications require unreliable, low-latency packet transmission. This extension adds datagram support while maintaining compatibility with existing libp2p APIs.

## Architecture

### Core Interfaces

1. **DatagramConn**: Represents a datagram-capable connection
2. **DatagramCapableConn**: Extends CapableConn with datagram capabilities  
3. **DatagramNetwork**: Extends Network with datagram operations
4. **DatagramTransport**: Indicates transport-level datagram support

### Supported Transports

- **QUIC**: Native datagram support via QUIC datagrams (RFC 9221)
- **WebRTC**: Datagram support via unordered data channels
- **TCP/WebSocket**: Not supported (stream-only)

## Usage Examples

### Basic Datagram Transmission

```go
// Create a datagram-capable host
h, err := libp2p.New(
    libp2p.Transport(quic.NewTransport),
)
if err != nil {
    log.Fatal(err)
}

// Convert to datagram host
datagramHost := basichost.NewDatagramHost(h.(*basichost.BasicHost))

// Set up datagram handler
datagramHost.SetDatagramHandler(func(data []byte, from peer.ID, localAddr, remoteAddr ma.Multiaddr) {
    fmt.Printf("Received datagram from %s: %s\n", from, string(data))
})

// Send datagram to peer
err = datagramHost.SendDatagram(context.Background(), peerID, []byte("Hello, World!"))
if err != nil {
    log.Printf("Failed to send datagram: %v", err)
}
```

### VPN Application

```go
// Set up VPN packet handler
datagramHost.SetDatagramHandler(func(data []byte, from peer.ID, localAddr, remoteAddr ma.Multiaddr) {
    // Parse IP packet
    if len(data) < 20 {
        return // Invalid packet
    }
    
    // Extract destination IP
    destIP := net.IPv4(data[16], data[17], data[18], data[19])
    
    // Route packet based on destination
    if shouldForwardToTUN(destIP) {
        tunInterface.Write(data)
    } else {
        // Forward to another peer
        nextPeer := findNextHop(destIP)
        datagramHost.SendDatagram(context.Background(), nextPeer, data)
    }
})
```

### Gaming Application

```go
// Game state update
type GameUpdate struct {
    PlayerID   string
    Position   [3]float64
    Timestamp  int64
}

// Send game update
update := GameUpdate{
    PlayerID:  "player1",
    Position:  [3]float64{10.5, 20.3, 5.1},
    Timestamp: time.Now().UnixNano(),
}

data, _ := json.Marshal(update)
err = datagramHost.SendDatagram(context.Background(), peerID, data)
```

## Implementation Details

### QUIC Datagram Support

QUIC datagrams provide:
- Unreliable delivery (no retransmission)
- In-order delivery within the same connection
- Congestion control
- Encryption and authentication
- MTU discovery

```go
// QUIC datagram connection
type datagramConn struct {
    quicConn        *quic.Conn
    localPeer       peer.ID
    remotePeerID    peer.ID
    // ... other fields
}

func (dc *datagramConn) SendDatagram(ctx context.Context, data []byte) error {
    return dc.quicConn.SendDatagram(data)
}
```

### WebRTC Datagram Support

WebRTC datagrams use unordered data channels:
- Unreliable delivery
- No ordering guarantees
- Lower latency than ordered channels
- Built-in NAT traversal

```go
// WebRTC datagram channel
datagramChannel, err := pc.CreateDataChannel("libp2p-datagram", &webrtc.DataChannelInit{
    Ordered: &[]bool{false}[0], // Unordered for datagram semantics
})
```

## Performance Characteristics

### Latency
- **QUIC**: ~1-5ms additional latency over UDP
- **WebRTC**: ~5-20ms depending on ICE negotiation

### Throughput
- **QUIC**: Limited by congestion control, typically 80-95% of available bandwidth
- **WebRTC**: Limited by SCTP buffer sizes, typically 50-80% of available bandwidth

### MTU
- **QUIC**: ~1200 bytes (path MTU discovery)
- **WebRTC**: ~1200 bytes (conservative estimate)

## Error Handling

Datagram transmission can fail for various reasons:

```go
err := datagramHost.SendDatagram(ctx, peerID, data)
switch {
case errors.Is(err, ErrDatagramNotSupported):
    // Peer doesn't support datagrams, fall back to streams
    stream, err := host.NewStream(ctx, peerID, "/my-protocol/1.0.0")
    // ... use stream instead
    
case errors.Is(err, ErrDatagramTooLarge):
    // Packet exceeds MTU, fragment or compress
    fragments := fragmentPacket(data, datagramConn.DatagramMTU())
    for _, fragment := range fragments {
        datagramHost.SendDatagram(ctx, peerID, fragment)
    }
    
case errors.Is(err, context.DeadlineExceeded):
    // Timeout, retry or drop packet
    log.Printf("Datagram send timeout")
}
```

## Migration Guide

### From Stream-based to Datagram

1. **Identify suitable use cases**: Real-time data, where packet loss is acceptable
2. **Update connection handling**: Check for datagram support
3. **Implement packet fragmentation**: Handle MTU limitations
4. **Add error handling**: Handle unreliable delivery

### Hybrid Approach

Many applications benefit from using both streams and datagrams:

```go
// Use streams for reliable control messages
controlStream, err := host.NewStream(ctx, peerID, "/control/1.0.0")

// Use datagrams for real-time data
datagramHost.SendDatagram(ctx, peerID, realtimeData)
```

## Testing

Run datagram-specific tests:

```bash
go test ./p2p/transport/quic -run TestDatagram
go test ./p2p/transport/webrtc -run TestDatagram
go test ./examples/datagram-vpn
```

## Limitations

1. **Transport Support**: Only QUIC and WebRTC support datagrams
2. **Reliability**: No delivery guarantees
3. **Ordering**: No ordering guarantees (WebRTC)
4. **MTU**: Limited packet size (~1200 bytes)
5. **Congestion Control**: May compete with streams for bandwidth

## Future Enhancements

1. **UDP Transport**: Direct UDP transport for maximum performance
2. **Multicast Support**: One-to-many datagram transmission
3. **QoS Integration**: Priority-based datagram handling
4. **Compression**: Automatic packet compression for small MTUs
5. **Fragmentation**: Automatic large packet fragmentation

## Contributing

When contributing datagram-related features:

1. Ensure backward compatibility with existing stream APIs
2. Add comprehensive tests for both reliable and unreliable scenarios
3. Document performance characteristics
4. Consider security implications of unreliable transmission
5. Update relevant examples and documentation

## References

- [RFC 9221: An Unreliable Datagram Extension to QUIC](https://tools.ietf.org/rfc/rfc9221.txt)
- [WebRTC Data Channels](https://developer.mozilla.org/en-US/docs/Web/API/WebRTC_API/Using_data_channels)
- [libp2p Transport Specification](https://github.com/libp2p/specs/tree/master/transport)