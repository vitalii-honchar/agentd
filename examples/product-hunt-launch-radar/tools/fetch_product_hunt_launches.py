#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixture", default="fixtures/product_hunt_sample.json")
    args = parser.parse_args()
    root = pathlib.Path(__file__).resolve().parents[1]
    launches = json.loads((root / args.fixture).read_text())
    print(json.dumps({"source": "product-hunt-fixture", "launches": launches}, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"tool failed: {exc}", file=sys.stderr)
        sys.exit(1)
