package state

import (
	"archive/tar"
	"bytes"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/idena-network/idena-go/common/math"
	idenastate "github.com/idena-network/idena-go/core/state"
	models "github.com/idena-network/idena-go/protobuf"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"
)

func TestReadTreeFromRejectsMalformedTar(t *testing.T) {
	err := readTreeFrom(testPrefixDB(), 1, bytes.NewBufferString("not a tar archive"))

	require.Error(t, err)
}

func TestReadTreeFromRejectsOversizedSnapshotChunk(t *testing.T) {
	var input bytes.Buffer
	tw := tar.NewWriter(&input)
	require.NoError(t, tw.WriteHeader(&tar.Header{
		Name: "0",
		Mode: 0600,
		Size: idenastate.MaxSnapshotChunkBytes + 1,
	}))

	err := readTreeFrom(testPrefixDB(), 1, &input)

	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds limit")
}

func TestReadTreeFromRejectsOutOfRangeNodeHeight(t *testing.T) {
	payload, err := proto.Marshal(&models.ProtoSnapshotNodes{
		Nodes: []*models.ProtoSnapshotNodes_Node{{Height: math.MaxInt8 + 1}},
	})
	require.NoError(t, err)

	var input bytes.Buffer
	tw := tar.NewWriter(&input)
	require.NoError(t, tw.WriteHeader(&tar.Header{Name: "0", Mode: 0600, Size: int64(len(payload))}))
	_, err = tw.Write(payload)
	require.NoError(t, err)
	require.NoError(t, tw.Close())

	err = readTreeFrom(testPrefixDB(), 1, &input)

	require.Error(t, err)
	require.Contains(t, err.Error(), "height")
}

func testPrefixDB() *db.PrefixDB {
	return db.NewPrefixDB(db.NewMemDB(), []byte("snapshot"))
}
