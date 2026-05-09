#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sources", default="sources/hiring_sources.json")
    args = parser.parse_args()
    root = pathlib.Path(__file__).resolve().parents[1]
    sources = json.loads((root / args.sources).read_text())
    fixture = json.loads((root / "fixtures" / "hiring_signals.json").read_text())
    print(json.dumps({"source": "ai-hiring-signals", "sources": sources, "signals": fixture}, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"tool failed: {exc}", file=sys.stderr)
        sys.exit(1)
