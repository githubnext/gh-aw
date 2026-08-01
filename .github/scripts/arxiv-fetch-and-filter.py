"""
arxiv-fetch-and-filter.py

DataOps helper for daily-arxiv-researcher workflow.

Reads /tmp/gh-aw/agent/arxiv/raw.xml (fetched by curl in the step),
parses the Atom XML, compares against the cache-memory dedup list,
and writes only unseen papers to /tmp/gh-aw/agent/arxiv/new-papers.json.
"""

import json
import os
import re
import sys
import xml.etree.ElementTree as ET
from datetime import datetime, timezone

ATOM_NS = {"atom": "http://www.w3.org/2005/Atom"}
RAW_XML = "/tmp/gh-aw/agent/arxiv/raw.xml"
PAPERS_JSON = "/tmp/gh-aw/agent/arxiv/papers.json"
NEW_PAPERS_JSON = "/tmp/gh-aw/agent/arxiv/new-papers.json"
SEEN_IDS_JSON = "/tmp/gh-aw/cache-memory/seen-paper-ids.json"


def parse_papers():
    if not os.path.exists(RAW_XML):
        return []
    try:
        tree = ET.parse(RAW_XML)
        root = tree.getroot()
    except ET.ParseError as exc:
        print("XML parse error: " + str(exc), file=sys.stderr)
        return []

    papers = []
    for entry in root.findall("atom:entry", ATOM_NS):
        arxiv_id_url = entry.findtext("atom:id", "", ATOM_NS)
        arxiv_id = re.sub(r".*abs/", "", arxiv_id_url).strip()
        if not arxiv_id:
            continue
        title = " ".join((entry.findtext("atom:title", "", ATOM_NS) or "").split())
        summary = " ".join((entry.findtext("atom:summary", "", ATOM_NS) or "").split())
        published = (entry.findtext("atom:published", "", ATOM_NS) or "")[:10]
        authors = [
            a.findtext("atom:name", "", ATOM_NS)
            for a in entry.findall("atom:author", ATOM_NS)
        ]
        categories = [c.get("term", "") for c in entry.findall("atom:category", ATOM_NS)]
        papers.append(
            {
                "id": arxiv_id,
                "title": title,
                "abstract": summary[:1200],
                "authors": authors[:3],
                "published": published,
                "categories": categories[:3],
                "url": "https://arxiv.org/abs/" + arxiv_id,
            }
        )
    return papers


def load_seen_ids():
    if os.path.exists(SEEN_IDS_JSON):
        with open(SEEN_IDS_JSON) as f:
            return set(json.load(f).get("ids", []))
    return set()


def main():
    papers = parse_papers()
    fetched_at = datetime.now(timezone.utc).strftime("%Y-%m-%d")

    with open(PAPERS_JSON, "w") as f:
        json.dump({"fetched_at": fetched_at, "count": len(papers), "papers": papers}, f, indent=2)

    seen_ids = load_seen_ids()
    new_papers = [p for p in papers if p["id"] not in seen_ids]

    result = {
        "total_fetched": len(papers),
        "already_seen": len(papers) - len(new_papers),
        "new_count": len(new_papers),
        "fetched_at": fetched_at,
        "papers": new_papers,
    }
    with open(NEW_PAPERS_JSON, "w") as f:
        json.dump(result, f, indent=2)

    print("Parsed " + str(len(papers)) + " papers, " + str(len(new_papers)) + " new")


if __name__ == "__main__":
    main()
