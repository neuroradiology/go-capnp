// Package testnetwork provides an in-memory implementation of rpc.Network for testing purposes.
package testnetwork

import (
	"context"
	"net"
	"sync"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/exp/spsc"
	"capnproto.org/go/capnp/v3/rpc"
)

// PeerID is the implementation of peer ids used by a test network
type PeerID uint64

type edge struct {
	To, From PeerID
}

func (e edge) Flip() edge {
	return edge{
		To:   e.From,
		From: e.To,
	}
}

// This test network uses the same set of options for all
// participants. The rpc.Options instance can be cloned
// without issue.
type network struct {
	myID    PeerID
	options rpc.Options
	global  *Joiner
}

// A Joiner is a global view of a test network, which can be joined by a
// peer to acquire a Network.
type Joiner struct {
	mu          sync.Mutex
	nextID      PeerID
	nextNonce   uint64
	connections map[edge]*connectionEntry
	incoming    map[PeerID]spsc.Queue[PeerID]
}

type connectionEntry struct {
	Transport rpc.Transport
	Conn      *rpc.Conn // Might be nil, if we haven't initialized this yet.
}

func NewJoiner() *Joiner {
	return &Joiner{
		connections: make(map[edge]*connectionEntry),
	}
}

func (j *Joiner) Join(opts *rpc.Options) rpc.Network {
	j.mu.Lock()
	defer j.mu.Unlock()
	ret := network{
		myID:    j.nextID,
		global:  j,
		options: *opts,
	}
	j.nextID++
	return ret
}

func (j *Joiner) getAcceptQueue(id PeerID) spsc.Queue[PeerID] {
	q, ok := j.incoming[id]
	if !ok {
		q = spsc.New[PeerID]()
		j.incoming[id] = q
	}
	return q
}

func (n network) LocalID() rpc.PeerID {
	return rpc.PeerID{Value: n.myID}
}

func (n network) Dial(dst rpc.PeerID) (*rpc.Conn, error) {
	opts := n.options
	opts.Network = n
	opts.RemotePeerID = dst
	dstID := dst.Value.(PeerID)
	toEdge := edge{
		From: n.myID,
		To:   dstID,
	}
	fromEdge := toEdge.Flip()

	n.global.mu.Lock()
	defer n.global.mu.Unlock()
	ent, ok := n.global.connections[toEdge]
	if !ok {
		c1, c2 := net.Pipe()
		t1 := rpc.NewStreamTransport(c1)
		t2 := rpc.NewStreamTransport(c2)
		ent = &connectionEntry{Transport: t1}
		n.global.connections[toEdge] = ent
		n.global.connections[fromEdge] = &connectionEntry{Transport: t2}

	}
	if ent.Conn == nil {
		ent.Conn = rpc.NewConn(ent.Transport, &opts)
	} else {
		// There's already a connection, so we're not going to use this, but
		// we own it. So drop it:
		opts.BootstrapClient.Release()
	}
	return ent.Conn, nil
}

func (n network) Serve(ctx context.Context) error {
	n.global.mu.Lock()
	q := n.global.getAcceptQueue(n.myID)
	n.global.mu.Unlock()

	for {
		incoming, err := q.Recv(ctx)
		if err != nil {
			return err
		}
		opts := n.options
		opts.Network = n
		opts.RemotePeerID = rpc.PeerID{incoming}
		n.global.mu.Lock()
		defer n.global.mu.Unlock()
		edge := edge{
			From: n.myID,
			To:   incoming,
		}
		ent := n.global.connections[edge]
		if ent.Conn == nil {
			ent.Conn = rpc.NewConn(ent.Transport, &opts)
		} else {
			opts.BootstrapClient.Release()
		}
	}
}

func (n network) Introduce(provider, recipient *rpc.Conn) (rpc.IntroductionInfo, error) {
	providerPeer := provider.RemotePeerID()
	recipientPeer := recipient.RemotePeerID()
	n.global.mu.Lock()
	defer n.global.mu.Unlock()
	nonce := n.global.nextNonce
	n.global.nextNonce++
	_, seg := capnp.NewSingleSegmentMessage(nil)
	ret := rpc.IntroductionInfo{}
	sendToRecipient, err := NewPeerAndNonce(seg)
	if err != nil {
		return ret, err
	}
	sendToProvider, err := NewPeerAndNonce(seg)
	if err != nil {
		return ret, err
	}
	sendToRecipient.SetPeerId(uint64(providerPeer.Value.(PeerID)))
	sendToRecipient.SetNonce(nonce)
	sendToProvider.SetPeerId(uint64(recipientPeer.Value.(PeerID)))
	sendToProvider.SetNonce(nonce)
	ret.SendToRecipient = rpc.ThirdPartyToContact(sendToRecipient.ToPtr())
	ret.SendToProvider = rpc.ThirdPartyToAwait(sendToProvider.ToPtr())
	return ret, nil
}
