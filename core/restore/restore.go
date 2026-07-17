package restore

import (
	"github.com/idena-network/idena-go/blockchain"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-go/core/appstate"
	"github.com/idena-network/idena-go/core/state"
	"github.com/idena-network/idena-indexer/core/conversion"
	"github.com/idena-network/idena-indexer/db"
	"github.com/idena-network/idena-indexer/log"
	"github.com/pkg/errors"
	"time"
)

type Restorer struct {
	db       db.Accessor
	appState *appstate.AppState
	chain    *blockchain.Blockchain
}

func NewRestorer(db db.Accessor, appState *appstate.AppState, chain *blockchain.Blockchain) *Restorer {
	return &Restorer{
		db:       db,
		appState: appState,
		chain:    chain,
	}
}

func (r *Restorer) Restore() {
	for {
		if err := r.tryToRestore(); err != nil {
			log.Error(err.Error())
			time.Sleep(time.Second * 5)
			continue
		}
		return
	}
}

func (r *Restorer) tryToRestore() error {
	data, err := r.collectData()
	if err != nil {
		return errors.Wrapf(err, "unable to collect data to restore")
	}
	err = r.db.SaveRestoredData(data)
	return errors.Wrapf(err, "unable to save restored data")
}

func (r *Restorer) collectData() (*db.RestoredData, error) {
	indexedHeight, err := r.db.GetLastHeight()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get last indexed block height")
	}
	stateHeight, err := r.restoreHeight()
	if err != nil {
		return nil, err
	}
	res := &db.RestoredData{BlockHeight: indexedHeight}
	if res.Balances, err = r.collectBalances(stateHeight); err != nil {
		return nil, err
	}
	if res.Birthdays, res.PoolSizes, res.Delegations, err = r.collectIdentityData(stateHeight); err != nil {
		return nil, err
	}
	return res, nil
}

func (r *Restorer) restoreHeight() (uint64, error) {
	head := r.chain.Head
	if head == nil {
		return 0, errors.New("blockchain header is nil")
	}
	if head.Height() == 0 {
		return 0, errors.New("blockchain header height is zero")
	}
	return head.Height() - 1, nil
}

func (r *Restorer) collectBalances(height uint64) ([]db.Balance, error) {
	appState, err := r.appState.Readonly(height)
	if err != nil {
		return nil, errors.Errorf("unable to get appState for height %d, err %v", height, err.Error())
	}
	var balances []db.Balance
	appState.State.IterateAccounts(func(key []byte, _ []byte) bool {
		if key == nil {
			return true
		}
		address := conversion.BytesToAddr(key)
		convertedAddress := conversion.ConvertAddress(address)
		balances = append(balances, db.Balance{
			Address: convertedAddress,
			Balance: blockchain.ConvertToFloat(appState.State.GetBalance(address)),
			Stake:   blockchain.ConvertToFloat(appState.State.GetStakeBalance(address)),
		})
		return false
	})
	return balances, nil
}

func (r *Restorer) collectIdentityData(height uint64) ([]db.Birthday, []*db.PoolSize, []*db.Delegation, error) {
	appState, err := r.appState.Readonly(height)
	if err != nil {
		return nil, nil, nil, errors.Errorf("unable to get appState for height %d, err %v", height, err.Error())
	}
	var birthdays []db.Birthday
	poolSizesByAddr := make(map[common.Address]uint64)
	poolDelegatorsByAddr := make(map[common.Address]uint64)
	var delegations []*db.Delegation
	appState.State.IterateOverIdentities(func(addr common.Address, identity state.Identity) {
		birthEpoch := identity.Birthday

		birthdays = append(birthdays, db.Birthday{
			Address:    conversion.ConvertAddress(addr),
			BirthEpoch: uint64(birthEpoch),
		})

		if identity.Delegatee() != nil {
			poolDelegatorsByAddr[*identity.Delegatee()]++
			if _, ok := poolSizesByAddr[*identity.Delegatee()]; !ok {
				poolSizesByAddr[*identity.Delegatee()] = uint64(appState.ValidatorsCache.PoolSize(*identity.Delegatee()))
			}
			delegation := &db.Delegation{
				Delegator: addr,
				Delegatee: *identity.Delegatee(),
			}
			delegations = append(delegations, delegation)
			if identity.State.NewbieOrBetter() || identity.State == state.Suspended || identity.State == state.Zombie {
				delegation.BirthEpoch = &birthEpoch
			}
		}
	})

	poolSizes := make([]*db.PoolSize, 0, len(poolDelegatorsByAddr))
	for addr, size := range poolDelegatorsByAddr {
		poolSizes = append(poolSizes, &db.PoolSize{
			Address:        addr,
			TotalDelegated: size,
			Size:           poolSizesByAddr[addr],
		})
	}

	return birthdays, poolSizes, delegations, nil
}
