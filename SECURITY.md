# Security Policy

Do not report private keys, API keys, identity data, database dumps, or live
node addresses in a public issue. Use GitHub's private vulnerability reporting
for security-sensitive reports.

## Candidate dependency exception

The compatibility candidate inherits `GO-2024-3218` from
`github.com/libp2p/go-libp2p-kad-dht` `v0.41.0` through the locked Kubo and
`idena-go` runtime. The Go vulnerability database currently names no fixed
version. The issue can let hostile DHT participants reduce content
availability; removing DHT behavior here would change Idena interoperability.

Until upstream supplies a compatible fix:

- keep the indexer and RPC API off the public Internet;
- use multiple independently operated bootnodes and monitor peer diversity;
- alert on stalled heights, low peer counts, and failed content retrieval;
- retain a known-good legacy node and database rollback path;
- do not promote `compatibility/stack-lock.json` from `candidate` to
  `released` solely because CI is green.

`scripts/check_vulnerabilities.py` permits only this exact module/version/ID
tuple for candidate CI and fails when the exception disappears or any new
reachable vulnerability appears. It also tracks the currently non-reachable
`GO-2026-5932` finding in `golang.org/x/crypto` `v0.54.0`, for which the
database lists no fixed version. The previous archive reader was removed
entirely after `govulncheck` identified RAR, XZ, and path-traversal findings.
