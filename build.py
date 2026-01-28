#!/usr/bin/env python3

from __future__ import annotations

import argparse
import os
import platform
import re
import shutil
import subprocess
import sys
import textwrap
import time
from pathlib import Path


def run(cmd: list[str], *, cwd: Path | None = None, env: dict[str, str] | None = None) -> None:
    subprocess.run(cmd, cwd=cwd, env=env, check=True)


def run_capture(cmd: list[str], *, cwd: Path | None = None) -> str:
    return subprocess.check_output(cmd, cwd=cwd, text=True).strip()


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


def parse_version_dir_name(name: str) -> tuple[int, ...] | None:
    if not re.fullmatch(r"[0-9]+(\.[0-9]+)*", name):
        return None
    try:
        return tuple(int(x) for x in name.split("."))
    except ValueError:
        return None


def find_ndk_dir() -> Path:
    for key in ("ANDROID_NDK_HOME", "ANDROID_NDK", "ANDROID_NDK_ROOT"):
        value = os.environ.get(key)
        if value:
            p = Path(value).expanduser()
            if p.exists():
                return p

    sdk_root = os.environ.get("ANDROID_HOME") or os.environ.get("ANDROID_SDK_ROOT")
    if sdk_root:
        ndk_root = Path(sdk_root).expanduser() / "ndk"
        if ndk_root.exists():
            versions: list[tuple[tuple[int, ...], Path]] = []
            for child in ndk_root.iterdir():
                if not child.is_dir():
                    continue
                v = parse_version_dir_name(child.name)
                if v is None:
                    continue
                versions.append((v, child))
            if versions:
                versions.sort(reverse=True)
                return versions[0][1]

        ndk_bundle = Path(sdk_root).expanduser() / "ndk-bundle"
        if ndk_bundle.exists():
            return ndk_bundle

    raise RuntimeError("Android NDK not found, please set ANDROID_NDK_HOME/ANDROID_NDK/ANDROID_HOME")


def find_ndk_host_tag(ndk_dir: Path) -> str:
    prebuilt = ndk_dir / "toolchains" / "llvm" / "prebuilt"
    if not prebuilt.exists():
        raise RuntimeError(f"Invalid NDK layout: {prebuilt} does not exist")

    host_tags = [p.name for p in prebuilt.iterdir() if p.is_dir()]
    if not host_tags:
        raise RuntimeError(f"Invalid NDK layout: no host directory found under {prebuilt}")

    system = platform.system().lower()
    if system.startswith("windows"):
        preferred = "windows-x86_64"
    elif system.startswith("linux"):
        preferred = "linux-x86_64"
    elif system.startswith("darwin"):
        preferred = "darwin-x86_64"
    else:
        preferred = host_tags[0]

    if preferred in host_tags:
        return preferred
    return host_tags[0]


def clang_path(ndk_dir: Path, *, host_tag: str, triple: str, api: int) -> Path:
    bin_dir = ndk_dir / "toolchains" / "llvm" / "prebuilt" / host_tag / "bin"
    exe = f"{triple}{api}-clang"
    if platform.system().lower().startswith("windows"):
        exe = f"{exe}.cmd"
    p = bin_dir / exe
    if not p.exists():
        raise RuntimeError(f"clang not found: {p}")
    return p


