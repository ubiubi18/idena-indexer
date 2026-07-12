# Idena Indexer Compatibility Fork

This fork indexes the existing Idena chain and does not define a new genesis,
network, gossip protocol, reward rule, or consensus change. It embeds the exact
`idena-go` and Wasm-binding revisions pinned by
`compatibility/stack-lock.json`.

The lock is a release candidate. Do not describe this branch as
legacy-compatible until its replay, P2P interoperability, Wasm differential,
independent rebuild, dependency, and secret-scan gates have all passed.

## Build

Requirements:

- Go `1.26.5`
- PostgreSQL compatible with the SQL under `resources/scripts`

```sh
go mod download
go test ./compatibility
go test ./... -run '^$'
go test ./config ./core/... ./indexer/...
scripts/build-reproducible.sh ./idena-indexer
python3 scripts/check_vulnerabilities.py
```

The PostgreSQL integration suite runs in CI against a digest-pinned database
image. Locally, with a disposable PostgreSQL instance accepting the example
loopback connection, run `scripts/test-postgres-integration.sh`. The script
serializes the two packages because they deliberately reuse and recreate the
same disposable schema.

The build script disables host VCS metadata and path-dependent build data. CI
builds twice and requires byte-identical outputs; publish a checksum only from
the exact locked commit and toolchain.

The vulnerability gate rejects every new reachable or module-only finding. An
upstream Kademlia DHT issue and one non-reachable `x/crypto` report, both
without fixed releases, are recorded as candidate-only in
`security/govuln-allowlist.json`; they still block changing the stack lock from
`candidate` to `released`. See `SECURITY.md` for the operational mitigations.

`conf/config.json` is a loopback-only development example. Put credentials and
machine-specific paths in untracked `conf/config.local.json`; never commit API
keys, node keys, identity data, database dumps, snapshots, or logs.

## Compatibility acceptance

Run the indexer against a dedicated copy of node and PostgreSQL state. Compare
every indexed block, epoch, reward category, reversal, identity root, and state
root against the unmodified 1.1.2 baseline over the same height range. Keep the
modern and legacy nodes on separate data directories and bind their RPC APIs to
loopback. A failed comparison blocks release and requires no chain migration or
fork activation.
