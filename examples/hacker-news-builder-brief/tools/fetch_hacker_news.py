#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys
import urllib.request


def fixture():
    path = pathlib.Path(__file__).resolve().parents[1] / "fixtures" / "top_stories.json"
    return json.loads(path.read_text())


def get_json(url):
    with urllib.request.urlopen(url, timeout=20) as response:
        return json.load(response)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--limit", type=int, default=30)
    args = parser.parse_args()
    try:
        ids = get_json("https://hacker-news.firebaseio.com/v0/topstories.json")[:args.limit]
        stories = [get_json(f"https://hacker-news.firebaseio.com/v0/item/{item_id}.json") for item_id in ids]
    except Exception as exc:
        print(f"live hacker news fetch failed, using fixture: {exc}", file=sys.stderr)
        stories = fixture()
    print(json.dumps({"source": "hacker-news-topstories", "stories": stories}, indent=2))


if __name__ == "__main__":
    main()
