#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.8"
# dependencies = [
#     "anthropic",
#     "python-dotenv",
# ]
# ///

"""
Task Summarizer LLM Utility

Generates natural language summaries of subagent task completions.
Designed for TTS announcements to provide personalized feedback.
"""

import os
import sys
from typing import Optional
from datetime import datetime
from dotenv import load_dotenv


def debug_log(message: str) -> None:
    """Write debug message to logs/subagent_debug.log"""
    try:
        log_dir = os.path.join(os.getcwd(), "logs")
        os.makedirs(log_dir, exist_ok=True)
        debug_path = os.path.join(log_dir, "subagent_debug.log")
        timestamp = datetime.now().isoformat()
        with open(debug_path, 'a') as f:
            f.write(f"[{timestamp}] [SUMMARIZER] {message}\n")
    except Exception:
        pass


def summarize_subagent_task(task_description: str, agent_name: Optional[str] = None) -> str:
    """
    Generate a natural language summary of a completed subagent task.

    Args:
        task_description: Description of the task that was completed
        agent_name: Optional name of the agent that completed the task

    Returns:
        str: A conversational summary suitable for TTS announcement
    """
    load_dotenv()
    debug_log(f"summarize_subagent_task called with: {task_description[:50]}...")

    api_key = os.getenv("ANTHROPIC_API_KEY")
    if not api_key:
        debug_log("ERROR: ANTHROPIC_API_KEY not found!")
        return "Subagent task completed"

    debug_log(f"API key found (length: {len(api_key)})")

    # Build agent context for the prompt
    if agent_name:
        agent_context = f"The agent named '{agent_name}' completed this task."
        agent_instruction = f"You can reference the agent by name ('{agent_name}') naturally."
    else:
        agent_context = "A subagent completed this task."
        agent_instruction = "Refer to it as 'your agent' or similar."

    prompt = f"""Generate a brief, conversational summary of a completed task for audio announcement.

Task completed: {task_description}

Context: {agent_context}

Requirements:
- Address the user as "Dan" directly (but not always at the start)
- Keep it under 20 words
- Focus on the outcome and value delivered
- Be conversational and personalized
- {agent_instruction}
- Do NOT include quotes, formatting, or explanations
- Return ONLY the summary text

Example styles:
- "Dan, authentication is ready with secure JWT token support."
- "Your file watcher is now monitoring for changes."
- "Builder finished setting up the TTS queue with file locks."
- "Dan, the new API endpoints are live and tested."

Generate ONE summary:"""

    try:
        import anthropic
        debug_log("Anthropic module imported successfully")

        client = anthropic.Anthropic(api_key=api_key)
        debug_log("Anthropic client created")

        debug_log("Calling Haiku API...")
        message = client.messages.create(
            model="claude-haiku-4-5-20251001",  # Haiku 4.5 - fast and cost-effective
            max_tokens=100,
            temperature=0.7,
            messages=[{"role": "user", "content": prompt}],
        )
        debug_log("API call completed")

        response = message.content[0].text.strip()
        debug_log(f"Raw response: {response}")

        # Clean up response - remove quotes and extra formatting
        if response:
            response = response.strip().strip('"').strip("'").strip()
            # Take first line if multiple lines
            response = response.split("\n")[0].strip()
            debug_log(f"Cleaned response: {response}")
            return response

        debug_log("Response was empty, returning fallback")
        return "Subagent task completed"

    except Exception as e:
        debug_log(f"EXCEPTION: {type(e).__name__}: {str(e)}")
        return "Subagent task completed"


def main() -> None:
    """Command line interface for testing."""
    import argparse

    parser = argparse.ArgumentParser(
        description="Generate natural language summaries of subagent task completions"
    )
    parser.add_argument(
        "task_description",
        nargs="?",
        help="Description of the completed task"
    )
    parser.add_argument(
        "--agent-name",
        "-a",
        type=str,
        default=None,
        help="Name of the agent that completed the task"
    )

    args = parser.parse_args()

    if not args.task_description:
        parser.print_help()
        print("\nExamples:")
        print('  uv run task_summarizer.py "Built authentication system"')
        print('  uv run task_summarizer.py "Built authentication system" --agent-name "builder"')
        sys.exit(1)

    summary = summarize_subagent_task(args.task_description, args.agent_name)
    print(summary)


if __name__ == "__main__":
    main()
