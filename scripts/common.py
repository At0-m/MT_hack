"""Docker/HTTP helpers. Only the Python standard library is required."""
from __future__ import annotations
import hashlib
import http.cookiejar
import json
import os
import platform
import re
import subprocess
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPTS = {"smoke", "forecast-day", "forecast-week", "forecast-month", "scenario", "mixed-workload", "bootstrap", "route-geometry", "export", "summary", "login"}

def environment() -> dict[str, str]:
    values: dict[str, str] = {}
    path = ROOT / ".env"
    if path.exists():
        for line in path.read_text().splitlines():
            line = line.strip()
            if line and not line.startswith("#") and "=" in line:
                key, value = line.split("=", 1)
                values[key.strip()] = value.strip().strip("\"'")
    values.update(os.environ)
    return values

def execute(args: list[str], *, env: dict[str, str] | None = None, check: bool = True, timeout: float = 60) -> subprocess.CompletedProcess:
    p = subprocess.run(args, cwd=ROOT, env=env or environment(), text=True, capture_output=True, timeout=timeout)
    if check and p.returncode:
        raise RuntimeError(f"Command failed ({p.returncode}): {' '.join(args[:4])}\n{redact(p.stderr, env or environment())[-4000:]}")
    return p

def redact(text: str, env: dict[str, str]) -> str:
    for key, value in env.items():
        if any(word in key for word in ("PASSWORD", "TOKEN", "SECRET")) and len(value) >= 8:
            text = text.replace(value, "[REDACTED]")
    return text

class Compose:
    def __init__(self, env: dict[str, str] | None = None):
        self.env = env or environment()
        files = self.env.get("COMPOSE_FILE", "compose.yaml").split(os.pathsep)
        self.prefix = ["docker", "compose", "--project-directory", str(ROOT)]
        for path in files:
            self.prefix += ["-f", path]
    def run(self, *args: str, timeout: float = 60, check: bool = True) -> subprocess.CompletedProcess:
        return execute(self.prefix + list(args), env=self.env, timeout=timeout, check=check)
    def ids(self, service: str | None = None) -> list[str]:
        args = ["ps", "-q"] + ([service] if service else [])
        return self.run(*args).stdout.split()

def inspect_container(cid: str, env: dict[str, str]) -> dict:
    raw = json.loads(execute(["docker", "inspect", cid], env=env).stdout)[0]
    host, config = raw["HostConfig"], raw["Config"]
    cores = host.get("NanoCpus", 0) / 1e9
    if not cores and host.get("CpuQuota", 0) > 0 and host.get("CpuPeriod", 0) > 0:
        cores = host["CpuQuota"] / host["CpuPeriod"]
    state = raw["State"]
    return {"id": raw["Id"], "name": raw["Name"].lstrip("/"), "service": config.get("Labels", {}).get("com.docker.compose.service", "load-generator"),
            "image": config["Image"], "image_id": raw["Image"], "cpu_limit_cores": cores, "memory_limit_bytes": host["Memory"],
            "memory_swap_limit_bytes": host.get("MemorySwap"), "cpuset": host.get("CpusetCpus", ""),
            "read_only": host.get("ReadonlyRootfs"), "user": config.get("User"), "restart_count": raw.get("RestartCount"),
            "state": {k: state.get(k) for k in ("Status", "OOMKilled", "ExitCode", "StartedAt", "FinishedAt")},
            "networks": {k: v.get("IPAddress", "") for k, v in raw["NetworkSettings"]["Networks"].items()},
            "settings": {item.split("=", 1)[0]: item.split("=", 1)[1] for item in config.get("Env", []) if item.split("=", 1)[0] in
                         {"FORECAST_CACHE_BYTES", "DB_MAX_CONNECTIONS", "ENABLE_PPROF", "GOMAXPROCS", "GOMEMLIMIT"}}}

def duration_seconds(text: str) -> float:
    m = re.fullmatch(r"(\d+)(ms|s|m|h)", text)
    if not m:
        raise ValueError("Duration must look like 30s, 5m or 1h")
    return int(m[1]) * {"ms": .001, "s": 1, "m": 60, "h": 3600}[m[2]]

