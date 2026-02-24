#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = [
#     "python-dotenv",
# ]
# ///

import argparse
import json
import os
import sys
import subprocess
from pathlib import Path
from datetime import datetime

try:
    from dotenv import load_dotenv
    load_dotenv()
except ImportError:
    pass  # dotenv is optional


def log_setup(input_data):
    """Log setup event to logs directory."""
    # Ensure logs directory exists
    log_dir = Path("logs")
    log_dir.mkdir(parents=True, exist_ok=True)
    log_file = log_dir / 'setup.json'

    # Read existing log data or initialize empty list
    if log_file.exists():
        with open(log_file, 'r') as f:
            try:
                log_data = json.load(f)
            except (json.JSONDecodeError, ValueError):
                log_data = []
    else:
        log_data = []

    # Append the entire input data with timestamp
    entry = {
        "timestamp": datetime.now().isoformat(),
        **input_data
    }
    log_data.append(entry)

    # Write back to file with formatting
    with open(log_file, 'w') as f:
        json.dump(log_data, f, indent=2)


def persist_env_variable(name, value):
    """Persist an environment variable via CLAUDE_ENV_FILE."""
    env_file = os.environ.get('CLAUDE_ENV_FILE')
    if env_file:
        with open(env_file, 'a') as f:
            f.write(f'export {name}="{value}"\n')
        return True
    return False


def check_dependencies():
    """Check if common project dependencies are available."""
    deps_status = {}

    # Check for Node.js / npm
    try:
        result = subprocess.run(['node', '--version'], capture_output=True, text=True, timeout=5)
        deps_status['node'] = result.stdout.strip() if result.returncode == 0 else None
    except Exception:
        deps_status['node'] = None

    # Check for Python
    try:
        result = subprocess.run(['python3', '--version'], capture_output=True, text=True, timeout=5)
        deps_status['python'] = result.stdout.strip() if result.returncode == 0 else None
    except Exception:
        deps_status['python'] = None

    # Check for uv
    try:
        result = subprocess.run(['uv', '--version'], capture_output=True, text=True, timeout=5)
        deps_status['uv'] = result.stdout.strip() if result.returncode == 0 else None
    except Exception:
        deps_status['uv'] = None

    # Check for git
    try:
        result = subprocess.run(['git', '--version'], capture_output=True, text=True, timeout=5)
        deps_status['git'] = result.stdout.strip() if result.returncode == 0 else None
    except Exception:
        deps_status['git'] = None

    return deps_status


def install_project_dependencies():
    """Attempt to install project dependencies based on detected files."""
    installed = []
    errors = []

    # Check for package.json (Node.js)
    if Path('package.json').exists():
        try:
            # Prefer npm ci for CI environments, npm install otherwise
            cmd = ['npm', 'ci'] if Path('package-lock.json').exists() else ['npm', 'install']
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=300)
            if result.returncode == 0:
                installed.append('npm dependencies')
            else:
                errors.append(f'npm: {result.stderr[:200]}')
        except Exception as e:
            errors.append(f'npm: {str(e)}')

    # Check for requirements.txt (Python)
    if Path('requirements.txt').exists():
        try:
            result = subprocess.run(
                ['pip', 'install', '-r', 'requirements.txt'],
                capture_output=True, text=True, timeout=300
            )
            if result.returncode == 0:
                installed.append('pip dependencies')
            else:
                errors.append(f'pip: {result.stderr[:200]}')
        except Exception as e:
            errors.append(f'pip: {str(e)}')

    # Check for pyproject.toml (Python with uv or pip)
    if Path('pyproject.toml').exists() and not Path('requirements.txt').exists():
        try:
            # Try uv first
            result = subprocess.run(['uv', 'sync'], capture_output=True, text=True, timeout=300)
            if result.returncode == 0:
                installed.append('uv dependencies')
            else:
                # Fallback to pip install .
                result = subprocess.run(
                    ['pip', 'install', '-e', '.'],
                    capture_output=True, text=True, timeout=300
                )
                if result.returncode == 0:
                    installed.append('pip (pyproject.toml)')
                else:
                    errors.append(f'pyproject.toml: {result.stderr[:200]}')
        except Exception as e:
            errors.append(f'pyproject.toml: {str(e)}')

    return installed, errors


