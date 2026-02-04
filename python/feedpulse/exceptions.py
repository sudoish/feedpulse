"""Custom exceptions for FeedPulse.

This module provides a hierarchy of custom exceptions for better error handling
and more informative error messages throughout the application.
"""


class FeedPulseError(Exception):
    """Base exception for all FeedPulse errors."""

    pass


class ConfigError(FeedPulseError):
    """Raised when there's an issue with configuration."""

    pass


class ConfigFileNotFoundError(ConfigError):
    """Raised when the config file cannot be found."""

    def __init__(self, path: str) -> None:
        super().__init__(
            f"Config file not found at '{path}'. "
            f"Create one with: feedpulse init"
        )
        self.path = path


class ConfigValidationError(ConfigError):
    """Raised when config validation fails."""

    pass


class InvalidYAMLError(ConfigError):
    """Raised when config file contains invalid YAML."""

    def __init__(self, path: str, details: str) -> None:
        super().__init__(f"Invalid YAML in '{path}': {details}")
        self.path = path
        self.details = details


class ParserError(FeedPulseError):
    """Raised when parsing feed data fails."""

    pass


class MalformedJSONError(ParserError):
    """Raised when JSON parsing fails."""

    def __init__(self, details: str) -> None:
        super().__init__(f"Malformed JSON: {details}")
        self.details = details


class UnexpectedStructureError(ParserError):
    """Raised when feed data has unexpected structure."""

    def __init__(self, feed_type: str, details: str) -> None:
        super().__init__(
            f"Unexpected structure for {feed_type} feed: {details}"
        )
        self.feed_type = feed_type
        self.details = details


class StorageError(FeedPulseError):
    """Raised when database operations fail."""

    pass


class DatabaseLockedError(StorageError):
    """Raised when database is locked."""

    def __init__(self, path: str, attempts: int) -> None:
        super().__init__(
            f"Database locked at '{path}' after {attempts} attempts"
        )
        self.path = path
        self.attempts = attempts


class DatabaseCorruptedError(StorageError):
    """Raised when database file is corrupted."""

    def __init__(self, path: str, details: str) -> None:
        super().__init__(f"Database corrupted at '{path}': {details}")
        self.path = path
        self.details = details


class FetchError(FeedPulseError):
    """Raised when fetching feed data fails."""

    pass


class NetworkError(FetchError):
    """Raised when network operations fail."""

    pass


class TimeoutError(FetchError):
    """Raised when a fetch operation times out."""

    def __init__(self, url: str, timeout: int) -> None:
        super().__init__(f"Timeout fetching '{url}' after {timeout}s")
        self.url = url
        self.timeout = timeout


class ValidationError(FeedPulseError):
    """Raised when data validation fails."""

    pass


class URLValidationError(ValidationError):
    """Raised when URL validation fails."""

    def __init__(self, url: str, reason: str) -> None:
        super().__init__(f"Invalid URL '{url}': {reason}")
        self.url = url
        self.reason = reason
