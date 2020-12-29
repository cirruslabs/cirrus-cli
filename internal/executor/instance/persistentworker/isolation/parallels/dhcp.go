package parallels

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
)

const DefaultDHCPLeasesFile = "/Library/Preferences/Parallels/parallels_dhcp_leases"

var leaseRegex = regexp.MustCompile("(.*)=\"(.*),(.*),(.*),(.*)\"")

var ErrDHCPSnoopFailed = errors.New("failed to retrieve VM's DHCP address")

type DHCPSnooper struct {
	DHCPLeasesFile string
}

type DHCPLease struct {
	MAC   []byte
	IP    string
	Start int64
}

func (snooper *DHCPSnooper) dhcpLeasesFile() string {
	if snooper.DHCPLeasesFile != "" {
		return snooper.DHCPLeasesFile
	}

	return DefaultDHCPLeasesFile
}

func (snooper *DHCPSnooper) Leases() ([]*DHCPLease, error) {
	var result []*DHCPLease

	leases, err := ioutil.ReadFile(snooper.dhcpLeasesFile())
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read leases file %q: %v",
			ErrDHCPSnoopFailed, snooper.dhcpLeasesFile(), err)
	}

	matches := leaseRegex.FindAllStringSubmatch(string(leases), -1)

	for _, match := range matches {
		mac, err := hex.DecodeString(match[4])
		if err != nil {
			return nil, fmt.Errorf("%w: failed to decode lease MAC-address %q: %v",
				ErrDHCPSnoopFailed, match[4], err)
		}

		expirationTimestamp, err := strconv.ParseInt(match[2], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to parse expiration timestamp %q: %v",
				ErrDHCPSnoopFailed, match[2], err)
		}

		leaseTimeSec, err := strconv.ParseInt(match[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to parse lease time %q: %v",
				ErrDHCPSnoopFailed, match[3], err)
		}

		result = append(result, &DHCPLease{
			MAC:   mac,
			IP:    match[1],
			Start: expirationTimestamp - leaseTimeSec,
		})
	}

	return result, nil
}

func (snooper *DHCPSnooper) FindNewestLease(mac []byte) (*DHCPLease, error) {
	leases, err := snooper.Leases()
	if err != nil {
		return nil, err
	}

	// Sort leases by lease start time, ascending
	sort.Slice(leases, func(i, j int) bool {
		return leases[i].Start < leases[j].Start
	})

	// Try to find the requested lease, starting from the freshest leases
	for i := len(leases) - 1; i >= 0; i-- {
		lease := leases[i]

		if bytes.Equal(lease.MAC, mac) {
			return lease, nil
		}
	}

	return nil, fmt.Errorf("%w: no lease found for MAC address %x", ErrDHCPSnoopFailed, mac)
}
