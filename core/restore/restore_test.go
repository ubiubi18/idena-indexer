package restore

import (
	"github.com/idena-network/idena-go/blockchain"
	"github.com/idena-network/idena-indexer/db"
	"github.com/stretchr/testify/require"
	"testing"
)

type lastHeightAccessor struct {
	db.Accessor
	height uint64
	err    error
}

func (a *lastHeightAccessor) GetLastHeight() (uint64, error) {
	return a.height, a.err
}

func TestCollectDataUsesLastIndexedBlockForAddressProvenance(t *testing.T) {
	t.Chdir(t.TempDir())
	chain, appState, _, _ := blockchain.NewTestBlockchain(true, nil)
	accessor := &lastHeightAccessor{height: 6}
	restorer := NewRestorer(accessor, appState, chain.Blockchain)

	data, err := restorer.collectData()

	require.NoError(t, err)
	require.Equal(t, uint64(6), data.BlockHeight)
}
