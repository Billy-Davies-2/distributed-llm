package network

import (
    "net"
    "sync"
    "time"
)

type Discovery struct {
    mu       sync.Mutex
    peers    map[string]time.Time
    interval time.Duration
}

func NewDiscovery(interval time.Duration) *Discovery {
    return &Discovery{
        peers:    make(map[string]time.Time),
        interval: interval,
    }
}

func (d *Discovery) Start() {
    go func() {
        for {
            d.mu.Lock()
            for peer := range d.peers {
                if time.Since(d.peers[peer]) > d.interval {
                    delete(d.peers, peer)
                }
            }
            d.mu.Unlock()
            time.Sleep(d.interval)
        }
    }()
}

func (d *Discovery) AddPeer(address string) {
    d.mu.Lock()
    defer d.mu.Unlock()
    d.peers[address] = time.Now()
}

func (d *Discovery) GetPeers() []string {
    d.mu.Lock()
    defer d.mu.Unlock()
    addresses := make([]string, 0, len(d.peers))
    for peer := range d.peers {
        addresses = append(addresses, peer)
    }
    return addresses
}

func (d *Discovery) Broadcast(address string) error {
    conn, err := net.Dial("udp", address)
    if err != nil {
        return err
    }
    defer conn.Close()
    // Implement broadcasting logic here
    return nil
}