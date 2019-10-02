package postgres

import (
	"github.com/idena-network/idena-indexer/explorer/types"
	"github.com/shopspring/decimal"
)

const (
	epochsCountQuery = "epochsCount.sql"
	epochsQuery      = "epochs.sql"
)

func (a *postgresAccessor) EpochsCount() (uint64, error) {
	return a.count(epochsCountQuery)
}

func (a *postgresAccessor) Epochs(startIndex uint64, count uint64) ([]types.EpochSummary, error) {
	rows, err := a.db.Query(a.getQuery(epochsQuery), startIndex, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var epochs []types.EpochSummary
	for rows.Next() {
		epoch := types.EpochSummary{
			Coins: types.AllCoins{},
		}
		err = rows.Scan(&epoch.Epoch,
			&epoch.ValidatedCount,
			&epoch.BlockCount,
			&epoch.EmptyBlockCount,
			&epoch.TxCount,
			&epoch.InviteCount,
			&epoch.FlipCount,
			&epoch.Coins.Burnt,
			&epoch.Coins.Minted,
			&epoch.Coins.TotalBalance,
			&epoch.Coins.TotalStake)
		if err != nil {
			return nil, err
		}
		// todo tmp for backward compatibility
		epoch.Coins.Balance = types.Coins{
			Minted: epoch.Coins.Minted,
			Burnt:  epoch.Coins.Burnt,
			Total:  epoch.Coins.TotalBalance,
		}
		epoch.Coins.Stake = types.Coins{
			Minted: decimal.Zero,
			Burnt:  decimal.Zero,
			Total:  epoch.Coins.TotalStake,
		}
		epochs = append(epochs, epoch)
	}
	return epochs, nil
}
