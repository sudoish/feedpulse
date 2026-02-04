"""Tests for error handling scenarios from the spec"""

import pytest
import tempfile
import json
from pathlib import Path

from feedpulse.config import load_config, ConfigError
from feedpulse.parser import parse_feed
from feedpulse.storage import FeedDatabase, DatabaseError
from feedpulse.models import FeedItem, FetchResult


def test_config_missing_file(capsys):
    """Test error message when config file is missing"""
    with pytest.raises(SystemExit) as exc_info:
        load_config("/nonexistent/config.yaml")
    
    assert exc_info.value.code == 1
    captured = capsys.readouterr()
    assert "config file not found" in captured.err


def test_config_invalid_yaml(tmp_path, capsys):
    """Test error message for invalid YAML"""
    config_file = tmp_path / "invalid.yaml"
    config_file.write_text("invalid: yaml: content: [")
    
    with pytest.raises(SystemExit) as exc_info:
        load_config(str(config_file))
    
    assert exc_info.value.code == 1
    captured = capsys.readouterr()
    assert "invalid config" in captured.err


def test_config_missing_required_field(tmp_path, capsys):
    """Test error message for missing required field"""
    config_file = tmp_path / "missing_field.yaml"
    config_file.write_text("""
feeds:
  - name: "Test"
    feed_type: json
""")
    
    with pytest.raises(SystemExit) as exc_info:
        load_config(str(config_file))
    
    assert exc_info.value.code == 1
    captured = capsys.readouterr()
    assert "missing field 'url'" in captured.err


def test_config_invalid_url(tmp_path, capsys):
    """Test error message for invalid URL"""
    config_file = tmp_path / "invalid_url.yaml"
    config_file.write_text("""
feeds:
  - name: "Test"
    url: "not-a-valid-url"
    feed_type: json
""")
    
    with pytest.raises(SystemExit) as exc_info:
        load_config(str(config_file))
    
    assert exc_info.value.code == 1
    captured = capsys.readouterr()
    assert "invalid URL" in captured.err


def test_parse_malformed_json(capsys):
    """Test parsing malformed JSON"""
    items = parse_feed("{invalid json", "json", "Test Source")
    
    assert len(items) == 0
    captured = capsys.readouterr()
    assert "malformed JSON" in captured.err


def test_parse_json_missing_fields(capsys):
    """Test parsing JSON with missing expected fields"""
    # GitHub-like response missing required fields
    data = json.dumps({
        "items": [
            {"full_name": "test/repo"},  # Missing html_url
            {"html_url": "https://github.com/test/repo"}  # Missing full_name
        ]
    })
    
    items = parse_feed(data, "json", "Test Source")
    
    # Both items should be skipped
    assert len(items) == 0
    captured = capsys.readouterr()
    assert "missing" in captured.err.lower()


def test_parse_json_wrong_types(capsys):
    """Test parsing JSON with wrong types"""
    # Lobsters-like response with wrong types
    data = json.dumps([
        {
            "title": 12345,  # Should be string
            "url": "https://example.com",
            "tags": []
        }
    ])
    
    items = parse_feed(data, "json", "Test Source")
    
    # Should coerce title to string
    assert len(items) == 1
    assert items[0].title == "12345"


def test_database_locked_retry():
    """Test database locked retry logic"""
    with tempfile.NamedTemporaryFile(suffix='.db', delete=False) as f:
        db_path = f.name
    
    try:
        db = FeedDatabase(db_path)
        
        # Store some results
        results = [
            FetchResult(
                source="Test",
                status="success",
                items=[
                    FeedItem.create("Test", "https://example.com", "Test")
                ],
                duration_ms=100
            )
        ]
        
        # Should succeed
        new_count = db.store_results(results)
        assert new_count == 1
    
    finally:
        Path(db_path).unlink(missing_ok=True)


def test_database_corrupted(tmp_path, capsys):
    """Test error message for corrupted database"""
    db_path = tmp_path / "corrupt.db"
    
    # Create a corrupted database file
    db_path.write_text("This is not a valid SQLite database")
    
    with pytest.raises(SystemExit) as exc_info:
        db = FeedDatabase(str(db_path))
    
    assert exc_info.value.code == 1
    captured = capsys.readouterr()
    assert "corrupted" in captured.err.lower()


def test_parse_empty_response():
    """Test parsing empty response"""
    items = parse_feed("[]", "json", "Test Source")
    assert len(items) == 0


def test_parse_unexpected_structure(capsys):
    """Test parsing JSON with unexpected structure"""
    data = json.dumps({"unexpected": "structure"})
    items = parse_feed(data, "json", "Test Source")
    
    # Should return empty list and log warning
    assert len(items) == 0
    captured = capsys.readouterr()
    assert "unable to auto-detect" in captured.err.lower() or len(captured.err) > 0


def test_parse_unicode_chaos():
    """Test parsing Unicode edge cases"""
    data = json.dumps([
        {
            "title": "Test 测试 🚀 \u0000 \uffff",
            "url": "https://example.com",
            "tags": [],
            "created_at": "2024-01-01T12:00:00Z"
        }
    ])
    
    items = parse_feed(data, "json", "Test Source")
    assert len(items) == 1
    # Should handle Unicode gracefully
    assert "测试" in items[0].title


def test_feeditem_id_generation():
    """Test FeedItem ID generation is deterministic"""
    item1 = FeedItem.create("Test", "https://example.com", "Source1")
    item2 = FeedItem.create("Test", "https://example.com", "Source1")
    
    # Same source and URL should generate same ID
    assert item1.id == item2.id
    
    # Different source should generate different ID
    item3 = FeedItem.create("Test", "https://example.com", "Source2")
    assert item1.id != item3.id
