#!/usr/bin/env python3
"""Regenerate licenses/THIRD_PARTY_LICENSES_{NPM,GO,RUST}.md.

Requires: npm (frontend deps installed, `npx license-checker`), go, cargo.
Usage: python3 scripts/generate_licenses.py
"""
import json, os, subprocess, sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUT = os.path.join(ROOT, "licenses")
LICENSE_PREFIXES = ("LICENSE", "LICENCE", "COPYING", "NOTICE", "UNLICENSE")


def run(cmd, cwd, **kw):
    return subprocess.run(cmd, cwd=cwd, check=True, capture_output=True, text=True, **kw).stdout


def license_files(directory):
    return sorted(
        f for f in os.listdir(directory)
        if f.upper().startswith(LICENSE_PREFIXES) and os.path.isfile(os.path.join(directory, f))
    )


def section(title, header, body):
    return f"## {title}\n\n{header}\n\n{body}\n"


def read(path):
    with open(path, errors="replace") as fh:
        return "```text\n" + fh.read().rstrip() + "\n```"


def npm():
    data = json.loads(run(["npx", "license-checker", "--production", "--json"], os.path.join(ROOT, "frontend")))
    out = ["# Third-party licenses – Frontend (npm, production dependencies)\n"]
    for name in sorted(data):
        v = data[name]
        if name.startswith("open-clubmanager@"):
            continue
        lf = v.get("licenseFile")
        body = read(lf) if lf and os.path.isfile(lf) and not lf.lower().endswith("readme.md") else "_(no license file shipped; see identifier)_"
        out.append(section(name, f"License: `{v['licenses']}`  \nSource: {v.get('repository', '')}", body))
    return "\n".join(out)


def go():
    server = os.path.join(ROOT, "server")
    fmt = "{{with .Module}}{{if .Dir}}{{.Path}}@{{.Version}}|{{.Dir}}{{end}}{{end}}"
    lines = sorted(set(run(["go", "list", "-deps", "-f", fmt, "./cmd/server"], server).split()))
    out = ["# Third-party licenses – Go server\n"]
    for line in lines:
        mod, directory = line.split("|", 1)
        if mod.startswith("github.com/Waschndolos/"):
            continue
        files = license_files(directory)
        body = "\n\n".join(f"### {f}\n\n{read(os.path.join(directory, f))}" for f in files) or "_(no license file found)_"
        out.append(section(mod, "", body))
    return "\n".join(out)


def rust():
    meta = json.loads(run(["cargo", "metadata", "--format-version", "1", "--locked"], os.path.join(ROOT, "src-tauri")))
    out = [
        "# Third-party licenses – Rust / Tauri desktop shell\n",
        "Where a crate offers several licenses (`OR`), any of them may be chosen. MPL-2.0 crates are file-level copyleft and compatible with AGPL-3.0.\n",
    ]
    for p in sorted((p for p in meta["packages"] if p["source"]), key=lambda p: p["name"]):
        directory = os.path.dirname(p["manifest_path"])
        files = license_files(directory)
        body = "\n\n".join(f"### {f}\n\n{read(os.path.join(directory, f))}" for f in files) or "_(no license file shipped in crate; see identifier)_"
        out.append(section(f"{p['name']} {p['version']}", f"License: `{p.get('license')}`  \nSource: {p.get('repository') or p.get('homepage') or p['source']}", body))
    return "\n".join(out)


if __name__ == "__main__":
    os.makedirs(OUT, exist_ok=True)
    for name, fn in (("NPM", npm), ("GO", go), ("RUST", rust)):
        with open(os.path.join(OUT, f"THIRD_PARTY_LICENSES_{name}.md"), "w") as fh:
            fh.write(fn())
        print("wrote", name, file=sys.stderr)
