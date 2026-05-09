#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys
import urllib.parse
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[1]


def fixture():
    return json.loads((ROOT / "fixtures" / "trending_repos.json").read_text())


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--languages", default="sources/languages.txt")
    args = parser.parse_args()
    try:
        languages = [line.strip() for line in (ROOT / args.languages).read_text().splitlines() if line.strip()]
        repos = []
        for language in languages:
            query = urllib.parse.quote(f"language:{language} stars:>500 pushed:>2025-01-01")
            url = f"https://api.github.com/search/repositories?q={query}&sort=stars&order=desc&per_page=3"
            request = urllib.request.Request(url, headers={"User-Agent": "agentd-example/1.0"})
            with urllib.request.urlopen(request, timeout=20) as response:
                payload = json.load(response)
            for item in payload.get("items", []):
                repos.append({
                    "name": item.get("full_name", ""),
                    "description": item.get("description", ""),
                    "url": item.get("html_url", ""),
                    "stars": item.get("stargazers_count", 0),
                    "language": item.get("language", ""),
                })
    except Exception as exc:
        print(f"live github fetch failed, using fixture: {exc}", file=sys.stderr)
        repos = fixture()
    print(json.dumps({"source": "github-search", "repositories": repos}, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"tool failed: {exc}", file=sys.stderr)
        sys.exit(1)