def parse_bytes(value: str) -> float:
    m = re.fullmatch(r"\s*([\d.]+)\s*([kKMGT]?i?B)\s*", value)
    if not m:
        raise ValueError(f"Cannot parse Docker memory value: {value}")
    units = {"B": 1, "kB": 1000, "KB": 1000, "MB": 1000**2, "GB": 1000**3, "TB": 1000**4,
             "KiB": 1024, "MiB": 1024**2, "GiB": 1024**3, "TiB": 1024**4}
    return float(m[1]) * units[m[2]]

def source_digest() -> str:
    digest = hashlib.sha256()
    skip = {".git", "__pycache__", "artifacts", "testdata", "tmp", "results", "data"}
    for p in sorted(ROOT.rglob("*")):
        rel = p.relative_to(ROOT)
        if not p.is_file() or any(part in skip for part in rel.parts) or p.name.startswith(".env"):
            continue
        digest.update(str(rel).encode()); digest.update(b"\0"); digest.update(p.read_bytes())
    return digest.hexdigest()

def host_metadata(env: dict[str, str]) -> dict:
    cpu_model = "unknown"
    path = Path("/proc/cpuinfo")
    if path.exists():
        for line in path.read_text().splitlines():
            if line.startswith("model name"):
                cpu_model = line.split(":", 1)[1].strip(); break
    commit = execute(["git", "rev-parse", "HEAD"], env=env, check=False).stdout.strip() or "not-a-git-checkout"
    return {"utc": time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime()), "commit": commit,
            "dirty": bool(execute(["git", "status", "--porcelain"], env=env, check=False).stdout.strip()),
            "source_sha256": source_digest(), "platform": platform.platform(), "cpu_model": cpu_model,
            "host_logical_cpus": os.cpu_count(), "docker": execute(["docker", "version", "--format", "{{json .Server}}"], env=env).stdout.strip(),
            "compose": execute(["docker", "compose", "version", "--short"], env=env).stdout.strip()}

class API:
    def __init__(self, base: str):
        self.base = base.rstrip("/")
        self.jar = http.cookiejar.CookieJar()
        self.client = urllib.request.build_opener(urllib.request.HTTPCookieProcessor(self.jar))
    def request(self, path: str, body: dict | None = None, timeout: float = 15) -> tuple[int, object, float]:
        raw = None if body is None else json.dumps(body).encode()
        req = urllib.request.Request(self.base + path, raw, {"Content-Type": "application/json"})
        start = time.perf_counter()
        try:
            with self.client.open(req, timeout=timeout) as res:
                data, status, content = res.read(), res.status, res.headers.get("Content-Type", "")
        except urllib.error.HTTPError as error:
            data, status, content = error.read(), error.code, error.headers.get("Content-Type", "")
        elapsed = (time.perf_counter() - start) * 1000
        return status, json.loads(data) if "json" in content and data else data.decode(errors="replace"), elapsed
    def login(self, env: dict[str, str]) -> None:
        status, _, _ = self.request("/api/v1/auth/login", {"username": env.get("API_USER", "devops"), "password": env["API_PASSWORD"]})
        if status != 200: raise RuntimeError(f"Login failed: HTTP {status}")
    def logout(self) -> None:
        self.request("/api/v1/auth/logout", {})

def wait_ready(env: dict[str, str], timeout: float = 180) -> None:
    api = API(f"http://127.0.0.1:{env.get('HTTP_PORT', '8080')}")
    deadline, last = time.monotonic() + timeout, "no response"
    while time.monotonic() < deadline:
        try:
            status, _, _ = api.request("/health/ready", timeout=3)
            if status == 200: return
            last = str(status)
        except (OSError, urllib.error.URLError) as e:
            last = type(e).__name__
        time.sleep(1)
    raise RuntimeError(f"Readiness did not pass within {timeout}s ({last})")

def scrape(cid: str, env: dict[str, str]) -> str:
    return execute(["docker", "exec", cid, "/app/healthcheck", "--url", "http://127.0.0.1:9090/metrics", "--print", "--timeout", "5s"], env=env, timeout=10).stdout
