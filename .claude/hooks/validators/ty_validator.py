#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""
Type Checker Validator for Claude Code PostToolUse Hook

Runs `uvx ty check <file_path>` for type checking on single Python files.

Outputs JSON decision for Claude Code PostToolUse hook:
- {"decision": "block", "reason": "..."} to block and retry
- {} to allow completion
"""
import json
import logging
import subprocess
import sys
from pathlib import Path

# Logging setup - log file next to this script
SCRIPT_DIR = Path(__file__).parent
LOG_FILE = SCRIPT_DIR / "ty_validator.log"

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s | %(levelname)s | %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[logging.FileHandler(LOG_FILE, mode='a')]
)
logger = logging.getLogger(__name__)


def main():
    logger.info("=" * 50)
    logger.info("TY VALIDATOR POSTTOOLUSE HOOK TRIGGERED")

    # Read hook input from stdin (Claude Code passes JSON)
    try:
        stdin_data = sys.stdin.read()
        if stdin_data.strip():
            hook_input = json.loads(stdin_data)
            logger.info(f"hook_input keys: {list(hook_input.keys())}")
        else:
            hook_input = {}
    except json.JSONDecodeError:
        hook_input = {}

    # Extract file_path from PostToolUse stdin
    file_path = hook_input.get("tool_input", {}).get("file_path", "")
    logger.info(f"file_path: {file_path}")

    # Only run for Python files
    if not file_path.endswith(".py"):
        logger.info("Skipping non-Python file")
        print(json.dumps({}))
        return

    # Run uvx ty check on the single file
    logger.info(f"Running: uvx ty check {file_path}")
    try:
        result = subprocess.run(
            ["uvx", "ty", "check", file_path],
            capture_output=True,
            text=True,
            timeout=120
        )

        stdout = result.stdout.strip()
        stderr = result.stderr.strip()

        if stdout:
            for line in stdout.split('\n')[:20]:  # Limit log lines
                logger.info(f"  {line}")

        if result.returncode == 0:
            logger.info("RESULT: PASS - Type check successful")
            print(json.dumps({}))
        else:
            logger.info(f"RESULT: BLOCK (exit code {result.returncode})")
            if stderr:
                for line in stderr.split('\n')[:10]:
                    logger.info(f"  ERROR: {line}")
            error_output = stderr or stdout or "Type check failed"
            print(json.dumps({
                "decision": "block",
                "reason": f"Type check failed:\n{error_output[:500]}"
            }))

    except subprocess.TimeoutExpired:
        logger.info("RESULT: BLOCK (timeout)")
        print(json.dumps({
            "decision": "block",
            "reason": "Type check timed out after 120 seconds"
        }))
    except FileNotFoundError:
        logger.info("RESULT: PASS (uvx ty not found, skipping)")
        print(json.dumps({}))


if __name__ == "__main__":
    main()
