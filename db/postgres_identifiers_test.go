package db

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestQuoteQualifiedIdentifier(t *testing.T) {
	quoted, err := quoteQualifiedIdentifier("Report.Refresh_Coins$Daily")

	require.NoError(t, err)
	require.Equal(t, `"report"."refresh_coins$daily"`, quoted)
}

func TestQuoteQualifiedIdentifierRejectsSQL(t *testing.T) {
	_, err := quoteQualifiedIdentifier("report.refresh_coins; DROP TABLE identities")

	require.Error(t, err)
}

func TestQuoteQualifiedIdentifierRejectsEmptySegment(t *testing.T) {
	_, err := quoteQualifiedIdentifier("report..refresh_coins")

	require.Error(t, err)
}
