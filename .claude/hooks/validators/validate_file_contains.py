#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///

"""
Validates that a newly created file contains required content strings.

Hook Type: Stop

Checks:
1. Find the most recently created file in the specified directory (like validate_new_file)
2. Verify the file contains all required strings (case-sensitive)

Exit codes:
- 0: Validation passed (file exists and contains all required strings)
- 1: Validation failed (file missing or missing required content)

Usage:
  uv run validate_file_contains.py -d specs -e .md --contains "## Task Description" --contains "## Objective"
  uv run validate_file_contains.py --directory output --extension .json --contains '"status":'

Frontmatter example:
  hooks:
    Stop:
      - hooks:
          - type: command
            command: "uv run $CLAUDE_PROJECT_DIR/.claude/hooks/validators/validate_file_contains.py -d specs -e .md --contains '## Task Description' --contains '## Objective'"
"""

import argparse
import json
import logging
import subprocess
import sys
import time
from pathlib import Path

# Logging setup - log file next to this script (SAME NAME)
SCRIPT_DIR = Path(__file__).parent
LOG_FILE = SCRIPT_DIR / "validate_file_contains.log"

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s | %(levelname)s | %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
    handlers=[logging.FileHandler(LOG_FILE, mode='a')]
)
logger = logging.getLogger(__name__)

# Constants
DEFAULT_DIRECTORY = "specs"
DEFAULT_EXTENSION = ".md"
DEFAULT_MAX_AGE_MINUTES = 5

NO_FILE_ERROR = (
    "VALIDATION FAILED: No new file found matching {pattern}.\n\n"
    "ACTION REQUIRED: Use the Write tool to create a new file in the {directory}/ directory. "
    "The file must be created before this validation can pass. "
    "Do not stop until the file has been created."
)

MISSING_CONTENT_ERROR = (
    "VALIDATION FAILED: File '{file}' is missing {count} required section(s).\n\n"
    "MISSING SECTIONS:\n{missing_list}\n\n"
    "ACTION REQUIRED: Use the Edit tool to add the missing sections to '{file}'. "
    "Each section must appear exactly as shown above (case-sensitive). "
    "Do not stop until all required sections are present in the file."
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
            status = line[:2]
            filepath = line[3:].strip()

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


def find_newest_file(directory: str, extension: str, max_age_minutes: int) -> str | None:
    """
    Find the most recently created/modified file in directory.

    Returns:
        Path to the newest file, or None if no recent files found.
    """
    # Check git for untracked/new files
    git_new = get_git_untracked_files(directory, extension)

    # Check for recently modified files
    recent_files = get_recent_files(directory, extension, max_age_minutes)

    # Combine and deduplicate
    all_files = list(set(git_new + recent_files))

    if not all_files:
        return None

    # Find the newest file by modification time
    newest = None
    newest_mtime = 0

    for filepath in all_files:
        try:
            path = Path(filepath)
            if path.exists():
                mtime = path.stat().st_mtime
                if mtime > newest_mtime:
                    newest_mtime = mtime
                    newest = str(path)
        except OSError:
            continue

    return newest


def check_file_contains(filepath: str, required_strings: list[str]) -> tuple[bool, list[str], list[str]]:
    """
    Check if file contains all required strings (case-sensitive).

    Args:
        filepath: Path to the file to check
        required_strings: List of strings that must be present

    Returns:
        tuple: (all_found: bool, found: list[str], missing: list[str])
    """
    try:
        content = Path(filepath).read_text(encoding='utf-8')
    except (OSError, UnicodeDecodeError) as e:
        logger.error(f"Failed to read file {filepath}: {e}")
        return False, [], required_strings

    found = []
    missing = []

    for req in required_strings:
        if req in content:
            found.append(req)
        else:
            missing.append(req)

    return len(missing) == 0, found, missing


def validate_file_contains(
    directory: str,
    extension: str,
    max_age_minutes: int,
    required_strings: list[str]
) -> tuple[bool, str]:
    """
    Validate that a new file was created AND contains required content.

    Args:
        directory: Directory to check for new files
        extension: File extension to match
        max_age_minutes: Maximum age in minutes for "recent" files
        required_strings: List of strings that must be in the file

    Returns:
        tuple: (success: bool, message: str)
    """
    pattern = f"{directory}/*{extension}"
    logger.info(f"Validating: directory={directory}, extension={extension}, max_age={max_age_minutes}min")
    logger.info(f"Required strings: {required_strings}")

    # Step 1: Find the newest file
    newest_file = find_newest_file(directory, extension, max_age_minutes)

    if not newest_file:
        msg = NO_FILE_ERROR.format(pattern=pattern, directory=directory)
        logger.warning(f"FAIL: {msg}")
        return False, msg

    logger.info(f"Found newest file: {newest_file}")

    # Step 2: Check if file contains all required strings
    if not required_strings:
        msg = f"File found: {newest_file} (no content checks specified)"
        logger.info(f"PASS: {msg}")
        return True, msg

    all_found, found, missing = check_file_contains(newest_file, required_strings)

    logger.info(f"Content check - Found: {len(found)}/{len(required_strings)}")
    if found:
        logger.info(f"  Found: {found}")
    if missing:
        logger.warning(f"  Missing: {missing}")

    if all_found:
        msg = f"File '{newest_file}' contains all {len(required_strings)} required sections"
        logger.info(f"PASS: {msg}")
        return True, msg
    else:
        missing_list = "\n".join(f"  - {m}" for m in missing)
        msg = MISSING_CONTENT_ERROR.format(
            file=newest_file,
            count=len(missing),
            missing_list=missing_list
        )
        logger.warning(f"FAIL: Missing {len(missing)} required sections")
        return False, msg


def parse_args() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Validate that a new file contains required content"
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
    parser.add_argument(
        '--contains',
        action='append',
        dest='required_strings',
        default=[],
        metavar='STRING',
        help='Required string that must be in the file (can be used multiple times)'
    )
    return parser.parse_args()


def main():
    """Main entry point for the validator."""
    logger.info("=" * 60)
    logger.info("Validator started: validate_file_contains")

    try:
        # Parse CLI arguments
        args = parse_args()
        logger.info(f"Args: directory={args.directory}, extension={args.extension}, max_age={args.max_age}")
        logger.info(f"Required strings count: {len(args.required_strings)}")

        # Read hook input from stdin (if provided)
        try:
            input_data = json.load(sys.stdin)
            logger.info(f"Stdin input received: {len(json.dumps(input_data))} bytes")
        except (json.JSONDecodeError, EOFError):
            input_data = {}
            logger.info("No stdin input or invalid JSON")

        # Run validation
        success, message = validate_file_contains(
            directory=args.directory,
            extension=args.extension,
            max_age_minutes=args.max_age,
            required_strings=args.required_strings
        )

        if success:
            result = {"result": "continue", "message": message}
            logger.info(f"Result: CONTINUE - {message}")
            print(json.dumps(result))
            sys.exit(0)
        else:
            result = {"result": "block", "reason": message}
            logger.info(f"Result: BLOCK")
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
