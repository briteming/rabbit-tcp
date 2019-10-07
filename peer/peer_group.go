package peer

import (
	"context"
	"github.com/ihciah/rabbit-tcp/tunnel"
	"github.com/ihciah/rabbit-tcp/tunnel_pool"
	"log"
	"net"
	"os"
	"sync"
)

type PeerGroup struct {
	lock        sync.Mutex
	cipher      tunnel.Cipher
	peerMapping map[uint32]*ServerPeer
	logger      *log.Logger
}

func NewPeerGroup(cipher tunnel.Cipher) PeerGroup {
	if initRand() != nil {
		panic("Error when initialize random seed.")
	}
	return PeerGroup{
		cipher:      cipher,
		peerMapping: make(map[uint32]*ServerPeer),
		logger:      log.New(os.Stdout, "[PeerGroup]", log.LstdFlags),
	}
}

// Add a tunnel to it's peer; will create peer if not exists
func (pg *PeerGroup) AddTunnel(tunnel *tunnel_pool.Tunnel) error {
	pg.logger.Println("AddTunnel called.")
	// add tunnel to peer(if absent, create peer to peer_group)
	pg.lock.Lock()
	var peer *ServerPeer
	var ok bool

	peerID := tunnel.GetPeerID()
	if peer, ok = pg.peerMapping[peerID]; !ok {
		peerContext, removePeerFunc := context.WithCancel(context.Background())
		serverPeer := NewServerPeerWithID(peerID, peerContext, removePeerFunc)
		peer = &serverPeer
		pg.peerMapping[peerID] = peer

		go func() {
			<-peerContext.Done()
			pg.RemovePeer(peerID)
		}()
	}
	pg.lock.Unlock()
	peer.tunnelPool.AddTunnel(tunnel)
	return nil
}

// Like AddTunnel, add a raw connection
func (pg *PeerGroup) AddTunnelFromConn(conn net.Conn) error {
	pg.logger.Println("AddTunnelFromConn called.")
	tun, err := tunnel_pool.NewPassiveTunnel(conn, pg.cipher)
	if err != nil {
		return err
	}
	return pg.AddTunnel(&tun)
}

func (pg *PeerGroup) RemovePeer(peerID uint32) {
	pg.logger.Printf("Peer %d removed from peer group.\n", peerID)
	pg.lock.Lock()
	defer pg.lock.Unlock()
	delete(pg.peerMapping, peerID)
}
