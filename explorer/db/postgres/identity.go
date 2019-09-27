package postgres

import (
	"database/sql"
	"github.com/idena-network/idena-go/common"
	"github.com/idena-network/idena-indexer/explorer/types"
	"math/big"
)

const (
	identityQuery                  = "identity.sql"
	identityAgeQuery               = "identityAge.sql"
	identityAnswerPointsQuery      = "identityAnswerPoints.sql"
	identityCurrentFlipsQuery      = "identityCurrentFlips.sql"
	identityEpochsCountQuery       = "identityEpochsCount.sql"
	identityEpochsQuery            = "identityEpochs.sql"
	identityFlipStatesQuery        = "identityFlipStates.sql"
	identityFlipRightAnswersQuery  = "identityFlipRightAnswers.sql"
	identityInvitesCountQuery      = "identityInvitesCount.sql"
	identityInvitesQuery           = "identityInvites.sql"
	identityStatesCountQuery       = "identityStatesCount.sql"
	identityStatesQuery            = "identityStates.sql"
	identityTxsCountQuery          = "identityTxsCount.sql"
	identityTxsQuery               = "identityTxs.sql"
	identityRewardsCountQuery      = "identityRewardsCount.sql"
	identityRewardsQuery           = "identityRewards.sql"
	identityEpochRewardsCountQuery = "identityEpochRewardsCount.sql"
	identityEpochRewardsQuery      = "identityEpochRewards.sql"
)

func (a *postgresAccessor) Identity(address string) (types.Identity, error) {
	identity := types.Identity{}
	err := a.db.QueryRow(a.getQuery(identityQuery), address).Scan(&identity.State)
	if err == sql.ErrNoRows {
		err = NoDataFound
	}
	if err != nil {
		return types.Identity{}, err
	}
	identity.Address = address

	if identity.ShortAnswers, identity.TotalShortAnswers, identity.LongAnswers, err = a.identityAnswerPoints(address); err != nil {
		return types.Identity{}, err
	}

	return identity, nil
}

func (a *postgresAccessor) identityAnswerPoints(address string) (short, totalShort, long types.IdentityAnswersSummary, err error) {
	rows, err := a.db.Query(a.getQuery(identityAnswerPointsQuery), address)
	if err != nil {
		return
	}
	defer rows.Close()
	if !rows.Next() {
		return
	}
	err = rows.Scan(&short.Point, &short.FlipsCount, &totalShort.Point, &totalShort.FlipsCount, &long.Point, &long.FlipsCount)
	return
}

func (a *postgresAccessor) IdentityAge(address string) (uint64, error) {
	var res uint64
	err := a.db.QueryRow(a.getQuery(identityAgeQuery), address).Scan(&res)
	if err == sql.ErrNoRows {
		err = NoDataFound
	}
	if err != nil {
		return 0, err
	}
	return res, nil
}

func (a *postgresAccessor) IdentityCurrentFlipCids(address string) ([]string, error) {
	rows, err := a.db.Query(a.getQuery(identityCurrentFlipsQuery), address)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []string
	for rows.Next() {
		var item string
		err = rows.Scan(&item)
		if err != nil {
			return nil, err
		}
		res = append(res, item)
	}
	return res, nil
}

func (a *postgresAccessor) IdentityEpochsCount(address string) (uint64, error) {
	return a.count(identityEpochsCountQuery, address)
}

func (a *postgresAccessor) IdentityEpochs(address string, startIndex uint64, count uint64) ([]types.EpochIdentitySummary, error) {
	rows, err := a.db.Query(a.getQuery(identityEpochsQuery), address, startIndex, count)
	if err != nil {
		return nil, err
	}
	return a.readEpochIdentitySummaries(rows)
}

func (a *postgresAccessor) IdentityFlipStates(address string) ([]types.StrValueCount, error) {
	rows, err := a.db.Query(a.getQuery(identityFlipStatesQuery), address)
	if err != nil {
		return nil, err
	}
	return a.readStrValueCounts(rows)
}

func (a *postgresAccessor) IdentityFlipQualifiedAnswers(address string) ([]types.StrValueCount, error) {
	rows, err := a.db.Query(a.getQuery(identityFlipRightAnswersQuery), address)
	if err != nil {
		return nil, err
	}
	return a.readStrValueCounts(rows)
}

func (a *postgresAccessor) IdentityInvitesCount(address string) (uint64, error) {
	return a.count(identityInvitesCountQuery, address)
}

func (a *postgresAccessor) IdentityInvites(address string, startIndex uint64, count uint64) ([]types.Invite, error) {
	rows, err := a.db.Query(a.getQuery(identityInvitesQuery), address, startIndex, count)
	if err != nil {
		return nil, err
	}
	return a.readInvites(rows)
}

func (a *postgresAccessor) IdentityStatesCount(address string) (uint64, error) {
	return a.count(identityStatesCountQuery, address)
}

func (a *postgresAccessor) IdentityStates(address string, startIndex uint64, count uint64) ([]types.IdentityState, error) {
	rows, err := a.db.Query(a.getQuery(identityStatesQuery), address, startIndex, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []types.IdentityState
	for rows.Next() {
		item := types.IdentityState{}
		var timestamp int64
		err = rows.Scan(&item.State, &item.Epoch, &item.BlockHeight, &item.BlockHash, &item.TxHash, &timestamp)
		if err != nil {
			return nil, err
		}
		item.Timestamp = common.TimestampToTime(big.NewInt(timestamp))
		res = append(res, item)
	}
	return res, nil
}

func (a *postgresAccessor) IdentityTxsCount(address string) (uint64, error) {
	return a.count(identityTxsCountQuery, address)
}

func (a *postgresAccessor) IdentityTxs(address string, startIndex uint64, count uint64) ([]types.TransactionSummary, error) {
	rows, err := a.db.Query(a.getQuery(identityTxsQuery), address, startIndex, count)
	if err != nil {
		return nil, err
	}
	return a.readTxs(rows)
}

func (a *postgresAccessor) IdentityRewardsCount(address string) (uint64, error) {
	return a.count(identityRewardsCountQuery, address)
}

func (a *postgresAccessor) IdentityRewards(address string, startIndex uint64, count uint64) ([]types.Reward, error) {
	return a.rewards(identityRewardsQuery, address, startIndex, count)
}

func (a *postgresAccessor) IdentityEpochRewardsCount(address string) (uint64, error) {
	return a.count(identityEpochRewardsCountQuery, address)
}

func (a *postgresAccessor) IdentityEpochRewards(address string, startIndex uint64, count uint64) ([]types.Rewards, error) {
	rows, err := a.db.Query(a.getQuery(identityEpochRewardsQuery), address, startIndex, count)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var res []types.Rewards
	var item *types.Rewards
	for rows.Next() {
		reward := types.Reward{}
		var epoch uint64
		if err := rows.Scan(&epoch, &reward.Balance, &reward.Stake, &reward.Type); err != nil {
			return nil, err
		}
		if item == nil || item.Epoch != epoch {
			if item != nil {
				res = append(res, *item)
			}
			item = &types.Rewards{
				Epoch: epoch,
			}
		}
		item.Rewards = append(item.Rewards, reward)
	}
	if item != nil {
		res = append(res, *item)
	}
	return res, nil
}
