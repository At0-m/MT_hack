#!/usr/bin/env python3
import json
import time
import urllib.error
import urllib.request

from common import API, ROOT, Compose, environment, inspect_container, wait_ready


def request(base, path, *, method="GET", body=None, token=None):
    raw = None if body is None else json.dumps(body).encode()
    headers = {"Content-Type": "application/json"}

    if token:
        headers["Cookie"] = f"tramflow_session={token}"

    req = urllib.request.Request(
        base + path,
        data=raw,
        headers=headers,
        method=method,
    )

    try:
        with urllib.request.urlopen(req, timeout=10) as res:
            return res.status, res.read().decode(errors="replace")
    except urllib.error.HTTPError as err:
        return err.code, err.read().decode(errors="replace")


def main():
    env = environment()
    dc = Compose(env)

    dc.run(
        "up",
        "-d",
        "--no-deps",
        "--scale",
        "api=2",
        "--wait",
        "--wait-timeout",
        "180",
        "api",
        timeout=240,
    )

    dc.run(
        "up",
        "-d",
        "--no-deps",
        "--force-recreate",
        "gateway",
        timeout=120,
    )

    wait_ready(env)

    ids = dc.ids("api")

    if len(ids) != 2:
        raise RuntimeError("Expected two API replicas")

    replicas = [inspect_container(cid, env) for cid in ids]
    a, b = replicas

    networks = set(a["networks"]) & set(b["networks"])

    if len(networks) != 1:
        raise RuntimeError("Expected one common network")

    network = next(iter(networks))

    base_a = "http://" + a["networks"][network] + ":8080"
    base_b = "http://" + b["networks"][network] + ":8080"

    # Login directly through replica A.
    api_a = API(base_a)
    api_a.login(env)

    token = next(
        (c.value for c in api_a.jar if c.name == "tramflow_session"),
        None,
    )

    if not token:
        raise RuntimeError(
            "Login succeeded but tramflow_session cookie was not returned"
        )

    # Session created on A must be visible on B.
    status_b, _ = request(
        base_b,
        "/api/v1/auth/session",
        token=token,
    )

    if status_b != 200:
        raise RuntimeError(
            f"Session created on replica A was not accepted "
            f"by replica B: HTTP {status_b}"
        )

    # Logout through replica B.
    logout_b, _ = request(
        base_b,
        "/api/v1/auth/logout",
        method="POST",
        body={},
        token=token,
    )

    if logout_b != 204:
        raise RuntimeError(
            f"Logout on replica B failed: HTTP {logout_b}"
        )

    # Revocation must immediately be visible on A.
    status_a_after, _ = request(
        base_a,
        "/api/v1/auth/session",
        token=token,
    )

    if status_a_after != 401:
        raise RuntimeError(
            f"Replica A still accepted a session revoked "
            f"on replica B: HTTP {status_a_after}"
        )

    checks = {
        "login_on_replica_a": True,
        "session_visible_on_replica_b": True,
        "logout_on_replica_b": True,
        "revocation_visible_on_replica_a": True,
    }

    out = (
        ROOT
        / "benchmarks"
        / "results"
        / time.strftime("scale-%Y%m%dT%H%M%SZ", time.gmtime())
    )

    out.mkdir(parents=True, exist_ok=False)

    (out / "cross-replica.json").write_text(
        json.dumps(
            {
                "checks": checks,
                "replicas": replicas,
            },
            indent=2,
        )
        + "\n"
    )

    print(json.dumps(checks, indent=2))
    print(out)
    print(
        "Two replicas remain running; "
        "restore with make up REPLICAS=1 when finished"
    )


if __name__ == "__main__":
    main()