def get_project_info(cwd):
    """Gather project information for context."""
    info = []

    # Check for git repository
    try:
        result = subprocess.run(
            ['git', 'rev-parse', '--abbrev-ref', 'HEAD'],
            capture_output=True, text=True, timeout=5, cwd=cwd
        )
        if result.returncode == 0:
            info.append(f"Git branch: {result.stdout.strip()}")
    except Exception:
        pass

    # Check for common project files
    project_files = [
        ('package.json', 'Node.js project'),
        ('pyproject.toml', 'Python project (pyproject.toml)'),
        ('requirements.txt', 'Python project (requirements.txt)'),
        ('Cargo.toml', 'Rust project'),
        ('go.mod', 'Go project'),
        ('Makefile', 'Makefile present'),
    ]

    for filename, description in project_files:
        if Path(cwd, filename).exists():
            info.append(f"Detected: {description}")

    # Check for .claude directory
    if Path(cwd, '.claude').exists():
        info.append("Claude Code configuration directory present")

        # Check for CLAUDE.md or CONTEXT.md
        for context_file in ['CLAUDE.md', 'CONTEXT.md']:
            context_path = Path(cwd, '.claude', context_file)
            if context_path.exists():
                info.append(f"Found {context_file} in .claude/")

    return info


def run_maintenance_tasks(cwd):
    """Run periodic maintenance tasks."""
    tasks_completed = []

    # Check disk usage of logs directory
    logs_dir = Path(cwd, 'logs')
    if logs_dir.exists():
        try:
            total_size = sum(f.stat().st_size for f in logs_dir.rglob('*') if f.is_file())
            size_mb = total_size / (1024 * 1024)
            if size_mb > 10:
                tasks_completed.append(f"Warning: logs directory is {size_mb:.2f}MB")
            else:
                tasks_completed.append(f"Logs directory size: {size_mb:.2f}MB")
        except Exception:
            pass

    # Run git gc if repository is large
    try:
        result = subprocess.run(
            ['git', 'count-objects', '-v'],
            capture_output=True, text=True, timeout=10, cwd=cwd
        )
        if result.returncode == 0:
            tasks_completed.append("Git repository status checked")
    except Exception:
        pass

    return tasks_completed


def main():
    try:
        # Parse command line arguments
        parser = argparse.ArgumentParser()
        parser.add_argument('--install-deps', action='store_true',
                          help='Install project dependencies')
        parser.add_argument('--verbose', action='store_true',
                          help='Print verbose output')
        args = parser.parse_args()

        # Read JSON input from stdin
        input_data = json.loads(sys.stdin.read())

        # Extract fields from Setup hook input
        session_id = input_data.get('session_id', 'unknown')
        _transcript_path = input_data.get('transcript_path', '')  # noqa: F841
        cwd = input_data.get('cwd', os.getcwd())
        _permission_mode = input_data.get('permission_mode', 'default')  # noqa: F841
        _hook_event_name = input_data.get('hook_event_name', 'Setup')  # noqa: F841
        trigger = input_data.get('trigger', 'init')  # "init" or "maintenance"

        # Log the setup event
        log_setup(input_data)

        # Build context information
        context_parts = []
        context_parts.append(f"Setup triggered: {trigger}")
        context_parts.append(f"Session: {session_id[:8]}...")
        context_parts.append(f"Working directory: {cwd}")

        # Gather project information
        project_info = get_project_info(cwd)
        if project_info:
            context_parts.append("\n--- Project Information ---")
            context_parts.extend(project_info)

        # Check dependencies status
        deps_status = check_dependencies()
        available_deps = [f"{k}: {v}" for k, v in deps_status.items() if v]
        if available_deps:
            context_parts.append("\n--- Available Tools ---")
            context_parts.extend(available_deps)

        # Handle trigger-specific actions
        if trigger == 'init':
            context_parts.append("\n--- Repository Initialization ---")

            # Persist project path as environment variable
            persist_env_variable('PROJECT_ROOT', cwd)

            # Install dependencies if requested
            if args.install_deps:
                context_parts.append("Installing dependencies...")
                installed, errors = install_project_dependencies()
                if installed:
                    context_parts.append(f"Installed: {', '.join(installed)}")
                if errors:
                    context_parts.append(f"Errors: {'; '.join(errors)}")

            context_parts.append("Repository initialized with custom configuration")

        elif trigger == 'maintenance':
            context_parts.append("\n--- Maintenance Tasks ---")

            # Run maintenance tasks
            maintenance_results = run_maintenance_tasks(cwd)
            if maintenance_results:
                context_parts.extend(maintenance_results)

            context_parts.append("Maintenance tasks completed")

        # Prepare JSON output with additionalContext
        context = "\n".join(context_parts)

        output = {
            "hookSpecificOutput": {
                "hookEventName": "Setup",
                "additionalContext": context
            }
        }

        print(json.dumps(output))
        sys.exit(0)

    except json.JSONDecodeError:
        # Handle JSON decode errors gracefully
        sys.exit(0)
    except Exception:
        # Handle any other errors gracefully
        sys.exit(0)


if __name__ == '__main__':
    main()
