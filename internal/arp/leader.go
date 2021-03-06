package arp

import (
	"time"

	"github.com/mdlayher/ethernet"
)

// Leader returns true if we are the leader in the daemon set.
func (a *Announce) Leader() bool {
	a.leaderMu.RLock()
	defer a.leaderMu.RUnlock()
	return a.leader
}

// SetLeader sets the leader boolean to b.
func (a *Announce) SetLeader(b bool) {
	a.leaderMu.Lock()
	defer a.leaderMu.Unlock()
	a.leader = b
	if a.leader {
		go a.Acquire()
	} else {
		go a.Relinquish()
	}
}

// Relinquish set the leader bit to false and stops the go-routine that sends unsolicited APR replies.
func (a *Announce) Relinquish() {
	a.stop <- true
}

// Acquire sends out a unsolicited ARP replies for all VIPs that should be announced.
func (a *Announce) Acquire() {
	go a.spam()
	a.unsolicited()
}

// spam broadcasts unsolicited ARP replies for 5 seconds.
func (a *Announce) spam() {
	start := time.Now()
	for time.Since(start) < 5*time.Second {

		if !a.Leader() {
			return
		}

		for _, u := range a.Packets() {
			a.client.WriteTo(u, ethernet.Broadcast)
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// unsolicited sends unsolicited ARP replies every 10 seconds.
func (a *Announce) unsolicited() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// Double check that we're still master.
			if !a.Leader() {
				continue
			}
			for _, u := range a.Packets() {
				a.client.WriteTo(u, ethernet.Broadcast)
			}

		case <-a.stop:
			return
		}
	}
}
