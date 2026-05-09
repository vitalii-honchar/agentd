#!/usr/bin/env python3
import argparse
import json
import os
import pathlib
import sys
import urllib.request


def load_local_env():
    for parent in pathlib.Path(__file__).resolve().parents:
        env_path = parent / ".env"
        if not env_path.exists():
            continue
        for line in env_path.read_text().splitlines():
            stripped = line.strip()
            if not stripped or stripped.startswith("#") or "=" not in line:
                continue
            key, value = line.split("=", 1)
            os.environ.setdefault(key.strip(), value.strip().strip('"').strip("'"))
        return


def fetch_public_json(subreddit, limit):
    url = f"https://www.reddit.com/r/{subreddit}/new.json?limit={limit}"
    request = urllib.request.Request(url, headers={"User-Agent": "agentd-example/1.0"})
    with urllib.request.urlopen(request, timeout=20) as response:
        payload = json.load(response)
    posts = []
    for child in payload.get("data", {}).get("children", []):
        data = child.get("data", {})
        posts.append({
            "title": data.get("title", ""),
            "url": "https://www.reddit.com" + data.get("permalink", ""),
            "score": data.get("score", 0),
            "created_utc": data.get("created_utc", 0),
            "selftext": data.get("selftext", "")[:800],
        })
    return posts


def fetch_with_praw(subreddit, limit):
    import praw

    reddit = praw.Reddit(
        client_id=os.environ["REDDIT_CLIENT_ID"],
        client_secret=os.environ["REDDIT_CLIENT_SECRET"],
        user_agent=os.environ.get("REDDIT_USER_AGENT", "agentd-example/1.0"),
    )
    reddit.read_only = True
    return [{
        "title": post.title,
        "url": f"https://www.reddit.com{post.permalink}",
        "score": post.score,
        "created_utc": post.created_utc,
        "selftext": post.selftext[:800],
    } for post in reddit.subreddit(subreddit).new(limit=limit)]


def main():
    load_local_env()
    parser = argparse.ArgumentParser()
    parser.add_argument("--subreddit", default="cybersecurity")
    parser.add_argument("--limit", type=int, default=25)
    args = parser.parse_args()
    try:
        if os.getenv("REDDIT_CLIENT_ID") and os.getenv("REDDIT_CLIENT_SECRET"):
            posts = fetch_with_praw(args.subreddit, args.limit)
        else:
            posts = fetch_public_json(args.subreddit, args.limit)
    except Exception as exc:
        print(f"reddit fetch failed: {exc}", file=sys.stderr)
        sys.exit(1)
    print(json.dumps({"source": f"r/{args.subreddit}", "posts": posts}, indent=2))


if __name__ == "__main__":
    try:
        main()
    except Exception as exc:
        print(f"tool failed: {exc}", file=sys.stderr)
        sys.exit(1)
