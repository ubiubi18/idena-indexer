package db

import (
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"github.com/idena-network/idena-go/common"
	data2 "github.com/idena-network/idena-indexer/data"
	"github.com/lib/pq"
	"github.com/pkg/errors"
)

func (a *postgresAccessor) GetDataList() ([]data2.Data, error) {
	dataTable, err := quoteQualifiedIdentifier(a.dataTable)
	if err != nil {
		return nil, errors.Wrap(err, "invalid data table")
	}
	dataStateTable, err := quoteQualifiedIdentifier(a.dataStateTable)
	if err != nil {
		return nil, errors.Wrap(err, "invalid data state table")
	}
	query := fmt.Sprintf("SELECT v.name, v.refresh_procedure, v.refresh_period, v.refresh_delay_minutes, vs.refresh_time, vs.refresh_epoch FROM %s v LEFT JOIN %s vs ON vs.name = v.name", dataTable, dataStateTable)
	rows, err := a.db.Query(query)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code.Name() == "undefined_table" {
			err = errors.Wrap(a.createDataTables(), "unable to create table")
			if err == nil {
				rows, err = a.db.Query(query)
			}
		}
		return nil, err
	}
	defer rows.Close()
	var result []data2.Data
	for rows.Next() {
		var item data2.Data
		var delayMinutes, refreshTime, refreshEpoch sql.NullInt64
		var refreshProcedure, refreshPeriod sql.NullString
		err = rows.Scan(
			&item.Name,
			&refreshProcedure,
			&refreshPeriod,
			&delayMinutes,
			&refreshTime,
			&refreshEpoch,
		)
		if err != nil {
			return nil, err
		}
		if refreshProcedure.Valid {
			item.RefreshProcedure = &refreshProcedure.String
		}
		if refreshPeriod.Valid {
			item.RefreshPeriod = &refreshPeriod.String
		}
		if delayMinutes.Valid {
			v := time.Duration(delayMinutes.Int64) * time.Minute
			item.RefreshDelay = &v
		}
		if refreshTime.Valid {
			v := timestampToTimeUTC(refreshTime.Int64)
			item.RefreshTime = &v
		}
		if refreshEpoch.Valid {
			v := uint16(refreshEpoch.Int64)
			item.RefreshEpoch = &v
		}
		result = append(result, item)
	}
	return result, nil
}

func (a *postgresAccessor) createDataTables() error {
	dataTable, err := quoteQualifiedIdentifier(a.dataTable)
	if err != nil {
		return errors.Wrap(err, "invalid data table")
	}
	dataStateTable, err := quoteQualifiedIdentifier(a.dataStateTable)
	if err != nil {
		return errors.Wrap(err, "invalid data state table")
	}
	_, err = a.db.Exec(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (name character varying(30) NOT NULL, refresh_procedure character varying(30), refresh_period character varying(1), refresh_delay_minutes smallint, endpoint_method character varying(30), \"limit\" smallint); CREATE TABLE IF NOT EXISTS %s (name character varying(100) NOT NULL, refresh_time bigint, refresh_epoch smallint, last_refresh_time bigint, CONSTRAINT data_state_pkey PRIMARY KEY (name));",
		dataTable,
		dataStateTable,
	))
	return err
}

func (a *postgresAccessor) UpdateRefreshTime(dataName string, refreshTime time.Time) error {
	dataStateTable, err := quoteQualifiedIdentifier(a.dataStateTable)
	if err != nil {
		return errors.Wrap(err, "invalid data state table")
	}
	_, err = a.db.Exec(fmt.Sprintf("INSERT INTO %s VALUES ($2, $1, null) ON CONFLICT (name) DO UPDATE SET refresh_time = $1", dataStateTable),
		refreshTime.Unix(),
		dataName,
	)
	return err
}

func (a *postgresAccessor) Refresh(dataName, refreshProcedure string, time time.Time, nextRefreshTime *time.Time, refreshEpoch *uint16) error {
	quotedRefreshProcedure, err := quoteQualifiedIdentifier(refreshProcedure)
	if err != nil {
		return errors.Wrap(err, "invalid refresh procedure")
	}
	dataStateTable, err := quoteQualifiedIdentifier(a.dataStateTable)
	if err != nil {
		return errors.Wrap(err, "invalid data state table")
	}
	var nextRefreshTimeUnix *int64
	if nextRefreshTime != nil {
		v := nextRefreshTime.Unix()
		nextRefreshTimeUnix = &v
	}
	tx, err := a.db.Begin()
	if err != nil {
		return getResultError(err)
	}
	defer tx.Rollback()
	_, err = tx.Exec(fmt.Sprintf("CALL %s()", quotedRefreshProcedure))
	if err != nil {
		return getResultError(err)
	}
	_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s VALUES ($3, $1, $2, $4) ON CONFLICT (name) DO UPDATE SET refresh_time = $1, refresh_epoch = $2, last_refresh_time = $4", dataStateTable),
		nextRefreshTimeUnix,
		refreshEpoch,
		dataName,
		time.Unix(),
	)
	if err != nil {
		return getResultError(err)
	}
	return tx.Commit()
}

func timestampToTimeUTC(timestamp int64) time.Time {
	return common.TimestampToTime(big.NewInt(timestamp)).UTC()
}
