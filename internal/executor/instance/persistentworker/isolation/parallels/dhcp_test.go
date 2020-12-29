package parallels_test

import (
	"github.com/cirruslabs/cirrus-cli/internal/executor/instance/persistentworker/isolation/parallels"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestLeases(t *testing.T) {
	snooper := parallels.DHCPSnooper{DHCPLeasesFile: "testdata/parallels_dhcp_leases"}
	leases, err := snooper.Leases()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []*parallels.DHCPLease{
		{MAC: []byte{0xFE, 0xED, 0xFA, 0xCE, 0xFE, 0xED}, IP: "192.168.0.1", Start: 0},
		{MAC: []byte{0x00, 0x00, 0x00, 0x11, 0x11, 0x11}, IP: "10.0.10.50", Start: 6400},
		{MAC: []byte{0x00, 0x00, 0x00, 0x11, 0x11, 0x11}, IP: "10.0.10.50", Start: 16400},
		{MAC: []byte{0x00, 0x00, 0x00, 0x11, 0x11, 0x11}, IP: "10.0.10.50", Start: 11400},
	}, leases)
}

func TestFindNewestLease(t *testing.T) {
	snooper := parallels.DHCPSnooper{DHCPLeasesFile: "testdata/parallels_dhcp_leases"}
	lease, err := snooper.FindNewestLease([]byte{0x00, 0x00, 0x00, 0x11, 0x11, 0x11})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, &parallels.DHCPLease{
		MAC:   []byte{0x00, 0x00, 0x00, 0x11, 0x11, 0x11},
		IP:    "10.0.10.50",
		Start: 16400,
	}, lease)
}
