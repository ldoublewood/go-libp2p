package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	basichost "github.com/libp2p/go-libp2p/p2p/host/basic"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"

	ma "github.com/multiformats/go-multiaddr"
)

// VPNNode represents a VPN node that can route packets
type VPNNode struct {
	host     network.DatagramNetwork
	tunIface *TUNInterface
	peers    map[peer.ID]ma.Multiaddr
}

// TUNInterface represents a TUN network interface (simplified)
type TUNInterface struct {
	name string
	// In a real implementation, this would contain actual TUN interface handling
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: datagram-vpn <listen-port> [peer-multiaddr]")
		os.Exit(1)
	}

	port := os.Args[1]
	
	// Create a libp2p host with QUIC transport (supports datagrams)
	h, err := libp2p.New(
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/udp/%s/quic-v1", port)),
		libp2p.Transport(quic.NewTransport),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer h.Close()

	// Convert to datagram-capable host
	basicHost := h.(*basichost.BasicHost)
	datagramHost := basichost.NewDatagramHost(basicHost)

	// Create VPN node
	vpnNode := &VPNNode{
		host:  datagramHost,
		peers: make(map[peer.ID]ma.Multiaddr),
	}

	// Set up datagram handler for incoming VPN packets
	datagramHost.SetDatagramHandler(vpnNode.handleVPNPacket)

	fmt.Printf("VPN Node started. Peer ID: %s\n", h.ID())
	fmt.Printf("Listening on: %v\n", h.Addrs())

	// Connect to peer if provided
	if len(os.Args) > 2 {
		peerAddr, err := ma.NewMultiaddr(os.Args[2])
		if err != nil {
			log.Fatal(err)
		}
		
		peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
		if err != nil {
			log.Fatal(err)
		}

		err = h.Connect(context.Background(), *peerInfo)
		if err != nil {
			log.Fatal(err)
		}
		
		vpnNode.peers[peerInfo.ID] = peerAddr
		fmt.Printf("Connected to peer: %s\n", peerInfo.ID)
		
		// Send a test VPN packet
		testPacket := []byte("VPN-TEST-PACKET")
		err = datagramHost.SendDatagram(context.Background(), peerInfo.ID, testPacket)
		if err != nil {
			log.Printf("Failed to send test packet: %v", err)
		} else {
			fmt.Println("Sent test VPN packet")
		}
	}

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Println("VPN Node shutting down...")
}

// handleVPNPacket processes incoming VPN packets
func (vpn *VPNNode) handleVPNPacket(data []byte, from peer.ID, localAddr, remoteAddr ma.Multiaddr) {
	fmt.Printf("Received VPN packet from %s: %s\n", from, string(data))
	
	// In a real VPN implementation, this would:
	// 1. Parse the IP packet
	// 2. Check routing table
	// 3. Forward to appropriate interface (TUN or another peer)
	// 4. Handle NAT/firewall rules
	
	if string(data) == "VPN-TEST-PACKET" {
		// Echo back a response
		response := []byte("VPN-TEST-RESPONSE")
		err := vpn.host.SendDatagram(context.Background(), from, response)
		if err != nil {
			log.Printf("Failed to send response: %v", err)
		} else {
			fmt.Println("Sent test response")
		}
	}
}

// Example of how VPN packet routing would work
func (vpn *VPNNode) routePacket(packet []byte) error {
	// Parse IP packet to get destination
	if len(packet) < 20 {
		return fmt.Errorf("packet too short")
	}
	
	// Extract destination IP (simplified IPv4 parsing)
	destIP := net.IPv4(packet[16], packet[17], packet[18], packet[19])
	
	// Look up peer responsible for this IP range
	targetPeer := vpn.findPeerForIP(destIP)
	if targetPeer == "" {
		return fmt.Errorf("no route to %s", destIP)
	}
	
	// Forward packet to peer via datagram
	return vpn.host.SendDatagram(context.Background(), targetPeer, packet)
}

func (vpn *VPNNode) findPeerForIP(ip net.IP) peer.ID {
	// In a real implementation, this would consult a routing table
	// For now, just return the first peer
	for peerID := range vpn.peers {
		return peerID
	}
	return ""
}

// Example of TUN interface integration (pseudo-code)
func (vpn *VPNNode) setupTUNInterface() error {
	// This would create and configure a TUN interface
	// tun, err := water.New(water.Config{
	//     DeviceType: water.TUN,
	// })
	// if err != nil {
	//     return err
	// }
	
	// Configure IP address and routing
	// exec.Command("ip", "addr", "add", "10.0.0.1/24", "dev", tun.Name()).Run()
	// exec.Command("ip", "link", "set", "dev", tun.Name(), "up").Run()
	
	// Start packet forwarding loop
	// go vpn.tunForwardLoop(tun)
	
	return nil
}

// tunForwardLoop would read packets from TUN and forward via libp2p datagrams
func (vpn *VPNNode) tunForwardLoop(tun interface{}) {
	// buffer := make([]byte, 1500)
	// for {
	//     n, err := tun.Read(buffer)
	//     if err != nil {
	//         continue
	//     }
	//     
	//     packet := buffer[:n]
	//     err = vpn.routePacket(packet)
	//     if err != nil {
	//         log.Printf("Failed to route packet: %v", err)
	//     }
	// }
}