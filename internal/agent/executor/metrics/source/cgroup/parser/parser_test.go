package parser

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseSingleValueFile(t *testing.T) {
	memoryUsageInBytesFile := `252387328
`

	result, err := ParseSingleValueFile(bytes.NewBufferString(memoryUsageInBytesFile))
	if err != nil {
		t.Fatal(err)
	}

	assert.EqualValues(t, 252387328, result)
}

func TestParseKeyValueFile(t *testing.T) {
	memoryStatFile := `cache 25182208
rss 2654208
rss_huge 0
shmem 0
mapped_file 9191424
dirty 0
writeback 0
pgpgin 46629
pgpgout 39828
pgfault 32208
pgmajfault 99
inactive_anon 8192
active_anon 2625536
inactive_file 4861952
active_file 20307968
unevictable 0
hierarchical_memory_limit 9223372036854771712
total_cache 227909632
total_rss 24612864
total_rss_huge 0
total_shmem 4055040
total_mapped_file 30547968
total_dirty 0
total_writeback 0
total_pgpgin 205821
total_pgpgout 144020
total_pgfault 257202
total_pgmajfault 297
total_inactive_anon 4108288
total_active_anon 24592384
total_inactive_file 165294080
total_active_file 60411904
total_unevictable 0
`

	result, err := ParseKeyValueFile(bytes.NewBufferString(memoryStatFile))
	if err != nil {
		t.Fatal(err)
	}

	totalInactiveFile, ok := result["total_inactive_file"]
	assert.True(t, ok)
	assert.EqualValues(t, totalInactiveFile, 165294080)
}
