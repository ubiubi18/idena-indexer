package state

import (
	"archive/tar"
	"bytes"
	"testing"

	nodeState "github.com/idena-network/idena-go/core/state"
	db "github.com/tendermint/tm-db"
)

func snapshotPrefixDB() *db.PrefixDB {
	return db.NewPrefixDB(db.NewMemDB(), []byte("snapshot-test"))
}

func TestReadTreeFromAcceptsNodeSnapshot(t *testing.T) {
	sourceDB := db.NewMemDB()
	sourceTree := nodeState.NewMutableTree(sourceDB)
	sourceTree.Set([]byte("identity"), []byte("state"))
	if _, _, err := sourceTree.SaveVersion(); err != nil {
		t.Fatal(err)
	}

	var archive bytes.Buffer
	if _, err := nodeState.WriteTreeTo2(sourceDB, 1, &archive); err != nil {
		t.Fatal(err)
	}
	targetDB := snapshotPrefixDB()
	if err := readTreeFrom(targetDB, 1, &archive); err != nil {
		t.Fatal(err)
	}
	targetTree := nodeState.NewMutableTree(targetDB)
	if _, err := targetTree.LoadVersion(1); err != nil {
		t.Fatal(err)
	}
	_, value := targetTree.Get([]byte("identity"))
	if !bytes.Equal(value, []byte("state")) {
		t.Fatalf("unexpected imported value: %x", value)
	}
}

func TestReadTreeFromRejectsOversizedChunk(t *testing.T) {
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "0",
		Mode:     0600,
		Size:     nodeState.MaxSnapshotChunkBytes + 1,
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}

	err := readTreeFrom(snapshotPrefixDB(), 1, &archive)
	if err == nil {
		t.Fatal("oversized snapshot chunk was accepted")
	}
}

func TestReadTreeFromRejectsNonRegularEntry(t *testing.T) {
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	if err := tw.WriteHeader(&tar.Header{
		Name:     "link",
		Linkname: "0",
		Mode:     0600,
		Typeflag: tar.TypeSymlink,
	}); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	err := readTreeFrom(snapshotPrefixDB(), 1, &archive)
	if err == nil {
		t.Fatal("non-regular snapshot entry was accepted")
	}
}
