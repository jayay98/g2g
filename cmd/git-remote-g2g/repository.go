package main

import (
	"context"
	"fmt"
	"regexp"
	"time"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
)

type Repository struct {
	addrs []peer.AddrInfo
	id    string
}

func NewRepository(kdht *dht.IpfsDHT, address string) (repo *Repository, err error) {
	peerId, repoId, err := parseRemoteAddr(address)
	if err != nil {
		return
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	peerAddr, err := kdht.FindPeer(timeoutCtx, peerId)
	if err != nil {
		return
	}
	logger.Infof("Remote addr found: %v", peerAddr)

	pid := []peer.ID{peerId}
	info := peerstore.AddrInfos(kdht.Host().Peerstore(), pid)

	repo = &Repository{addrs: info, id: repoId}
	return
}

func (r *Repository) AddAddressInto(node host.Host) {
	for _, addr := range r.addrs {
		node.Peerstore().AddAddrs(addr.ID, addr.Addrs, peerstore.PermanentAddrTTL)
	}
}

func parseRemoteAddr(addr string) (peerId peer.ID, repoId string, err error) {
	re, _ := regexp.Compile(`^g2g:\/\/(?P<peerId>\w+)\/(?P<repoId>[\w_-]+\.git)$`)
	if !re.MatchString(addr) {
		err = fmt.Errorf("remote address does not end with \".git\"")
		return
	}

	match := re.FindStringSubmatch(addr)
	result := make(map[string]string)
	for i, name := range re.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}

	repoId = result["repoId"]
	peerId, err = peer.Decode(result["peerId"])
	return
}
