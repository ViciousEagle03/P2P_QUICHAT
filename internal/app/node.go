package app

import (
	"context"
	"fmt"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	host "github.com/libp2p/go-libp2p/core/host"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	ma "github.com/multiformats/go-multiaddr"
)

// Node encapsulates a libp2p host with DHT and PubSub functionality.
type Node struct {
	ctx           context.Context
	nick, port    string
	bootstrapAddr string

	Host   host.Host
	DHT    *dht.IpfsDHT
	PubSub *pubsub.PubSub
	Topic  *pubsub.Topic
	Sub    *pubsub.Subscription
}

// NewNode constructs and initializes a Node.
func NewNode(ctx context.Context, nick, port, bootstrapAddr string) (*Node, error) {
	n := &Node{ctx: ctx, nick: nick, port: port, bootstrapAddr: bootstrapAddr}

	// Step-by-step initialization
	if err := n.initHost(); err != nil {
		return nil, err
	}
	if err := n.initDHT(); err != nil {
		return nil, err
	}
	if err := n.connectBootstrapPeer(); err != nil {
		return nil, err
	}
	if err := n.initPubSub(); err != nil {
		return nil, err
	}
	n.registerJoinNotifier()
	n.printReachableAddr()
	n.printWelcomeBanner()

	return n, nil
}

// initHost sets up the libp2p Host with AutoRelay.
func (n *Node) initHost() error {
	var err error
	n.Host, err = libp2p.New(
		libp2p.ListenAddrStrings(
			"/ip4/0.0.0.0/tcp/"+n.port,
			"/ip4/0.0.0.0/udp/"+n.port+"/quic-v1",
		),
		libp2p.EnableAutoRelayWithPeerSource(n.relayCandidates),
	)
	return err
}

// relayCandidates provides peers from the DHT routing table for AutoRelay.
func (n *Node) relayCandidates(ctx context.Context, num int) <-chan peer.AddrInfo {
	ch := make(chan peer.AddrInfo, num)
	go func() {
		defer close(ch)
		if n.DHT == nil {
			return
		}
		for _, pid := range n.DHT.RoutingTable().ListPeers() {
			addrs := n.Host.Peerstore().Addrs(pid)
			ch <- peer.AddrInfo{ID: pid, Addrs: addrs}
		}
	}()
	return ch
}

// initDHT creates and bootstraps the DHT.
func (n *Node) initDHT() error {
	var err error
	n.DHT, err = dht.New(n.ctx, n.Host, dht.Mode(dht.ModeAuto))
	if err != nil {
		return err
	}
	return n.DHT.Bootstrap(n.ctx)
}

// connectBootstrapPeer attempts a timed dial to the bootstrap node.
func (n *Node) connectBootstrapPeer() error {
	if n.bootstrapAddr == "" {
		return nil
	}
	maddr, err := ma.NewMultiaddr(n.bootstrapAddr)
	if err != nil {
		return fmt.Errorf("invalid bootstrap multiaddr %q: %w", n.bootstrapAddr, err)
	}
	info, err := peer.AddrInfoFromP2pAddr(maddr)
	fmt.Println(info)
	if err != nil {
		return fmt.Errorf("invalid bootstrap multiaddr %q: %w", n.bootstrapAddr, err)
	}
	dialCtx, cancel := context.WithTimeout(n.ctx, 20*time.Second)
	fmt.Println(dialCtx)
	defer cancel()

	return n.Host.Connect(dialCtx, *info)
}

// initPubSub sets up GossipSub and subscribes to the global topic.
func (n *Node) initPubSub() error {
	var err error
	n.PubSub, err = pubsub.NewGossipSub(n.ctx, n.Host)
	if err != nil {
		return err
	}
	n.Topic, err = n.PubSub.Join("peerchat:global")
	if err != nil {
		return err
	}
	n.Sub, err = n.Topic.Subscribe()
	return err
}

// registerJoinNotifier publishes a "joined" message on new connections.
func (n *Node) registerJoinNotifier() {
	n.Host.Network().Notify(&network.NotifyBundle{
		ConnectedF: func(net network.Network, conn network.Conn) {
		},
	})
}

// printReachableAddr outputs one of the host's listen addresses.
func (n *Node) printReachableAddr() {
	addr := n.Host.Addrs()[0]
	fmt.Printf("Your multiaddr: %s/p2p/%s\n", addr, n.Host.ID().String())
}

func (n *Node) printWelcomeBanner() {
	banner := `
  ___ ___ ___    ___  _   _ ___ ___ _  _   _ _____ 
 | _ |_  | _ \  / _ \| | | |_ _/ __| || | /_|_   _|
 |  _// /|  _/ | (_) | |_| || | (__| __ |/ _ \| |  
 |_| /___|_|    \__\_\\___/|___\___|_||_/_/ \_|_|  
                                                                                                                                                                                                                                                   
`
	fmt.Println(banner)
	fmt.Println("Welcome to P2P Quichat! 🚀")
}
