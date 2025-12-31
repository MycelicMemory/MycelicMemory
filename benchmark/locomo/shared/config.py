"""
Centralized configuration for LoCoMo benchmark.
Loads settings from environment variables with .env file support.
"""

import os
import sys
from pathlib import Path

# Load .env file if present
try:
    from dotenv import load_dotenv
    # Look for .env in the benchmark/locomo directory (parent of shared/)
    env_path = Path(__file__).parent.parent / ".env"
    if env_path.exists():
        load_dotenv(env_path)
except ImportError:
    pass  # python-dotenv not installed, rely on environment only


def get_required_env(key: str) -> str:
    """Get required environment variable or exit with error."""
    value = os.getenv(key)
    if not value:
        print(f"❌ Missing required environment variable: {key}")
        print(f"   Set it via: export {key}=your_value")
        print(f"   Or create a .env file (see .env.example)")
        sys.exit(1)

    # Strip whitespace and newlines (common copy-paste issue with secrets)
    value = value.strip()
    return value


# API Keys
DEEPSEEK_API_KEY = get_required_env("DEEPSEEK_API_KEY")

# Server URLs
ULTRATHINK_URL = os.getenv("ULTRATHINK_URL", "http://localhost:3099/api/v1")

# DeepSeek API settings
DEEPSEEK_BASE_URL = "https://api.deepseek.com"
DEEPSEEK_MODEL = "deepseek-chat"
