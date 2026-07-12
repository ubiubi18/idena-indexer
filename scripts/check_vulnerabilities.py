#!/usr/bin/env python3
"""Reject new reachable Go vulnerabilities with an exact candidate allowlist."""

from __future__ import annotations

import json
import os
import subprocess
import sys
from pathlib import Path
from typing import Any, Iterable


ROOT = Path(__file__).resolve().parents[1]
ALLOWLIST = ROOT / "security" / "govuln-allowlist.json"
SCANNER_VERSION = "v1.6.0"
MAX_POLICY_BYTES = 64 * 1024


class PolicyError(ValueError):
    pass


def require(condition: bool, message: str) -> None:
    if not condition:
        raise PolicyError(message)


def decode_json_stream(payload: str) -> list[dict[str, Any]]:
    decoder = json.JSONDecoder()
    events: list[dict[str, Any]] = []
    offset = 0
    while offset < len(payload):
        while offset < len(payload) and payload[offset].isspace():
            offset += 1
        if offset == len(payload):
            break
        event, offset = decoder.raw_decode(payload, offset)
        require(isinstance(event, dict), "govulncheck emitted a non-object event")
        events.append(event)
    return events


def vulnerability_findings(events: Iterable[dict[str, Any]]) -> set[tuple[str, str, str, bool]]:
    findings: dict[tuple[str, str, str], bool] = {}
    for event in events:
        finding = event.get("finding")
        if not isinstance(finding, dict):
            continue
        trace = finding.get("trace")
        if not isinstance(trace, list):
            continue
        reachable = any(isinstance(frame, dict) and frame.get("function") for frame in trace)
        source = next(
            (
                frame
                for frame in trace
                if isinstance(frame, dict)
                and isinstance(frame.get("module"), str)
                and isinstance(frame.get("version"), str)
                and frame["module"] != "github.com/idena-network/idena-indexer"
            ),
            None,
        )
        require(source is not None, "vulnerability finding lacks a dependency source")
        vuln_id = finding.get("osv")
        require(isinstance(vuln_id, str), "vulnerability finding lacks an OSV ID")
        key = (vuln_id, source["module"], source["version"])
        findings[key] = findings.get(key, False) or reachable
    return {(*key, reachable) for key, reachable in findings.items()}


def load_allowlist(path: Path = ALLOWLIST) -> set[tuple[str, str, str, bool]]:
    info = path.lstat()
    require(path.is_file() and not path.is_symlink(), "allowlist must be a regular file")
    require(info.st_size <= MAX_POLICY_BYTES, "allowlist is unexpectedly large")
    payload = json.loads(path.read_text(encoding="utf-8"))
    require(isinstance(payload, dict) and payload.get("schema") == 1, "unsupported allowlist schema")
    exceptions = payload.get("exceptions")
    require(isinstance(exceptions, list), "allowlist exceptions must be a list")
    allowed: set[tuple[str, str, str, bool]] = set()
    for exception in exceptions:
        require(isinstance(exception, dict), "invalid allowlist exception")
        require(exception.get("candidateOnly") is True, "exception must be candidate-only")
        require(exception.get("fixedVersion") is None, "fixed vulnerabilities cannot be allowlisted")
        identity = (exception.get("id"), exception.get("module"), exception.get("version"))
        require(all(isinstance(value, str) and value for value in identity), "invalid exception identity")
        require(isinstance(exception.get("reachable"), bool), "exception reachability must be explicit")
        key = (*identity, exception["reachable"])
        require(key not in allowed, "duplicate vulnerability exception")
        allowed.add(key)
    return allowed


def verify_policy(events: list[dict[str, Any]], allowed: set[tuple[str, str, str, bool]]) -> None:
    configs = [event["config"] for event in events if isinstance(event.get("config"), dict)]
    require(len(configs) == 1, "govulncheck configuration event is missing or duplicated")
    require(configs[0].get("scanner_version") == SCANNER_VERSION, "unexpected govulncheck version")
    actual = vulnerability_findings(events)
    unexpected = actual - allowed
    stale = allowed - actual
    if unexpected:
        rendered = ", ".join(
            f"{vuln} ({module}@{version}, reachable={str(reachable).lower()})"
            for vuln, module, version, reachable in sorted(unexpected)
        )
        raise PolicyError(f"new vulnerability findings: {rendered}")
    if stale:
        rendered = ", ".join(
            f"{vuln} ({module}@{version}, reachable={str(reachable).lower()})"
            for vuln, module, version, reachable in sorted(stale)
        )
        raise PolicyError(f"stale vulnerability exceptions: {rendered}")


def run_scanner() -> list[dict[str, Any]]:
    env = os.environ.copy()
    env["GOTOOLCHAIN"] = "go1.26.5"
    command = [
        "go",
        "run",
        f"golang.org/x/vuln/cmd/govulncheck@{SCANNER_VERSION}",
        "-json",
        "./...",
    ]
    result = subprocess.run(
        command,
        cwd=ROOT,
        env=env,
        text=True,
        capture_output=True,
        check=False,
        timeout=900,
    )
    require(result.returncode == 0, "govulncheck execution failed")
    return decode_json_stream(result.stdout)


def main() -> int:
    try:
        allowed = load_allowlist()
        verify_policy(run_scanner(), allowed)
    except (OSError, json.JSONDecodeError, subprocess.TimeoutExpired, PolicyError) as exc:
        print(f"Vulnerability policy failed: {exc}", file=sys.stderr)
        return 1
    print(f"Vulnerability policy passed with {len(allowed)} candidate-only exception(s).")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
