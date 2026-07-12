from __future__ import annotations

import unittest

import check_vulnerabilities as policy


def stream_for(vuln_id: str = "GO-2024-3218", version: str = "v0.41.0") -> str:
    return """
{"config":{"scanner_version":"v1.6.0"}}
{"finding":{"osv":"%s","trace":[
  {"module":"github.com/libp2p/go-libp2p-kad-dht","version":"%s","package":"example","function":"Provide"},
  {"module":"github.com/idena-network/idena-indexer","package":"example","function":"Run"}
]}}
""" % (vuln_id, version)


class VulnerabilityPolicyTest(unittest.TestCase):
    def test_accepts_exact_candidate_exception(self) -> None:
        events = policy.decode_json_stream(stream_for())
        allowed = {("GO-2024-3218", "github.com/libp2p/go-libp2p-kad-dht", "v0.41.0", True)}
        policy.verify_policy(events, allowed)

    def test_rejects_new_reachable_vulnerability(self) -> None:
        events = policy.decode_json_stream(stream_for("GO-2099-0001"))
        allowed = {("GO-2024-3218", "github.com/libp2p/go-libp2p-kad-dht", "v0.41.0", True)}
        with self.assertRaisesRegex(policy.PolicyError, "new vulnerability"):
            policy.verify_policy(events, allowed)

    def test_rejects_stale_exception(self) -> None:
        events = policy.decode_json_stream('{"config":{"scanner_version":"v1.6.0"}}')
        allowed = {("GO-2024-3218", "github.com/libp2p/go-libp2p-kad-dht", "v0.41.0", True)}
        with self.assertRaisesRegex(policy.PolicyError, "stale"):
            policy.verify_policy(events, allowed)

    def test_module_only_findings_require_an_exact_exception(self) -> None:
        events = policy.decode_json_stream(
            '{"config":{"scanner_version":"v1.6.0"}}\n'
            '{"finding":{"osv":"GO-2099-0002","trace":['
            '{"module":"example.invalid/module","version":"v1.0.0"}]}}'
        )
        allowed = {("GO-2099-0002", "example.invalid/module", "v1.0.0", False)}
        policy.verify_policy(events, allowed)


if __name__ == "__main__":
    unittest.main()
