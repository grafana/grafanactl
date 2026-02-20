#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///

"""
Generic validator that checks if a new file was created in a specified directory.

Checks:
1. Git status for untracked/new files matching the pattern
2. File modification time within the specified age

Exit codes:
- 0: Validation passed (new file found)
- 1: Validation failed (no new file found)

Usage:
  uv run validate_new_file.py --directory specs --extension .md
  uv run validate_new_file.py -d output -e .json --max-age 10
"""

import argparse
import json
import logging
import subprocess
import sys
import time
from datetime import datetime
from pathlib import Path

# Logging setup - log file next to this script
SCRIPT_DIR = Path(__file__).parent
LOG_FILE = SCRIPT_DIR / "validate_new_file.log"

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s | %(levelname)s | %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[
        logging.FileHandler(LOG_FILE, mode='a'),
    ]
)
logger = logging.getLogger(__name__)

# Constants
DEFAULT_DIRECTORY = "specs"
DEFAULT_EXTENSION = ".md"
DEFAULT_MAX_AGE_MINUTES = 5

NO_FILE_ERROR = (
    "VALIDATION FAILED: No new file found matching {pattern}.\n\n"
    "ACTION REQUIRED: Use the Write tool to create a new file in the {directory}/ directory. "
    "The file must match the expected pattern ({pattern}). "
    "Do not stop until the file has been created."
)


def get_git_untracked_files(directory: str, extension: str) -> list[str]:
    """Get list of untracked files in directory from git."""
    try:
        result = subprocess.run(
            ["git", "status", "--porcelain", f"{directory}/"],
            capture_output=True,
            text=True,
            timeout=5
        )
        if result.returncode != 0:
            logger.info(f"git status returned non-zero: {result.returncode}")
            return []

        untracked = []
        for line in result.stdout.strip().split('\n'):
            if not line:
                continue
            # Git status format: XY filename
            # ?? = untracked, A = added, M = modified
            status = line[:2]
            filepath = line[3:].strip()

            # Check for new/untracked files with matching extension
            if status in ('??', 'A ', ' A', 'AM') and filepath.endswith(extension):
                untracked.append(filepath)

        logger.info(f"Git untracked files: {untracked}")
        return untracked
    except (subprocess.TimeoutExpired, subprocess.SubprocessError) as e:
        logger.warning(f"Git command failed: {e}")
        return []


def get_recent_files(directory: str, extension: str, max_age_minutes: int) -> list[str]:
    """Get list of files in directory modified within the last N minutes."""
    target_dir = Path(directory)
    if not target_dir.exists():
        return []

    recent = []
    now = time.time()
    max_age_seconds = max_age_minutes * 60

    # Handle extension with or without leading dot
    ext = extension if extension.startswith('.') else f'.{extension}'
    pattern = f"*{ext}"

    for filepath in target_dir.glob(pattern):
        try:
            mtime = filepath.stat().st_mtime
            age = now - mtime
            if age <= max_age_seconds:
                recent.append(str(filepath))
        except OSError:
            continue

    return recent


def validate_new_file(directory: str, extension: str, max_age_minutes: int) -> tuple[bool, str]:
    """
    Validate that a new file was created.

    Args:
        directory: Directory to check for new files
        extension: File extension to match (e.g., '.md', '.json')
        max_age_minutes: Maximum age in minutes for "recent" files

    Returns:
        tuple: (success: bool, message: str)
    """
    pattern = f"{directory}/*{extension}"
    logger.info(f"Validating: directory={directory}, extension={extension}, max_age={max_age_minutes}min")

    # Check git for untracked/new files
    git_new = get_git_untracked_files(directory, extension)
    logger.info(f"Git new files: {git_new}")

    # Check for recently modified files
    recent_files = get_recent_files(directory, extension, max_age_minutes)
    logger.info(f"Recent files: {recent_files}")

    # If git shows new files, that's a strong signal
    if git_new:
        msg = f"New file(s) found: {', '.join(git_new)}"
        logger.info(f"PASS: {msg}")
        return True, msg

    # If no git new files, check if there are any recent files
    if recent_files:
        msg = f"Recently created file(s) found: {', '.join(recent_files)}"
        logger.info(f"PASS: {msg}")
        return True, msg

    msg = NO_FILE_ERROR.format(pattern=pattern, directory=directory)
    logger.warning(f"FAIL: {msg}")
    return False, msg


def parse_args() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Validate that a new file was created in a directory"
    )
    parser.add_argument(
        '-d', '--directory',
        type=str,
        default=DEFAULT_DIRECTORY,
        help=f'Directory to check for new files (default: {DEFAULT_DIRECTORY})'
    )
    parser.add_argument(
        '-e', '--extension',
        type=str,
        default=DEFAULT_EXTENSION,
        help=f'File extension to match (default: {DEFAULT_EXTENSION})'
    )
    parser.add_argument(
        '--max-age',
        type=int,
        default=DEFAULT_MAX_AGE_MINUTES,
        help=f'Maximum file age in minutes (default: {DEFAULT_MAX_AGE_MINUTES})'
    )
    return parser.parse_args()


def main():
    """Main entry point for the validator."""
    logger.info("=" * 60)
    logger.info("Validator started")

    try:
        # Parse CLI arguments
        args = parse_args()
        logger.info(f"Args: directory={args.directory}, extension={args.extension}, max_age={args.max_age}")

        # Read hook input from stdin (if provided)
        try:
            input_data = json.load(sys.stdin)
            logger.info(f"Stdin input received: {len(json.dumps(input_data))} bytes")
        except (json.JSONDecodeError, EOFError):
            input_data = {}
            logger.info("No stdin input or invalid JSON")

        # Run validation
        success, message = validate_new_file(
            directory=args.directory,
            extension=args.extension,
            max_age_minutes=args.max_age
        )

        if success:
            result = {"result": "continue", "message": message}
            logger.info(f"Result: CONTINUE - {message}")
            print(json.dumps(result))
            sys.exit(0)
        else:
            result = {"result": "block", "reason": message}
            logger.info(f"Result: BLOCK - {message}")
            print(json.dumps(result))
            sys.exit(1)

    except Exception as e:
        # On error, allow through but log
        logger.exception(f"Validation error: {e}")
        print(json.dumps({
            "result": "continue",
            "message": f"Validation error (allowing through): {str(e)}"
        }))
        sys.exit(0)


if __name__ == "__main__":
    main()
