#!/usr/bin/env python3
import argparse
import json
import pathlib
import sys
import urllib.request


ROOT = pathlib.Path(__file__).resolve().parents[1]


def fixture():
    return json.loads((ROOT / "fixtures" / "pain_posts.json").read_text())


def fetch_subreddit(name, per_subreddit):
    url = f"https://www.reddit.com/r/{name}/new.json?limit={per_subreddit}"
    request = urllib.request.Request(url, headers={"User-Agent": "agentd-example/1.0"})
    with urllib.request.urlopen(request, timeout=15) as response:
        payload = json.load(response)
    posts = []
    for child in payload.get("data", {}).get("children", []):
        data = child.get("data", {})
        posts.append({
            "subreddit": name,
            "title": data.get("title", ""),
            "url": "https://www.reddit.com" + data.get("permalink", ""),
            "score": data.get("score", 0),
            "selftext": data.get("selftext", "")[:700],
        })
    return posts


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--sources", default="sources/subreddits.txt")
    parser.add_argument("--limit", type=int, default=40)
    args = parser.parse_args()
    try:
        subreddits = [line.strip() for line in (ROOT / args.sources).read_text().splitlines() if line.strip()]
        per_subreddit = max(1, args.limit // max(1, len(subreddits)))
        posts = []
        for subreddit in subreddits:
            posts.extend(fetch_subreddit(subreddit, per_subreddit))
    except Exception as exc:
        print(f"live reddit pain fetch failed, using fixture: {exc}", file=sys.stderr)
        posts = fixture()
    print(json.dumps({"source": "reddit-customer-pain", "posts": posts[:args.limit]}, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"tool failed: {exc}", file=sys.stderr)
        sys.exit(1)