def ensure_parent_dir(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def ndk_tool_path(ndk_dir: Path, *, host_tag: str, name: str) -> Path | None:
    bin_dir = ndk_dir / "toolchains" / "llvm" / "prebuilt" / host_tag / "bin"
    system = platform.system().lower()
    candidates: list[Path] = [bin_dir / name]
    if system.startswith("windows"):
        candidates = [
            bin_dir / f"{name}.exe",
            bin_dir / f"{name}.cmd",
            bin_dir / f"{name}.bat",
            bin_dir / name,
        ]
    for candidate in candidates:
        if candidate.exists():
            return candidate
    return None


def resolve_tool(
    name: str,
    *,
    ndk_dir: Path,
    host_tag: str,
    fallbacks: list[str] | None = None,
) -> str:
    p = ndk_tool_path(ndk_dir, host_tag=host_tag, name=name)
    if p is not None:
        return str(p)

    for fallback in fallbacks or []:
        found = shutil.which(fallback)
        if found:
            return found

    raise RuntimeError(f"tool not found: {name}")


def display_path(path: Path, *, repo_root: Path) -> str:
    try:
        return str(path.relative_to(repo_root))
    except ValueError:
        return str(path)


def format_elapsed(seconds: float) -> str:
    total = max(0.0, float(seconds))
    hours, rem = divmod(total, 3600.0)
    minutes, rem = divmod(rem, 60.0)
    if hours >= 1:
        return f"{int(hours)}h{int(minutes):02d}m{rem:04.1f}s"
    if minutes >= 1:
        return f"{int(minutes)}m{rem:04.1f}s"
    return f"{total:.1f}s"


def export_android_layout(*, repo_root: Path, export_dir: Path, artifacts: list[tuple[Path, str]]) -> None:
    bridge_header = repo_root / "android-wrapper" / "bridge.h"
    if not bridge_header.exists():
        raise RuntimeError(f"missing bridge header: {bridge_header}")

    for so_path, abi in artifacts:
        header_path = so_path.with_suffix(".h")
        if not header_path.exists():
            raise RuntimeError(f"missing generated C header: {header_path}")

        lib_dir = export_dir / abi
        include_dir = export_dir / "includes" / abi
        lib_dir.mkdir(parents=True, exist_ok=True)
        include_dir.mkdir(parents=True, exist_ok=True)

        shutil.copy2(so_path, lib_dir / "libclash.so")
        shutil.copy2(header_path, include_dir / "libclash.h")
        shutil.copy2(bridge_header, include_dir / "bridge.h")


EXPECTED_C_EXPORTS = {
    "invokeAction",
    "setEventListener",
    "suspend",
    "forceGC",
    "updateDns",
    "getTraffic",
    "getTotalTraffic",
    "startTUN",
    "stopTun",
}


def strip_shared_library(*, ndk_dir: Path, host_tag: str, so_path: Path) -> None:
    strip_bin = resolve_tool("llvm-strip", ndk_dir=ndk_dir, host_tag=host_tag, fallbacks=["llvm-strip", "strip"])
    run([strip_bin, "--strip-unneeded", str(so_path)])


def verify_shared_library(
    *,
    ndk_dir: Path,
    host_tag: str,
    so_path: Path,
    goarch: str,
) -> None:
    if not so_path.exists():
        raise RuntimeError(f"missing output file: {so_path}")
    if so_path.stat().st_size <= 0:
        raise RuntimeError(f"output file is empty: {so_path}")

    header_path = so_path.with_suffix(".h")
    if not header_path.exists():
        raise RuntimeError(f"missing C header: {header_path}")

    expected_machine = "AArch64" if goarch == "arm64" else "X86-64"

    readelf = resolve_tool(
        "llvm-readelf",
        ndk_dir=ndk_dir,
        host_tag=host_tag,
        fallbacks=["llvm-readelf", "readelf"],
    )
    elf_header = run_capture([readelf, "-h", str(so_path)])
    machine_line = next((line for line in elf_header.splitlines() if "Machine:" in line), "")
    if expected_machine not in machine_line:
        raise RuntimeError(f"ELF machine mismatch: {so_path}: {machine_line.strip() or 'unknown'}")

    nm = resolve_tool(
        "llvm-nm",
        ndk_dir=ndk_dir,
        host_tag=host_tag,
        fallbacks=["llvm-nm", "nm"],
    )
    nm_out = run_capture([nm, "-D", "--defined-only", str(so_path)])
    exports = {
        line.split()[-1].split("@", 1)[0]
        for line in nm_out.splitlines()
        if line.strip() and not line.startswith("nm:")
    }
    missing = sorted(EXPECTED_C_EXPORTS - exports)
    if missing:
        raise RuntimeError(f"missing exported symbols: {so_path}: {', '.join(missing)}")


def ensure_go_mod_files(wrapper_dir: Path) -> None:
    go_mod = wrapper_dir / "go.mod"
    if go_mod.exists():
        return
    raise RuntimeError(f"go.mod not found: {go_mod}")


def build_one(
    *,
    wrapper_dir: Path,
    ndk_dir: Path,
    host_tag: str,
    api: int,
    version: str,
    goarch: str,
    out_path: Path,
    mode: str,
) -> None:
    ensure_parent_dir(out_path)
    if out_path.exists():
        out_path.unlink()
    header_path = out_path.with_suffix(".h")
    if header_path.exists():
        header_path.unlink()

    env = dict(os.environ)
    env["CGO_ENABLED"] = "1"
    env["GOOS"] = "android"
    env["GOARCH"] = goarch

    if goarch == "arm64":
        cc = clang_path(ndk_dir, host_tag=host_tag, triple="aarch64-linux-android", api=api)
    elif goarch == "amd64":
        cc = clang_path(ndk_dir, host_tag=host_tag, triple="x86_64-linux-android", api=api)
    else:
        raise RuntimeError(f"unsupported GOARCH: {goarch}")

    env["CC"] = str(cc)

    # Go 1.24+ may refuse to write go.mod in some environments.
    # Build with a temporary -modfile to avoid touching android-wrapper/go.mod.
    modfile_path = wrapper_dir / "go.build.mod"
    sumfile_path = wrapper_dir / "go.build.sum"

    if sumfile_path.exists():
        sumfile_path.unlink()
    shutil.copyfile(wrapper_dir / "go.mod", modfile_path)

    try:
        run(["go", "mod", "download", "-modfile", str(modfile_path), "all"], cwd=wrapper_dir, env=env)
        ldflags_parts: list[str] = []
        build_args = [
            "go",
            "build",
            "-mod=mod",
            "-modfile",
            str(modfile_path),
            "-buildmode=c-shared",
            "-tags",
            "with_gvisor,cmfa",
        ]

        if mode == "release":
            build_args.append("-trimpath")
            ldflags_parts.extend(["-s", "-w", "-buildid="])
        elif mode == "debug":
            pass
        else:
            raise RuntimeError(f"unsupported build mode: {mode}")

        ldflags_parts.append(f"-X github.com/metacubex/mihomo/constant.Version={version}")
        ldflags = " ".join(ldflags_parts)
        build_args.extend([f"-ldflags={ldflags}", "-o", str(out_path)])
        run(build_args, cwd=wrapper_dir, env=env)

        if mode == "release":
            strip_shared_library(ndk_dir=ndk_dir, host_tag=host_tag, so_path=out_path)
    finally:
        try:
            modfile_path.unlink()
        except FileNotFoundError:
            pass
        try:
            sumfile_path.unlink()
        except FileNotFoundError:
            pass


def resolve_mihomo_tag(src_dir: Path) -> str:
    if not (src_dir / ".git").exists():
        return ""

    for cmd in (
        ["git", "describe", "--tags", "--exact-match"],
        ["git", "describe", "--tags", "--abbrev=0"],
    ):
        try:
            return run_capture(cmd, cwd=src_dir)
        except subprocess.CalledProcessError:
            continue
    return ""


def main(argv: list[str] | None = None) -> int:
    argv = sys.argv[1:] if argv is None else argv

    repo_root = Path(__file__).resolve().parent
    wrapper_dir = repo_root / "android-wrapper"
    src_dir = repo_root / "mihomo-source"
    started_at = time.perf_counter()

    parser = argparse.ArgumentParser(
        description="Build step: compile Android shared libraries (.so) from local mihomo-source/.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=textwrap.dedent(
            """\
            Workflow:
              1) python prebuild.py --tag v1.19.19
              2) python build.py --tag v1.19.19 --arch all

            Examples:
              python build.py --tag v1.19.19 --arch arm64
              python build.py --tag v1.19.19 --arch arm64 --mode release
              python build.py --verify-only --arch all
            """
        ),
    )
    parser.add_argument(
        "--tag",
        default="",
        help="Mihomo tag (embedded into version); default: auto-detect from mihomo-source/",
    )
    parser.add_argument("--api", type=int, default=31, help="Android API level (default: 31)")
    parser.add_argument("--proxy", default="", help="HTTP/HTTPS proxy, e.g. http://127.0.0.1:PROXY_PORT")
    parser.add_argument(
        "--arch",
        default="all",
        choices=["all", "arm64", "x86_64"],
        help="Target arch: all/arm64/x86_64 (default: all)",
    )
    parser.add_argument(
        "--mode",
        default="debug",
        choices=["debug", "release"],
        help="Build mode: debug/release (default: debug; release strips symbols to reduce .so size)",
    )
    parser.add_argument("--out-dir", default="dist", help="Output directory (default: dist)")
    parser.add_argument(
        "--export-android",
        nargs="?",
        const="libclash/android",
        default="",
        metavar="DIR",
        help="Also export an Android-friendly directory layout into DIR (default: libclash/android)",
    )
    try:
        bool_action = argparse.BooleanOptionalAction  # type: ignore[attr-defined]
    except AttributeError:
        bool_action = None

    if bool_action is None:
        parser.add_argument("--verify", dest="verify", action="store_true", default=True, help="Verify output .so")
        parser.add_argument("--no-verify", dest="verify", action="store_false", help="Skip output verification")
    else:
        parser.add_argument(
            "--verify",
            action=bool_action,
            default=True,
            help="Verify .so exported symbols/arch (default: true)",
        )
    parser.add_argument("--verify-only", action="store_true", help="Only verify outputs and exit")

    if not argv:
        parser.print_help()
        return 0

    args = parser.parse_args(argv)

    try:
        apply_proxy_env_from_args(args.proxy or None)
        apply_proxy_env_from_windows_system()

        ensure_go_mod_files(wrapper_dir)

        out_dir = Path(args.out_dir).expanduser()
        if not out_dir.is_absolute():
            out_dir = repo_root / out_dir

        export_dir: Path | None = None
        if args.export_android:
            export_dir = Path(args.export_android).expanduser()
            if not export_dir.is_absolute():
                export_dir = repo_root / export_dir

        ndk_dir = find_ndk_dir()
        host_tag = find_ndk_host_tag(ndk_dir)
        print(f"[builder] NDK={ndk_dir} host={host_tag} API={args.api}")

        arm64_out = out_dir / "libclash_arm64.so"
        x86_64_out = out_dir / "libclash_x86_64.so"

        export_artifacts: list[tuple[Path, str]] = []
        if args.arch in ("all", "arm64"):
            export_artifacts.append((arm64_out, "arm64-v8a"))
        if args.arch in ("all", "x86_64"):
            export_artifacts.append((x86_64_out, "x86_64"))

        if args.verify_only:
            if args.arch in ("all", "arm64"):
                verify_shared_library(ndk_dir=ndk_dir, host_tag=host_tag, so_path=arm64_out, goarch="arm64")
                print(f"[builder] verified: {display_path(arm64_out, repo_root=repo_root)}")
            if args.arch in ("all", "x86_64"):
                verify_shared_library(ndk_dir=ndk_dir, host_tag=host_tag, so_path=x86_64_out, goarch="amd64")
                print(f"[builder] verified: {display_path(x86_64_out, repo_root=repo_root)}")
            if export_dir is not None:
                export_android_layout(repo_root=repo_root, export_dir=export_dir, artifacts=export_artifacts)
                print(f"[builder] exported Android layout: {display_path(export_dir, repo_root=repo_root)}")
            return 0

        if not src_dir.exists():
            raise RuntimeError(f"mihomo-source is missing: {src_dir} (run: python prebuild.py --tag <tag>)")
        if not (src_dir / ".git").exists():
            raise RuntimeError(f"mihomo-source is not a git repository: {src_dir} (recreate it via prebuild.py)")

        tag = args.tag or resolve_mihomo_tag(src_dir)
        if tag:
            print(f"[builder] mihomo-source tag: {tag}")
        else:
            print("[builder] mihomo-source tag: unknown (use --tag to set version)")

        version = tag[1:] if tag.startswith("v") else tag

        if args.arch in ("all", "arm64"):
            build_started_at = time.perf_counter()
            build_one(
                wrapper_dir=wrapper_dir,
                ndk_dir=ndk_dir,
                host_tag=host_tag,
                api=args.api,
                version=version,
                goarch="arm64",
                out_path=arm64_out,
                mode=args.mode,
            )
            if args.verify:
                verify_shared_library(ndk_dir=ndk_dir, host_tag=host_tag, so_path=arm64_out, goarch="arm64")
                print(f"[builder] verified: {display_path(arm64_out, repo_root=repo_root)}")
            print(f"[builder] built: {display_path(arm64_out, repo_root=repo_root)}")
            print(f"[builder] build time (arm64): {format_elapsed(time.perf_counter() - build_started_at)}")

        if args.arch in ("all", "x86_64"):
            build_started_at = time.perf_counter()
            build_one(
                wrapper_dir=wrapper_dir,
                ndk_dir=ndk_dir,
                host_tag=host_tag,
                api=args.api,
                version=version,
                goarch="amd64",
                out_path=x86_64_out,
                mode=args.mode,
            )
            if args.verify:
                verify_shared_library(ndk_dir=ndk_dir, host_tag=host_tag, so_path=x86_64_out, goarch="amd64")
                print(f"[builder] verified: {display_path(x86_64_out, repo_root=repo_root)}")
            print(f"[builder] built: {display_path(x86_64_out, repo_root=repo_root)}")
            print(f"[builder] build time (x86_64): {format_elapsed(time.perf_counter() - build_started_at)}")

        if export_dir is not None:
            export_android_layout(repo_root=repo_root, export_dir=export_dir, artifacts=export_artifacts)
            print(f"[builder] exported Android layout: {display_path(export_dir, repo_root=repo_root)}")

        return 0
    finally:
        print(f"[builder] total time: {format_elapsed(time.perf_counter() - started_at)}")


if __name__ == "__main__":
    raise SystemExit(main())
