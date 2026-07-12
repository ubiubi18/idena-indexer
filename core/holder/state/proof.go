package state

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/cosmos/iavl"
	"github.com/golang/protobuf/proto"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/common/hexutil"
	"github.com/idena-network/idena-go/common/math"
	"github.com/idena-network/idena-go/core/state"
	models "github.com/idena-network/idena-go/protobuf"
	"github.com/idena-network/idena-indexer/log"
	"github.com/pkg/errors"
	db "github.com/tendermint/tm-db"
)

type Holder interface {
	IdentityWithProof(epoch uint64, address common.Address) (*hexutil.Bytes, error)
}

func NewHolder(treeSnapshotDir string, logger log.Logger) Holder {
	return &holderImpl{
		treeSnapshotDir: treeSnapshotDir,
		statesByVersion: make(map[uint64]*state.StateDB),
		logger:          logger,
	}
}

type holderImpl struct {
	statesByVersion map[uint64]*state.StateDB
	lock            sync.RWMutex
	treeSnapshotDir string
	logger          log.Logger
}

func (h *holderImpl) IdentityWithProof(epoch uint64, address common.Address) (*hexutil.Bytes, error) {
	state, err := h.getState(epoch)
	if err != nil {
		return nil, err
	}
	valueWithProof, err := state.GetIdentityWithProof(address)
	if err != nil {
		return nil, err
	}
	if len(valueWithProof) == 0 {
		return nil, nil
	}
	res := hexutil.Bytes(valueWithProof)
	return &res, nil
}

func (h *holderImpl) getState(epoch uint64) (*state.StateDB, error) {
	h.lock.RLock()
	st, ok := h.statesByVersion[epoch]
	h.lock.RUnlock()
	if ok {
		return st, nil
	}

	h.lock.Lock()
	defer h.lock.Unlock()

	st, ok = h.statesByVersion[epoch]
	if ok {
		return st, nil
	}
	h.logger.Info(fmt.Sprintf("Start loading state for epoch %v", epoch))
	file, err := os.Open(filepath.Join(h.treeSnapshotDir, fmt.Sprintf("%v.tar", epoch)))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	mdb := db.NewMemDB()
	st, err = state.NewLazy(mdb)
	if err != nil {
		return nil, err
	}
	const height uint64 = math.MaxInt64
	pdb := db.NewPrefixDB(mdb, state.StateDbKeys.BuildDbPrefix(height))
	if err := readTreeFrom(pdb, height, file); err != nil {
		return nil, err
	}
	st.CommitSnapshot(height, nil)
	h.statesByVersion[epoch] = st
	h.logger.Info(fmt.Sprintf("State for epoch %v loaded", epoch))
	return st, nil
}

func readTreeFrom(pdb *db.PrefixDB, height uint64, from io.Reader) error {
	tr := tar.NewReader(from)

	tree := state.NewMutableTree(pdb)
	importer, err := tree.Importer(int64(height))
	if err != nil {
		return err
	}
	defer importer.Close()

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			common.ClearDb(pdb)
			return err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			common.ClearDb(pdb)
			return errors.Errorf("snapshot contains unsupported tar entry type %d", header.Typeflag)
		}
		if header.Size < 0 || header.Size > state.MaxSnapshotChunkBytes {
			common.ClearDb(pdb)
			return errors.Errorf(
				"snapshot chunk size %d exceeds limit %d",
				header.Size,
				state.MaxSnapshotChunkBytes,
			)
		}
		if data, err := io.ReadAll(tr); err != nil {
			common.ClearDb(pdb)
			return err
		} else {
			sb := new(models.ProtoSnapshotNodes)
			if err := proto.Unmarshal(data, sb); err != nil {
				common.ClearDb(pdb)
				return err
			}
			for _, node := range sb.Nodes {

				exportNode := &iavl.ExportNode{
					Key:     node.Key,
					Value:   node.Value,
					Version: int64(node.Version),
					Height:  int8(node.Height),
				}

				if node.EmptyValue {
					exportNode.Value = make([]byte, 0)
				}

				importer.Add(exportNode)
			}
		}
	}
	if err := importer.Commit(); err != nil {
		common.ClearDb(pdb)
		return err
	}

	if _, err := tree.LoadVersion(int64(height)); err != nil {
		common.ClearDb(pdb)
		return err
	}
	if !tree.ValidateTree() {
		common.ClearDb(pdb)
		return errors.New("corrupted tree")
	}
	return nil
}
