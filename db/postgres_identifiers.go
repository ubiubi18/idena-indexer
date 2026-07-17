package db

import (
	"regexp"
	"strings"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

var postgresIdentifierPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_$]*$`)

func quoteQualifiedIdentifier(value string) (string, error) {
	parts := strings.Split(value, ".")
	if len(parts) == 0 {
		return "", errors.New("identifier is empty")
	}
	quoted := make([]string, len(parts))
	for i, part := range parts {
		if !postgresIdentifierPattern.MatchString(part) {
			return "", errors.Errorf("invalid identifier segment %q", part)
		}
		quoted[i] = pq.QuoteIdentifier(strings.ToLower(part))
	}
	return strings.Join(quoted, "."), nil
}
