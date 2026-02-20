#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "python-dotenv",
# ]
# ///

import argparse
import json
import sys
from pathlib import Path
from datetime import datetime

try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass  # dotenv is optional


def log_session_end(input_data):
    """Log session end event to logs directory."""
    # Ensure logs directory exists
    log_dir = Path("logs")
    log_dir.mkdir(parents=True, exist_ok=True)
    log_file = log_dir / 'session_end.json'

    # Read existing log data or initialize empty list
    if log_file.exists():
        with open(log_file, 'r') as f:
            try:
                log_data = json.load(f)
            except (json.JSONDecodeError, ValueError):
                log_data = []
    else:
        log_data = []

    # Add timestamp to the input data
    input_data['logged_at'] = datetime.now().isoformat()

    # Append the entire input data
    log_data.append(input_data)

    # Write back to file with formatting
    with open(log_file, 'w') as f:
        json.dump(log_data, f, indent=2)


def perform_cleanup():
    """Perform optional cleanup tasks at session end."""
    cleanup_actions = []

    # Example cleanup: Remove temporary files from logs directory
    log_dir = Path("logs")
    if log_dir.exists():
        # Clean up any .tmp files
        for tmp_file in log_dir.glob("*.tmp"):
            try:
                tmp_file.unlink()
                cleanup_actions.append(f"Removed temp file: {tmp_file.name}")
            except Exception:
                pass

    # Example cleanup: Clean up old chat.json if it exists and is stale
    chat_file = log_dir / "chat.json" if log_dir.exists() else None
    if chat_file and chat_file.exists():
        try:
            # Check if file is older than 24 hours
            file_age = datetime.now().timestamp() - chat_file.stat().st_mtime
            if file_age > 86400:  # 24 hours in seconds
                chat_file.unlink()
                cleanup_actions.append("Removed stale chat.json (older than 24 hours)")
        except Exception:
            pass

    return cleanup_actions


def main():
    try:
        # Parse command line arguments
        parser = argparse.ArgumentParser()
        parser.add_argument('--cleanup', action='store_true',
                          help='Perform cleanup tasks at session end')
        args = parser.parse_args()

        # Read JSON input from stdin
        input_data = json.loads(sys.stdin.read())

        # Extract session_id for cleanup logging
        session_id = input_data.get('session_id', 'unknown')

        # Log the session end event
        log_session_end(input_data)

        # Perform cleanup if requested
        if args.cleanup:
            cleanup_actions = perform_cleanup()
            if cleanup_actions:
                # Log cleanup actions
                cleanup_log = {
                    "session_id": session_id,
                    "cleanup_at": datetime.now().isoformat(),
                    "actions": cleanup_actions
                }
                log_dir = Path("logs")
                cleanup_file = log_dir / "cleanup.json"

                # Read existing cleanup log
                if cleanup_file.exists():
                    with open(cleanup_file, 'r') as f:
                        try:
                            cleanup_data = json.load(f)
                        except (json.JSONDecodeError, ValueError):
                            cleanup_data = []
                else:
                    cleanup_data = []

                cleanup_data.append(cleanup_log)

                with open(cleanup_file, 'w') as f:
                    json.dump(cleanup_data, f, indent=2)

        # Success
        sys.exit(0)

    except json.JSONDecodeError:
        # Handle JSON decode errors gracefully
        sys.exit(0)
    except Exception:
        # Handle any other errors gracefully
        sys.exit(0)


if __name__ == '__main__':
    main()
