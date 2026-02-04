# Contributing to FeedPulse

Thank you for your interest in contributing to FeedPulse! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Code Style](#code-style)
- [Testing](#testing)
- [Pull Request Process](#pull-request-process)
- [Issue Guidelines](#issue-guidelines)

## Getting Started

### Prerequisites

- Python 3.10 or higher
- Git
- Basic understanding of async/await in Python
- Familiarity with SQLite

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR_USERNAME/feedpulse.git
cd feedpulse/python
```

3. Add the upstream repository:

```bash
git remote add upstream https://github.com/original/feedpulse.git
```

## Development Setup

### 1. Create Virtual Environment

```bash
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate
```

### 2. Install Dependencies

```bash
# Install runtime dependencies
pip install -r requirements.txt

# Install development dependencies
pip install -r requirements-dev.txt

# Install package in editable mode
pip install -e .
```

### 3. Verify Setup

```bash
# Run tests
pytest

# Check types
mypy feedpulse/

# Run linter
ruff check feedpulse/
```

## Code Style

### Python Style Guide

We follow [PEP 8](https://peps.python.org/pep-0008/) with some modifications:

- **Line length**: 100 characters (not 79)
- **Quotes**: Use double quotes for strings (not single)
- **Type hints**: Required for all public functions
- **Docstrings**: Required for all public functions (Google style)

### Formatting

Use `black` for automatic code formatting:

```bash
# Format all files
black feedpulse/ tests/

# Check without modifying
black --check feedpulse/
```

### Type Hints

All public functions must have complete type hints:

```python
# Good
def parse_feed(response_body: str, feed_type: str, source: str) -> list[FeedItem]:
    """Parse feed response based on feed_type."""
    ...

# Bad
def parse_feed(response_body, feed_type, source):
    """Parse feed response based on feed_type."""
    ...
```

Use modern type syntax (Python 3.10+):
- `list[str]` instead of `List[str]`
- `dict[str, int]` instead of `Dict[str, int]`
- `tuple[int, str]` instead of `Tuple[int, str]`

### Docstrings

Use Google-style docstrings for all public functions:

```python
def fetch_feed(url: str, timeout: int = 10) -> str:
    """Fetch feed content from URL.
    
    This function makes an HTTP GET request to the specified URL
    and returns the response body as a string.
    
    Args:
        url: Feed URL to fetch from (must be HTTP/HTTPS)
        timeout: Request timeout in seconds (default: 10)
        
    Returns:
        Response body as string
        
    Raises:
        NetworkError: If the request fails or times out
        URLValidationError: If the URL is invalid
        
    Examples:
        >>> content = fetch_feed("https://example.com/feed.json")
        >>> len(content) > 0
        True
    """
    ...
```

### Imports

Organize imports in this order:

1. Standard library
2. Third-party packages
3. Local modules

Use absolute imports for local modules:

```python
# Good
from feedpulse.models import FeedItem
from feedpulse.utils import parse_timestamp

# Bad
from .models import FeedItem
from .utils import parse_timestamp
```

## Testing

### Writing Tests

- Place tests in `tests/` directory
- Name test files `test_*.py`
- Name test functions `test_*`
- Use descriptive test names

Example:

```python
def test_parse_hackernews_valid_response():
    """Test parsing a valid HackerNews API response."""
    data = {"hits": [{"objectID": "123", "title": "Test", "url": "https://example.com"}]}
    items = parse_hackernews(data, "test-source")
    
    assert len(items) == 1
    assert items[0].title == "Test"
    assert items[0].url == "https://example.com"
```

### Test Categories

1. **Unit Tests**: Test individual functions in isolation
2. **Integration Tests**: Test complete workflows
3. **Edge Cases**: Test boundary conditions and unusual inputs
4. **Error Scenarios**: Test error handling and resilience

### Running Tests

```bash
# Run all tests
pytest

# Run specific test file
pytest tests/test_parser.py

# Run specific test
pytest tests/test_parser.py::test_parse_hackernews_valid_response

# Run with coverage
pytest --cov=feedpulse --cov-report=html

# Run with verbose output
pytest -v

# Run only failed tests
pytest --lf
```

### Test Coverage

- Aim for **95%+ coverage** for new code
- Don't sacrifice test quality for coverage numbers
- Focus on testing behavior, not implementation

Check coverage:

```bash
pytest --cov=feedpulse --cov-report=term-missing
```

### Using Fixtures

Use pytest fixtures for common test setup:

```python
import pytest
from feedpulse.storage import FeedDatabase

@pytest.fixture
def temp_db():
    """Create a temporary database for testing."""
    db = FeedDatabase(":memory:")
    yield db
    # Cleanup happens automatically

def test_insert_items(temp_db):
    """Test inserting items into database."""
    item = FeedItem.create(title="Test", url="https://example.com", source="test")
    count = temp_db.insert_items([item])
    assert count == 1
```

## Pull Request Process

### Before Submitting

1. **Create a branch** for your changes:
   ```bash
   git checkout -b feature/my-new-feature
   ```

2. **Make your changes** following the code style guidelines

3. **Add tests** for new functionality

4. **Run the test suite**:
   ```bash
   pytest
   mypy feedpulse/
   black --check feedpulse/
   ruff check feedpulse/
   ```

5. **Update documentation** if needed (README, docstrings)

6. **Commit your changes**:
   ```bash
   git add .
   git commit -m "Add feature: short description"
   ```

### Commit Message Format

Use conventional commit format:

```
<type>: <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, no logic changes)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

Example:

```
feat: add support for Lobsters feed parsing

- Implement parse_lobsters() function
- Add tests for various Lobsters response formats
- Update documentation with Lobsters example

Closes #42
```

### Submitting Pull Request

1. **Push your branch**:
   ```bash
   git push origin feature/my-new-feature
   ```

2. **Open a pull request** on GitHub

3. **Fill out the PR template** with:
   - Description of changes
   - Motivation and context
   - How to test
   - Related issues

4. **Wait for review** and address feedback

5. **Squash commits** if requested before merging

### Review Process

- Maintainers will review your PR within 1-2 days
- Address review comments by pushing new commits
- CI must pass before merging
- At least one maintainer approval required

## Issue Guidelines

### Reporting Bugs

Include:
- Python version
- FeedPulse version
- Operating system
- Minimal reproduction steps
- Expected vs actual behavior
- Error messages and stack traces

### Suggesting Features

Include:
- Use case and motivation
- Proposed API or behavior
- Examples of usage
- Any alternatives considered

### Issue Labels

- `bug`: Something isn't working
- `enhancement`: New feature or improvement
- `documentation`: Documentation improvements
- `good first issue`: Good for newcomers
- `help wanted`: Extra attention needed

## Development Tools

### Recommended VS Code Extensions

- Python (Microsoft)
- Pylance (Microsoft)
- Python Test Explorer
- GitLens
- Even Better TOML

### VS Code Settings

```json
{
  "python.linting.enabled": true,
  "python.linting.mypyEnabled": true,
  "python.formatting.provider": "black",
  "editor.formatOnSave": true,
  "python.testing.pytestEnabled": true
}
```

### Pre-commit Hooks

Install pre-commit hooks to catch issues before committing:

```bash
pip install pre-commit
pre-commit install
```

Create `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/psf/black
    rev: 23.3.0
    hooks:
      - id: black

  - repo: https://github.com/charliermarsh/ruff-pre-commit
    rev: v0.0.270
    hooks:
      - id: ruff

  - repo: https://github.com/pre-commit/mirrors-mypy
    rev: v1.3.0
    hooks:
      - id: mypy
```

## Questions?

- Open a [Discussion](https://github.com/yourusername/feedpulse/discussions)
- Ask in an [Issue](https://github.com/yourusername/feedpulse/issues)
- Check the [Wiki](https://github.com/yourusername/feedpulse/wiki)

## Code of Conduct

Be respectful, inclusive, and constructive. See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for details.

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
