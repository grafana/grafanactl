#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "python-dotenv",
# ]
# ///

import json
import sys
from datetime import datetime
from pathlib import Path

try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass  # dotenv is optional


def main():
    try:
        # Read JSON input from stdin
        input_data = json.load(sys.stdin)

        # Add timestamp to the log entry
        input_data['logged_at'] = datetime.now().isoformat()

        # Extract key fields for enhanced logging
        tool_name = input_data.get('tool_name', 'unknown')
        tool_use_id = input_data.get('tool_use_id', 'unknown')
        error = input_data.get('error', {})

        # Create a structured log entry with error details
        log_entry = {
            'timestamp': input_data['logged_at'],
            'session_id': input_data.get('session_id', ''),
            'hook_event_name': input_data.get('hook_event_name', 'PostToolUseFailure'),
            'tool_name': tool_name,
            'tool_use_id': tool_use_id,
            'tool_input': input_data.get('tool_input', {}),
            'error': error,
            'cwd': input_data.get('cwd', ''),
            'permission_mode': input_data.get('permission_mode', ''),
            'transcript_path': input_data.get('transcript_path', ''),
            'raw_input': input_data
        }

        # Ensure log directory exists
        log_dir = Path.cwd() / 'logs'
        log_dir.mkdir(parents=True, exist_ok=True)
        log_path = log_dir / 'post_tool_use_failure.json'

        # Read existing log data or initialize empty list
        if log_path.exists():
            with open(log_path, 'r') as f:
                try:
                    log_data = json.load(f)
                except (json.JSONDecodeError, ValueError):
                    log_data = []
        else:
            log_data = []

        # Append new log entry
        log_data.append(log_entry)

        # Write back to file with formatting
        with open(log_path, 'w') as f:
            json.dump(log_data, f, indent=2)

        sys.exit(0)

    except json.JSONDecodeError:
        # Handle JSON decode errors gracefully
        sys.exit(0)
    except Exception:
        # Exit cleanly on any other error
        sys.exit(0)


if __name__ == '__main__':
    main()
