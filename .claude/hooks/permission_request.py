#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "python-dotenv",
# ]
# ///

"""
PermissionRequest Hook
======================
Triggered when the user is shown a permission dialog.

This hook can:
- Log all permission requests for auditing
- Auto-allow specific patterns (e.g., read-only operations)
- Deny permission requests based on security policies
- Modify tool inputs before allowing

Input JSON includes:
- session_id, transcript_path, cwd, permission_mode
- hook_event_name: "PermissionRequest"
- tool_name, tool_input, tool_use_id

Output JSON for decision control:
{
  "hookSpecificOutput": {
    "hookEventName": "PermissionRequest",
    "decision": {
      "behavior": "allow" | "deny",
      "updatedInput": {...},  // optional for allow
      "message": "...",       // optional for deny
      "interrupt": false      // optional for deny
    }
  }
}
"""

import argparse
import json
import re
import sys
from pathlib import Path

try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass  # dotenv is optional


# Read-only patterns that can be auto-allowed
READ_ONLY_PATTERNS = {
    "Read": lambda tool_input: True,  # All Read operations
    "Glob": lambda tool_input: True,  # All Glob operations
    "Grep": lambda tool_input: True,  # All Grep operations
    "Bash": lambda tool_input: is_safe_bash_command(tool_input.get("command", "")),
}

# Safe bash commands that can be auto-allowed
SAFE_BASH_COMMANDS = [
    r"^ls\b",
    r"^pwd\b",
    r"^echo\b",
    r"^cat\b(?!.*>)",  # cat without redirection
    r"^head\b",
    r"^tail\b",
    r"^wc\b",
    r"^which\b",
    r"^whereis\b",
    r"^type\b",
    r"^file\b",
    r"^stat\b",
    r"^git\s+(status|log|diff|show|branch|tag)\b",
    r"^git\s+remote\s+-v\b",
    r"^npm\s+(list|ls|outdated|view)\b",
    r"^pip\s+(list|show|freeze)\b",
    r"^uv\s+(pip\s+list|tree)\b",
    r"^python\s+--version\b",
    r"^node\s+--version\b",
    r"^npm\s+--version\b",
]


def is_safe_bash_command(command: str) -> bool:
    """
    Check if a bash command is safe (read-only).

    Args:
        command: The bash command to check

    Returns:
        True if the command is considered safe/read-only
    """
    if not command:
        return False

    # Normalize command
    normalized = command.strip()

    # Check against safe patterns
    for pattern in SAFE_BASH_COMMANDS:
        if re.search(pattern, normalized):
            return True

    return False


def should_auto_allow(tool_name: str, tool_input: dict) -> bool:
    """
    Determine if a tool call should be auto-allowed based on read-only patterns.

    Args:
        tool_name: Name of the tool being called
        tool_input: The tool's input parameters

    Returns:
        True if the tool call should be auto-allowed
    """
    if tool_name in READ_ONLY_PATTERNS:
        check_func = READ_ONLY_PATTERNS[tool_name]
        return check_func(tool_input)

    return False


def get_auto_allow_reason(tool_name: str, tool_input: dict) -> str:
    """
    Get a reason string for why a tool was auto-allowed.

    Args:
        tool_name: Name of the tool being called
        tool_input: The tool's input parameters

    Returns:
        Human-readable reason for auto-allowing
    """
    if tool_name == "Read":
        file_path = tool_input.get("file_path", "unknown")
        return f"Read operation auto-allowed: {file_path}"
    elif tool_name == "Glob":
        pattern = tool_input.get("pattern", "unknown")
        return f"Glob pattern search auto-allowed: {pattern}"
    elif tool_name == "Grep":
        pattern = tool_input.get("pattern", "unknown")
        return f"Grep search auto-allowed: {pattern}"
    elif tool_name == "Bash":
        command = tool_input.get("command", "unknown")
        return f"Safe bash command auto-allowed: {command[:50]}..."

    return f"{tool_name} auto-allowed (read-only operation)"


def create_allow_response(updated_input: dict | None = None, reason: str | None = None) -> dict:
    """
    Create a JSON response to allow a permission request.

    Args:
        updated_input: Optional modified tool input
        reason: Optional reason for allowing (for logging)

    Returns:
        JSON-serializable response dict
    """
    decision = {"behavior": "allow"}

    if updated_input is not None:
        decision["updatedInput"] = updated_input

    return {
        "hookSpecificOutput": {
            "hookEventName": "PermissionRequest",
            "decision": decision
        }
    }


def create_deny_response(message: str, interrupt: bool = False) -> dict:
    """
    Create a JSON response to deny a permission request.

    Args:
        message: Message explaining why permission was denied
        interrupt: Whether to stop Claude after denying

    Returns:
        JSON-serializable response dict
    """
    return {
        "hookSpecificOutput": {
            "hookEventName": "PermissionRequest",
            "decision": {
                "behavior": "deny",
                "message": message,
                "interrupt": interrupt
            }
        }
    }


def log_permission_request(input_data: dict, log_dir: Path):
    """
    Log the permission request to a JSON file.

    Args:
        input_data: The input data from the hook
        log_dir: Path to the logs directory
    """
    log_path = log_dir / "permission_request.json"

    # Read existing log data or initialize empty list
    if log_path.exists():
        with open(log_path, "r") as f:
            try:
                log_data = json.load(f)
            except (json.JSONDecodeError, ValueError):
                log_data = []
    else:
        log_data = []

    # Append new data
    log_data.append(input_data)

    # Write back to file with formatting
    with open(log_path, "w") as f:
        json.dump(log_data, f, indent=2)


def main():
    try:
        # Parse command line arguments
        parser = argparse.ArgumentParser(
            description="PermissionRequest hook for Claude Code"
        )
        parser.add_argument(
            "--auto-allow",
            action="store_true",
            help="Auto-allow read-only operations (Read, Glob, Grep, safe Bash commands)"
        )
        parser.add_argument(
            "--log-only",
            action="store_true",
            help="Only log permission requests, do not make decisions"
        )
        args = parser.parse_args()

        # Read JSON input from stdin
        input_data = json.load(sys.stdin)

        # Extract fields
        tool_name = input_data.get("tool_name", "")
        tool_input = input_data.get("tool_input", {})
        hook_event_name = input_data.get("hook_event_name", "")

        # Verify this is a PermissionRequest event
        if hook_event_name != "PermissionRequest":
            # Not a PermissionRequest event, exit gracefully
            sys.exit(0)

        # Ensure log directory exists
        log_dir = Path.cwd() / "logs"
        log_dir.mkdir(parents=True, exist_ok=True)

        # Log the permission request
        log_permission_request(input_data, log_dir)

        # If log-only mode, exit without making a decision
        if args.log_only:
            sys.exit(0)

        # Handle auto-allow for read-only operations
        if args.auto_allow and should_auto_allow(tool_name, tool_input):
            reason = get_auto_allow_reason(tool_name, tool_input)
            response = create_allow_response(reason=reason)
            print(json.dumps(response))
            sys.exit(0)

        # Default: exit without making a decision (let user decide)
        sys.exit(0)

    except json.JSONDecodeError:
        # Gracefully handle JSON decode errors
        sys.exit(0)
    except Exception:
        # Handle any other errors gracefully
        sys.exit(0)


if __name__ == "__main__":
    main()
