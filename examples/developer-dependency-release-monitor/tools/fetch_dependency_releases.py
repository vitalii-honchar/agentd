#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[1]


def fixture():
    return json.loads((ROOT / "fixtures" / "releases.json").read_text())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sources", default="sources/dependencies.json")
    args = parser.parse_args()
    try:
        sources = json.loads((ROOT / args.sources).read_text())
        releases = []
        for source in sources:
            request = urllib.request.Request(source["url"], headers={"User-Agent": "agentd-example/1.0"})
            with urllib.request.urlopen(request, timeout=20) as response:
                payload = json.load(response)
            releases.append({
                "name": source["name"],
                "kind": source["kind"],
                "url": source["url"],
                "version": payload.get("version") or payload.get("tag_name") or payload.get("info", {}).get("version", ""),
                "notes": payload.get("body", "")[:1000],
            })
    except Exception as exc:
        print(f"live dependency fetch failed, using fixture: {exc}", file=sys.stderr)
        releases = fixture()
    print(json.dumps({"source": "dependency-releases", "releases": releases}, indent=2))


if __name__ == "__main__":
    main()
