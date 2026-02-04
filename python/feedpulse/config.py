"""Configuration parsing and validation"""

import sys
from pathlib import Path
from typing import Dict, Any
from urllib.parse import urlparse
import yaml

from .models import Config, Settings, FeedConfig


class ConfigError(Exception):
    """Configuration validation error"""
    pass


def validate_url(url: str) -> bool:
    """Validate that URL is HTTP or HTTPS"""
    try:
        parsed = urlparse(url)
        return parsed.scheme in ('http', 'https') and bool(parsed.netloc)
    except Exception:
        return False


def validate_feed(feed_data: Dict[str, Any], index: int) -> FeedConfig:
    """Validate and parse a single feed configuration"""

    # Required: name
    if 'name' not in feed_data:
        raise ConfigError(f"feed #{index}: missing field 'name'")

    name = feed_data['name']
    if not isinstance(name, str) or not name.strip():
        raise ConfigError(f"feed #{index}: 'name' must be a non-empty string")

    # Required: url
    if 'url' not in feed_data:
        raise ConfigError(f"feed '{name}': missing field 'url'")

    url = feed_data['url']
    if not isinstance(url, str):
        raise ConfigError(f"feed '{name}': 'url' must be a string")

    if not validate_url(url):
        raise ConfigError(f"feed '{name}': invalid URL '{url}'")

    # Required: feed_type
    if 'feed_type' not in feed_data:
        raise ConfigError(f"feed '{name}': missing field 'feed_type'")

    feed_type = feed_data['feed_type']
    valid_types = ('json', 'rss', 'atom', 'hackernews', 'reddit', 'github', 'lobsters')
    if feed_type not in valid_types:
        raise ConfigError(
            f"feed '{name}': feed_type must be one of: {', '.join(valid_types)} (got '{feed_type}')"
        )

    # Optional: refresh_interval_secs
    refresh_interval = feed_data.get('refresh_interval_secs', 300)
    if not isinstance(refresh_interval, int) or refresh_interval <= 0:
        raise ConfigError(f"feed '{name}': refresh_interval_secs must be a positive integer")

    # Optional: headers
    headers = feed_data.get('headers', {})
    if not isinstance(headers, dict):
        raise ConfigError(f"feed '{name}': headers must be a dictionary")

    return FeedConfig(
        name=name,
        url=url,
        feed_type=feed_type,
        refresh_interval_secs=refresh_interval,
        headers=headers
    )


def validate_settings(settings_data: Dict[str, Any]) -> Settings:
    """Validate and parse settings"""

    settings = Settings()

    if 'max_concurrency' in settings_data:
        max_conc = settings_data['max_concurrency']
        if not isinstance(max_conc, int) or max_conc < 1 or max_conc > 50:
            raise ConfigError("settings: max_concurrency must be an integer between 1 and 50")
        settings.max_concurrency = max_conc

    if 'default_timeout_secs' in settings_data:
        timeout = settings_data['default_timeout_secs']
        if not isinstance(timeout, int) or timeout <= 0:
            raise ConfigError("settings: default_timeout_secs must be a positive integer")
        settings.default_timeout_secs = timeout

    if 'retry_max' in settings_data:
        retry = settings_data['retry_max']
        if not isinstance(retry, int) or retry < 0:
            raise ConfigError("settings: retry_max must be a non-negative integer")
        settings.retry_max = retry

    if 'retry_base_delay_ms' in settings_data:
        delay = settings_data['retry_base_delay_ms']
        if not isinstance(delay, int) or delay < 0:
            raise ConfigError("settings: retry_base_delay_ms must be a non-negative integer")
        settings.retry_base_delay_ms = delay

    if 'database_path' in settings_data:
        db_path = settings_data['database_path']
        if not isinstance(db_path, str):
            raise ConfigError("settings: database_path must be a string")
        settings.database_path = db_path

    return settings


def load_config(config_path: str) -> Config:
    """Load and validate configuration from YAML file"""

    path = Path(config_path)

    # Check if file exists
    if not path.exists():
        print(f"Error: config file not found: {config_path}", file=sys.stderr)
        sys.exit(1)

    # Parse YAML
    try:
        with open(path, 'r') as f:
            data = yaml.safe_load(f)
    except yaml.YAMLError as e:
        print(f"Error: invalid config: {e}", file=sys.stderr)
        sys.exit(1)
    except Exception as e:
        print(f"Error: failed to read config: {e}", file=sys.stderr)
        sys.exit(1)

    if not isinstance(data, dict):
        print("Error: invalid config: root must be a dictionary", file=sys.stderr)
        sys.exit(1)

    # Validate and parse
    try:
        # Settings (optional)
        settings_data = data.get('settings', {})
        if not isinstance(settings_data, dict):
            raise ConfigError("settings must be a dictionary")
        settings = validate_settings(settings_data)

        # Feeds (required)
        if 'feeds' not in data:
            raise ConfigError("missing required field 'feeds'")

        feeds_data = data['feeds']
        if not isinstance(feeds_data, list):
            raise ConfigError("feeds must be a list")

        if not feeds_data:
            raise ConfigError("feeds list cannot be empty")

        feeds = []
        for i, feed_data in enumerate(feeds_data, 1):
            if not isinstance(feed_data, dict):
                raise ConfigError(f"feed #{i}: must be a dictionary")
            feeds.append(validate_feed(feed_data, i))

        return Config(settings=settings, feeds=feeds)

    except ConfigError as e:
        print(f"Error: invalid config: {e}", file=sys.stderr)
        sys.exit(1)
