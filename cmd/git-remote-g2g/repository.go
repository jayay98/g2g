package main

import (
	"fmt"
	"regexp"

	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	ma "github.com/multiformats/go-multiaddr"
)

type Repository struct {
	address *peer.AddrInfo
	id      string
}

func NewRepository(address string) (repo *Repository, err error) {
	peerAddr, repoId, err := parseRemoteAddr(address)
	if err != nil {
		return
	}

	multiaddr, err := ma.NewMultiaddr(peerAddr)
	if err != nil {
		return
	}

	info, err := peer.AddrInfoFromP2pAddr(multiaddr)
	if err != nil {
		return
	}

	repo = &Repository{address: info, id: repoId}
	return
}

func (r *Repository) AddAddressInto(node host.Host) {
	node.Peerstore().AddAddrs(r.address.ID, r.address.Addrs, peerstore.PermanentAddrTTL)
}

func parseRemoteAddr(addr string) (ma string, repoId string, err error) {
	re, _ := regexp.Compile(`^g2g:\/\/(?P<ma>(\/[\w\.-]+)+)\/(?P<repoId>[\w_-]+\.git)$`)
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
	ma = result["ma"]
	return
}
