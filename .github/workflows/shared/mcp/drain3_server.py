#!/usr/bin/env python3
# Drain3 MCP HTTP server — live streaming JSONL
# Tools: index_file, query_file, list_templates
# Deps: pip install fastmcp drain3
from __future__ import annotations
from typing import Any, Dict, Iterable, List, Optional
import os, json, time, sys, logging
from pathlib import Path

from fastmcp import FastMCP
from drain3 import TemplateMiner
from drain3.file_persistence import FilePersistence
from drain3.template_miner_config import TemplateMinerConfig

# -----------------------
# Logging Configuration
# -----------------------
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s [%(levelname)s] %(name)s: %(message)s',
    stream=sys.stderr
)
logger = logging.getLogger(__name__)

# -----------------------
# Configuration
# -----------------------
HOST = os.getenv("HOST", "0.0.0.0")
PORT = int(os.getenv("PORT", "8766"))

logger.info(f"Initializing Drain3 MCP server")
logger.info(f"Configuration: HOST={HOST}, PORT={PORT}")

STATE_DIR = Path(os.getenv("STATE_DIR", ".drain3")).resolve()
STATE_DIR.mkdir(parents=True, exist_ok=True)
logger.info(f"State directory: {STATE_DIR}")

SIM_TH = float(os.getenv("DRAIN3_SIM_TH", "0.4"))
DEPTH = int(os.getenv("DRAIN3_DEPTH", "4"))
MAX_CHILDREN = int(os.getenv("DRAIN3_MAX_CHILDREN", "100"))
MAX_CLUSTERS = int(os.getenv("DRAIN3_MAX_CLUSTERS", "0"))

logger.info(f"Drain3 config: SIM_TH={SIM_TH}, DEPTH={DEPTH}, MAX_CHILDREN={MAX_CHILDREN}, MAX_CLUSTERS={MAX_CLUSTERS}")

# Stream tuning
STREAM_FLUSH_EVERY = int(os.getenv("STREAM_FLUSH_EVERY", "500"))  # emit a progress event every N lines
STREAM_SLEEP = float(os.getenv("STREAM_SLEEP", "0"))              # throttle (seconds) between flushes; 0 = no sleep

logger.info(f"Stream config: FLUSH_EVERY={STREAM_FLUSH_EVERY}, SLEEP={STREAM_SLEEP}")

logger.info("Creating FastMCP instance")
mcp = FastMCP("drain3-http")
logger.info("FastMCP instance created successfully")

# -----------------------
# Helpers
# -----------------------
def _snapshot_path_for(file_path: Path) -> Path:
    safe_stem = file_path.name.replace("/", "_")
    return STATE_DIR / f"{safe_stem}.snapshot.json"

def _build_config() -> TemplateMinerConfig:
    cfg = TemplateMinerConfig()
    cfg.set("DRAIN", "sim_th", str(SIM_TH))
    cfg.set("DRAIN", "depth", str(DEPTH))
    cfg.set("DRAIN", "max_children", str(MAX_CHILDREN))
    if MAX_CLUSTERS > 0:
        cfg.set("DRAIN", "max_clusters", str(MAX_CLUSTERS))
    # Sensible masking to improve template quality
    cfg.set("MASKING", "masking", json.dumps([
        {"regex_pattern": r"((?<=[^A-Za-z0-9])|^)(\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3})((?=[^A-Za-z0-9])|$)", "mask_with": "IP"},
        {"regex_pattern": r"((?<=[^A-Za-z0-9])|^)([\-+]?\d+)((?=[^A-Za-z0-9])|$)", "mask_with": "NUM"},
        {"regex_pattern": r"0x[0-9a-fA-F]+", "mask_with": "HEX"},
        {"regex_pattern": r"[a-f0-9]{8}\-[a-f0-9\-]{27,36}", "mask_with": "GUID"},
    ]))
    return cfg

def _new_miner(snapshot_path: Path) -> TemplateMiner:
    return TemplateMiner(FilePersistence(str(snapshot_path)), _build_config())

def _read_lines(p: Path, encoding="utf-8") -> Iterable[str]:
    with p.open("r", encoding=encoding, errors="ignore") as f:
        for ln in f:
            yield ln.rstrip("\n")

def _clusters_as_dicts(miner: TemplateMiner, limit: Optional[int] = None) -> List[Dict[str, Any]]:
    clusters = getattr(miner.drain, "clusters", []) or []
    if limit:
        clusters = clusters[:limit]
    return [
        {
            "cluster_id": getattr(c, "cluster_id", None),
            "size": getattr(c, "size", None),
            "template": " ".join(getattr(c, "log_template_tokens", []) or [])
        }
        for c in clusters
    ]

def _jsonl(obj: Any) -> str:
    return json.dumps(obj, ensure_ascii=False) + "\n"

