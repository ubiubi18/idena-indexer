package compatibility

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	expectedReleaseID    = "idena-mainnet-legacy-compat-2026.07.12-rc3"
	expectedLegacyCommit = "938be81dbdeff85f888f4337060a8ebabb12e5b5"
	expectedNodeCommit   = "aafb254786ac3c82308550a7a82642019f077d6b"
	expectedNodeVersion  = "v0.17.2-0.20260712191802-4947ddfd4139"
	expectedBinding      = "v0.0.0-20260710141316-67ba065fdb02"
)

type stackLock struct {
	Schema         int    `json:"schema"`
	ReleaseID      string `json:"releaseId"`
	Status         string `json:"status"`
	LegacyBaseline struct {
		Commit      string `json:"commit"`
		NodeVersion string `json:"nodeVersion"`
	} `json:"legacyBaseline"`
	ChainInvariants struct {
		MainnetNetworkID                int    `json:"mainnetNetworkId"`
		GossipProtocol                  string `json:"gossipProtocol"`
		IntermediateGenesisHeaderSHA256 string `json:"intermediateGenesisHeaderSha256"`
		StateSnapshotSHA256             string `json:"stateSnapshotSha256"`
		IdentitySnapshotSHA256          string `json:"identitySnapshotSha256"`
		ConsensusChangesAllowed         bool   `json:"consensusChangesAllowed"`
	} `json:"chainInvariants"`
	Components []struct {
		Name       string `json:"name"`
		Repository string `json:"repository"`
		Commit     string `json:"commit"`
	} `json:"components"`
	ConsumerPins  map[string]map[string]string `json:"consumerPins"`
	RequiredGates []string                     `json:"requiredGates"`
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs("..")
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func loadLock(t *testing.T) stackLock {
	t.Helper()
	path := filepath.Join(repositoryRoot(t), "compatibility", "stack-lock.json")
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 || info.Size() > 1024*1024 {
		t.Fatal("stack lock must be a small regular file")
	}
	payload, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lock stackLock
	if err := json.Unmarshal(payload, &lock); err != nil {
		t.Fatal(err)
	}
	return lock
}

func TestLegacyCompatibilityLock(t *testing.T) {
	lock := loadLock(t)
	if lock.Schema != 1 || lock.ReleaseID != expectedReleaseID || lock.Status != "candidate" {
		t.Fatal("unexpected compatibility release candidate")
	}
	if lock.LegacyBaseline.NodeVersion != "1.1.2" || lock.LegacyBaseline.Commit != expectedLegacyCommit {
		t.Fatal("legacy baseline changed")
	}
	invariants := lock.ChainInvariants
	if invariants.MainnetNetworkID != 1 || invariants.GossipProtocol != "/idena/gossip/1.1.0" || invariants.ConsensusChangesAllowed {
		t.Fatal("chain or consensus invariants changed")
	}
	expectedDigests := map[string]string{
		"genesis":  "27e696414b955714ba7ed4defe063794c8dcadef28a7e61dd9249b8623571b3c",
		"state":    "7cf6f8c334d76a3617cbd5ac3aa5a104a8d337cb6ceb8d6906c62bf7fab8d131",
		"identity": "f136ec8939e3f78587a38de517128c7071501e283bac7d12c24ce4be830ff8aa",
	}
	actualDigests := map[string]string{
		"genesis":  invariants.IntermediateGenesisHeaderSHA256,
		"state":    invariants.StateSnapshotSHA256,
		"identity": invariants.IdentitySnapshotSHA256,
	}
	for name, expected := range expectedDigests {
		if actualDigests[name] != expected {
			t.Fatalf("%s resource digest changed", name)
		}
	}

	var nodeCount int
	for _, component := range lock.Components {
		if component.Name == "idena-go" {
			nodeCount++
			if component.Repository != "https://github.com/ubiubi18/idena-go.git" || component.Commit != expectedNodeCommit {
				t.Fatal("idena-go component changed")
			}
		}
	}
	if nodeCount != 1 || lock.ConsumerPins["idena-indexer"]["idena-go"] != expectedNodeCommit {
		t.Fatal("indexer node pin changed")
	}

	gates := make(map[string]bool)
	for _, gate := range lock.RequiredGates {
		gates[gate] = true
	}
	for _, gate := range []string{"legacy-state-replay-differential", "legacy-modern-p2p-interoperability", "secret-scan"} {
		if !gates[gate] {
			t.Fatalf("required gate missing: %s", gate)
		}
	}
}

func TestGoModPinsReviewedForks(t *testing.T) {
	payload, err := os.ReadFile(filepath.Join(repositoryRoot(t), "go.mod"))
	if err != nil {
		t.Fatal(err)
	}
	mod := string(payload)
	required := []string{
		"replace github.com/idena-network/idena-go => github.com/ubiubi18/idena-go " + expectedNodeVersion,
		"replace github.com/idena-network/idena-wasm-binding => github.com/ubiubi18/idena-wasm-binding " + expectedBinding,
		"replace github.com/cosmos/iavl => github.com/idena-network/iavl v0.12.3-0.20211223100228-a33b117aa31e",
	}
	for _, line := range required {
		if strings.Count(mod, line) != 1 {
			t.Fatalf("expected exactly one module pin: %s", line)
		}
	}
	for _, obsolete := range []string{"github.com/lucas-clemente/quic-go =>", "github.com/bits-and-blooms/bitset =>", "go.opentelemetry.io/otel/exporters/otlp =>"} {
		if strings.Contains(mod, obsolete) {
			t.Fatalf("obsolete compatibility replacement remains: %s", obsolete)
		}
	}
}
