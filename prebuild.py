#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import platform
import subprocess
import sys
import textwrap
import urllib.request
from pathlib import Path


def run(cmd: list[str], *, cwd: Path | None = None) -> None:
    subprocess.run(cmd, cwd=cwd, check=True)


def run_capture(cmd: list[str], *, cwd: Path | None = None) -> str:
    return subprocess.check_output(cmd, cwd=cwd, text=True).strip()


def fetch_latest_tag(upstream_repo: str) -> str:
    url = f"https://api.github.com/repos/{upstream_repo}/releases/latest"
    opener = urllib.request.build_opener(urllib.request.ProxyHandler())
    req = urllib.request.Request(url, headers={"User-Agent": "core-compiled-prebuild"})
    with opener.open(req, timeout=30) as resp:
        data = json.loads(resp.read().decode("utf-8"))
    tag = data.get("tag_name")
    if not tag:
        raise RuntimeError("failed to fetch latest upstream tag")
    return str(tag)


def normalize_proxy_url(value: str) -> str:
    if "://" in value:
        return value
    return f"http://{value}"


def apply_proxy_env_from_args(proxy: str | None) -> None:
    if not proxy:
        return
    value = normalize_proxy_url(proxy)
    os.environ["HTTP_PROXY"] = value
    os.environ["HTTPS_PROXY"] = value
    os.environ["http_proxy"] = value
    os.environ["https_proxy"] = value


def apply_proxy_env_from_windows_system() -> None:
    if platform.system().lower() != "windows":
        return
    if any(os.environ.get(k) for k in ("HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy")):
        return

    try:
        import winreg  # type: ignore
    except Exception:
        return

    try:
        with winreg.OpenKey(
            winreg.HKEY_CURRENT_USER,
            r"Software\Microsoft\Windows\CurrentVersion\Internet Settings",
        ) as key:
            enabled, _ = winreg.QueryValueEx(key, "ProxyEnable")
            server, _ = winreg.QueryValueEx(key, "ProxyServer")
    except OSError:
        return

    if int(enabled) != 1:
        return
    if not isinstance(server, str) or not server.strip():
        return

    server = server.strip()
    http_proxy: str | None = None
    https_proxy: str | None = None

    if ";" in server or "=" in server:
        for part in server.split(";"):
            part = part.strip()
            if not part or "=" not in part:
                continue
            scheme, addr = part.split("=", 1)
            scheme = scheme.strip().lower()
            addr = addr.strip()
            if not addr:
                continue
            if scheme == "http":
                http_proxy = normalize_proxy_url(addr)
            elif scheme == "https":
                https_proxy = normalize_proxy_url(addr)
    else:
        http_proxy = normalize_proxy_url(server)
        https_proxy = normalize_proxy_url(server)

    if http_proxy:
        os.environ["HTTP_PROXY"] = http_proxy
        os.environ["http_proxy"] = http_proxy
    if https_proxy:
        os.environ["HTTPS_PROXY"] = https_proxy
        os.environ["https_proxy"] = https_proxy


def ensure_mihomo_source(src_dir: Path, *, upstream_repo: str, tag: str) -> bool:
    repo_url = f"https://github.com/{upstream_repo}.git"
    if not src_dir.exists():
        run(["git", "clone", "--depth=1", "--branch", tag, repo_url, str(src_dir)])
        return True

    if not (src_dir / ".git").exists():
        raise RuntimeError(f"{src_dir} exists but is not a git repository")

    try:
        current_tag = run_capture(["git", "describe", "--tags", "--exact-match"], cwd=src_dir)
    except subprocess.CalledProcessError:
        current_tag = ""

    if current_tag == tag:
        return False

    run(["git", "fetch", "--depth=1", "origin", "tag", tag], cwd=src_dir)
    run(["git", "checkout", "--detach", "FETCH_HEAD"], cwd=src_dir)
    run(["git", "reset", "--hard"], cwd=src_dir)
    run(["git", "clean", "-xfd"], cwd=src_dir)
    return True


def main(argv: list[str] | None = None) -> int:
    argv = sys.argv[1:] if argv is None else argv
    repo_root = Path(__file__).resolve().parent
    src_dir = repo_root / "mihomo-source"

    parser = argparse.ArgumentParser(
        description="Prebuild step: download/update mihomo source into mihomo-source/.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent(
            """\
            Examples:
              python prebuild.py --tag v1.19.19
              python prebuild.py --tag v1.19.19 --proxy http://127.0.0.1:PROXY_PORT

            Next:
              python build.py --tag v1.19.19 --arch all
            """
        ),
    )
    parser.add_argument("--repo", default="MetaCubeX/mihomo", help="Upstream repository (default: MetaCubeX/mihomo)")
    parser.add_argument("--tag", default="", help="Upstream release tag (default: latest)")
    parser.add_argument("--proxy", default="", help="HTTP/HTTPS proxy, e.g. http://127.0.0.1:PROXY_PORT")

    if not argv:
        parser.print_help()
        return 0

    args = parser.parse_args(argv)

    apply_proxy_env_from_args(args.proxy or None)
    apply_proxy_env_from_windows_system()

    tag = args.tag or fetch_latest_tag(args.repo)
    updated = ensure_mihomo_source(src_dir, upstream_repo=args.repo, tag=tag)
    if updated:
        print(f"[prebuild] mihomo-source switched to {tag}")
    else:
        print(f"[prebuild] mihomo-source already at {tag}, skipping update")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