# -----------------------
# MCP tools (streaming)
# -----------------------
@mcp.tool()
def index_file(path: str, encoding: str = "utf-8", max_lines: Optional[int] = None):
    """
    Stream-mines templates from a log file and persists a Drain3 snapshot.
    Yields JSONL lines progressively:
      - {"event":"start", ...}
      - {"event":"progress", processed:<n>}
      - {"event":"template", cluster_id, size, template}
      - {"event":"summary", cluster_count, processed_lines, ...}
    """
    logger.info(f"index_file called: path={path}, encoding={encoding}, max_lines={max_lines}")
    p = Path(path).expanduser().resolve()
    if not p.exists() or not p.is_file():
        logger.error(f"File not found: {str(p)}")
        yield _jsonl({"event": "error", "error": f"File not found: {str(p)}"})
        return
    
    logger.info(f"File found: {str(p)}")

    snapshot = _snapshot_path_for(p)
    logger.info(f"Snapshot path: {str(snapshot)}")
    miner = _new_miner(snapshot)
    logger.info("Template miner created")

    yield _jsonl({"event": "start", "file": str(p), "snapshot": str(snapshot)})
    logger.info("Started processing file")

    processed = 0
    for processed, ln in enumerate(_read_lines(p, encoding), start=1):
        if max_lines and processed > max_lines:
            processed -= 1  # last increment doesn't count
            break
        if ln.strip():
            miner.add_log_message(ln)

        if processed % STREAM_FLUSH_EVERY == 0:
            yield _jsonl({"event": "progress", "processed": processed})
            if STREAM_SLEEP > 0:
                time.sleep(STREAM_SLEEP)

    # Save at end (older Drain3 may auto-save, but we try explicitly)
    try:
        miner.save_state("manual_save")
    except Exception:
        pass

    clusters = _clusters_as_dicts(miner)
    # Emit clusters as independent events so consumers can start handling immediately
    for c in clusters:
        yield _jsonl({"event": "template", **c})

    yield _jsonl({
        "event": "summary",
        "file": str(p),
        "snapshot": str(snapshot),
        "processed_lines": processed,
        "cluster_count": len(clusters),
    })

@mcp.tool()
def query_file(path: str, text: str):
    """
    Streams a single JSONL event with the match result:
      - {"event":"query", cluster_id, cluster_size, template, ...}
    """
    logger.info(f"query_file called: path={path}, text_len={len(text)}")
    p = Path(path).expanduser().resolve()
    snapshot = _snapshot_path_for(p)
    if not snapshot.exists():
        logger.error(f"No snapshot found for {str(p)}")
        yield _jsonl({"event": "error", "error": f"No snapshot for {str(p)}. Run index_file first.", "file": str(p)})
        return
    
    logger.info(f"Snapshot exists: {str(snapshot)}")

    miner = _new_miner(snapshot)
    result = miner.match(text)
    if result is None:
        yield _jsonl({"event": "query", "file": str(p), "snapshot": str(snapshot),
                      "cluster_id": None, "cluster_size": None, "template": None})
        return

    cluster = result[0]
    yield _jsonl({
        "event": "query",
        "file": str(p),
        "snapshot": str(snapshot),
        "cluster_id": getattr(cluster, "cluster_id", None),
        "cluster_size": getattr(cluster, "size", None),
        "template": " ".join(getattr(cluster, "log_template_tokens", []) or []),
    })

@mcp.tool()
def list_templates(path: str, limit: Optional[int] = None):
    """
    Streams templates from an existing snapshot:
      - one {"event":"template", ...} per cluster
      - final {"event":"summary", count, ...}
    """
    logger.info(f"list_templates called: path={path}, limit={limit}")
    p = Path(path).expanduser().resolve()
    snapshot = _snapshot_path_for(p)
    if not snapshot.exists():
        logger.error(f"No snapshot found for {str(p)}")
        yield _jsonl({"event": "error", "error": f"No snapshot for {str(p)}. Run index_file first.", "file": str(p)})
        return
    
    logger.info(f"Snapshot exists: {str(snapshot)}")

    miner = _new_miner(snapshot)
    clusters = _clusters_as_dicts(miner, limit)
    for c in clusters:
        yield _jsonl({"event": "template", "file": str(p), "snapshot": str(snapshot), **c})

    yield _jsonl({"event": "summary", "file": str(p), "snapshot": str(snapshot), "count": len(clusters)})

# -----------------------
# Entry point
# -----------------------
if __name__ == "__main__":
    # HTTP transport for self-hosted MCP server
    # Note: Do NOT use 'fastmcp run' which defaults to stdio transport
    # For HTTP, run this script directly with Python
    logger.info("="*60)
    logger.info("Starting Drain3 MCP HTTP Server")
    logger.info(f"Host: {HOST}")
    logger.info(f"Port: {PORT}")
    logger.info(f"Transport: http")
    logger.info("="*60)
    
    try:
        logger.info("Calling mcp.run()...")
        mcp.run(transport="http", host=HOST, port=PORT)
        logger.info("mcp.run() completed")
    except Exception as e:
        logger.error(f"Server failed with exception: {e}", exc_info=True)
        raise
