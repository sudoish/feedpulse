"""Feed data normalization and parsing.

This module provides parsers for different feed types:
- HackerNews API
- Reddit JSON
- Lobsters JSON
- GitHub Events API

Each parser normalizes feed-specific formats into FeedItem objects.
"""

import sys
import json
from typing import Any, Optional

from .models import FeedItem
from .utils import coerce_to_string, parse_timestamp
from .exceptions import MalformedJSONError, UnexpectedStructureError


def parse_hackernews(data: Any, source: str) -> list[FeedItem]:
    """Parse HackerNews API response.
    
    Supports two formats:
    1. Top stories API: array of story IDs [12345, 23456, ...]
    2. Algolia search API: {hits: [{objectID, title, url, ...}, ...]}
    
    Args:
        data: Parsed JSON data
        source: Source feed name
        
    Returns:
        List of FeedItem objects
    """
    items: list[FeedItem] = []
    
    try:
        # Check if it's the Algolia format with 'hits'
        if isinstance(data, dict) and 'hits' in data:
            hits = data.get('hits', [])
            if not isinstance(hits, list):
                print(f"Warning: {source}: 'hits' must be a list", file=sys.stderr)
                return items
            
            for i, hit in enumerate(hits):
                try:
                    if not isinstance(hit, dict):
                        continue
                    
                    # Get title
                    title = hit.get('title')
                    if not title:
                        continue
                    title = coerce_to_string(title)
                    if not title:
                        continue
                    
                    # Get URL (or generate from objectID)
                    url = hit.get('url')
                    if not url:
                        story_id = hit.get('objectID') or hit.get('story_id')
                        if story_id:
                            url = f"https://news.ycombinator.com/item?id={story_id}"
                        else:
                            continue
                    
                    url = coerce_to_string(url)
                    if not url:
                        continue
                    
                    # Parse timestamp
                    timestamp = None
                    created_at = hit.get('created_at_i') or hit.get('created_at')
                    if created_at:
                        timestamp = parse_timestamp(created_at)
                    
                    item = FeedItem.create(
                        title=title,
                        url=url,
                        source=source,
                        timestamp=timestamp,
                        raw_data=hit
                    )
                    items.append(item)
                    
                except Exception as e:
                    print(f"Warning: {source}: item {i}: failed to parse: {e}", file=sys.stderr)
                    continue
            
            return items
        
        # Original format: list of story IDs
        if not isinstance(data, list):
            print(f"Warning: {source}: expected list or dict with 'hits', got {type(data).__name__}", file=sys.stderr)
            return items
        
        for i, story_id in enumerate(data[:30]):  # Limit to top 30
            try:
                # Validate ID
                if not isinstance(story_id, int):
                    print(f"Warning: {source}: item {i}: expected int ID, got {type(story_id).__name__}", 
                          file=sys.stderr)
                    continue
                
                title = f"HN Story {story_id}"
                url = f"https://news.ycombinator.com/item?id={story_id}"
                
                item = FeedItem.create(
                    title=title,
                    url=url,
                    source=source,
                    raw_data={'id': story_id}
                )
                items.append(item)
                
            except Exception as e:
                print(f"Warning: {source}: item {i}: failed to parse: {e}", file=sys.stderr)
                continue
    
    except Exception as e:
        print(f"Error: {source}: failed to parse HackerNews data: {e}", file=sys.stderr)
    
    return items


def parse_github(data: Any, source: str) -> list[FeedItem]:
    """
    Parse GitHub API response.
    Extract from items[]: full_name → title, html_url → url, 
    topics → tags, updated_at → timestamp
    """
    items: list[FeedItem] = []
    
    try:
        if not isinstance(data, dict):
            print(f"Warning: {source}: expected dict, got {type(data).__name__}", file=sys.stderr)
            return items
        
        github_items = data.get('items', [])
        if not isinstance(github_items, list):
            print(f"Warning: {source}: 'items' must be a list", file=sys.stderr)
            return items
        
        for i, item_data in enumerate(github_items):
            try:
                if not isinstance(item_data, dict):
                    print(f"Warning: {source}: item {i}: must be a dict", file=sys.stderr)
                    continue
                
                # Required: full_name → title
                title = item_data.get('full_name')
                if not title:
                    title = item_data.get('name')  # Fallback
                if not title:
                    print(f"Warning: {source}: item {i}: missing 'full_name' or 'name'", file=sys.stderr)
                    continue
                
                title = coerce_to_string(title)
                if not title:
                    print(f"Warning: {source}: item {i}: could not convert title to string", file=sys.stderr)
                    continue
                
                # Required: html_url → url
                url = item_data.get('html_url')
                if not url:
                    print(f"Warning: {source}: item {i}: missing 'html_url'", file=sys.stderr)
                    continue
                
                url = coerce_to_string(url)
                if not url:
                    print(f"Warning: {source}: item {i}: could not convert url to string", file=sys.stderr)
                    continue
                
                # Optional: topics → tags
                tags = item_data.get('topics', [])
                if not isinstance(tags, list):
                    tags = []
                else:
                    tags = [str(t) for t in tags if t]
                
                # Optional: updated_at → timestamp
                timestamp = parse_timestamp(item_data.get('updated_at'))
                
                item = FeedItem.create(
                    title=title,
                    url=url,
                    source=source,
                    timestamp=timestamp,
                    tags=tags,
                    raw_data=item_data
                )
                items.append(item)
                
            except Exception as e:
                print(f"Warning: {source}: item {i}: failed to parse: {e}", file=sys.stderr)
                continue
    
    except Exception as e:
        print(f"Error: {source}: failed to parse GitHub data: {e}", file=sys.stderr)
    
    return items


def parse_reddit(data: Any, source: str) -> list[FeedItem]:
    """
    Parse Reddit API response.
    Extract from data.children[].data: title → title, url → url,
    created_utc → timestamp, link_flair_text → tags
    """
    items: list[FeedItem] = []
    
    try:
        if not isinstance(data, dict):
            print(f"Warning: {source}: expected dict, got {type(data).__name__}", file=sys.stderr)
            return items
        
        data_obj = data.get('data', {})
        if not isinstance(data_obj, dict):
            print(f"Warning: {source}: 'data' must be a dict", file=sys.stderr)
            return items
        
        children = data_obj.get('children', [])
        if not isinstance(children, list):
            print(f"Warning: {source}: 'children' must be a list", file=sys.stderr)
            return items
        
        for i, child in enumerate(children):
            try:
                if not isinstance(child, dict):
                    print(f"Warning: {source}: item {i}: child must be a dict", file=sys.stderr)
                    continue
                
                item_data = child.get('data', {})
                if not isinstance(item_data, dict):
                    print(f"Warning: {source}: item {i}: child.data must be a dict", file=sys.stderr)
                    continue
                
                # Required: title
                title = item_data.get('title')
                if not title:
                    print(f"Warning: {source}: item {i}: missing 'title'", file=sys.stderr)
                    continue
                
                title = coerce_to_string(title)
                if not title:
                    print(f"Warning: {source}: item {i}: could not convert title to string", file=sys.stderr)
                    continue
                
                # Required: url
                url = item_data.get('url')
                if not url:
                    print(f"Warning: {source}: item {i}: missing 'url'", file=sys.stderr)
                    continue
                
                url = coerce_to_string(url)
                if not url:
                    print(f"Warning: {source}: item {i}: could not convert url to string", file=sys.stderr)
                    continue
                
                # Optional: created_utc → timestamp (unix timestamp)
                timestamp = parse_timestamp(item_data.get('created_utc'))
                
                # Optional: link_flair_text → tags (single item)
                tags = []
                flair = item_data.get('link_flair_text')
                if flair:
                    flair_str = coerce_to_string(flair)
                    if flair_str:
                        tags = [flair_str]
                
                item = FeedItem.create(
                    title=title,
                    url=url,
                    source=source,
                    timestamp=timestamp,
                    tags=tags,
                    raw_data=item_data
                )
                items.append(item)
                
            except Exception as e:
                print(f"Warning: {source}: item {i}: failed to parse: {e}", file=sys.stderr)
                continue
    
    except Exception as e:
        print(f"Error: {source}: failed to parse Reddit data: {e}", file=sys.stderr)
    
    return items


def parse_lobsters(data: Any, source: str) -> list[FeedItem]:
    """
    Parse Lobsters API response.
    Extract from root array: title → title, url → url (fallback to comments_url),
    created_at → timestamp, tags → tags
    """
    items: list[FeedItem] = []
    
    try:
        if not isinstance(data, list):
            print(f"Warning: {source}: expected list, got {type(data).__name__}", file=sys.stderr)
            return items
        
        for i, item_data in enumerate(data):
            try:
                if not isinstance(item_data, dict):
                    print(f"Warning: {source}: item {i}: must be a dict", file=sys.stderr)
                    continue
                
                # Required: title
                title = item_data.get('title')
                if not title:
                    print(f"Warning: {source}: item {i}: missing 'title'", file=sys.stderr)
                    continue
                
                title = coerce_to_string(title)
                if not title:
                    print(f"Warning: {source}: item {i}: could not convert title to string", file=sys.stderr)
                    continue
                
                # Required: url (fallback to comments_url)
                url = item_data.get('url') or item_data.get('comments_url')
                if not url:
                    print(f"Warning: {source}: item {i}: missing 'url' and 'comments_url'", file=sys.stderr)
                    continue
                
                url = coerce_to_string(url)
                if not url:
                    print(f"Warning: {source}: item {i}: could not convert url to string", file=sys.stderr)
                    continue
                
                # Optional: created_at → timestamp
                timestamp = parse_timestamp(item_data.get('created_at'))
                
                # Optional: tags
                tags = item_data.get('tags', [])
                if not isinstance(tags, list):
                    tags = []
                else:
                    tags = [str(t) for t in tags if t]
                
                item = FeedItem.create(
                    title=title,
                    url=url,
                    source=source,
                    timestamp=timestamp,
                    tags=tags,
                    raw_data=item_data
                )
                items.append(item)
                
            except Exception as e:
                print(f"Warning: {source}: item {i}: failed to parse: {e}", file=sys.stderr)
                continue
    
    except Exception as e:
        print(f"Error: {source}: failed to parse Lobsters data: {e}", file=sys.stderr)
    
    return items


def parse_feed(response_body: str, feed_type: str, source: str) -> list[FeedItem]:
    """Parse feed response based on feed_type.
    
    Supports direct feed types (hackernews, reddit, github, lobsters)
    and auto-detection mode (json).
    
    Args:
        response_body: Raw feed response (JSON string)
        feed_type: Feed type ('hackernews', 'reddit', 'github', 'lobsters', or 'json')
        source: Source feed name (for logging)
        
    Returns:
        List of normalized FeedItem objects
    """
    # Parse JSON
    try:
        data = json.loads(response_body)
    except json.JSONDecodeError as e:
        print(f"Error: {source}: malformed JSON response: {e}", file=sys.stderr)
        return []
    except Exception as e:
        print(f"Error: {source}: failed to parse response: {e}", file=sys.stderr)
        return []
    
    # Route to appropriate parser based on explicit feed_type
    if feed_type == 'hackernews':
        return parse_hackernews(data, source)
    elif feed_type == 'reddit':
        return parse_reddit(data, source)
    elif feed_type == 'github':
        return parse_github(data, source)
    elif feed_type == 'lobsters':
        return parse_lobsters(data, source)
    elif feed_type == 'json':
        # Auto-detect based on structure
        if isinstance(data, list):
            # Could be HackerNews (list of ints) or Lobsters (list of dicts)
            if data and isinstance(data[0], int):
                return parse_hackernews(data, source)
            else:
                return parse_lobsters(data, source)
        elif isinstance(data, dict):
            if 'items' in data:
                return parse_github(data, source)
            elif 'data' in data and isinstance(data.get('data'), dict):
                return parse_reddit(data, source)
            elif 'hits' in data:
                return parse_hackernews(data, source)
        
        print(f"Warning: {source}: unable to auto-detect JSON feed structure", file=sys.stderr)
        return []
    elif feed_type in ('rss', 'atom'):
        # Not implemented for this experiment
        print(f"Warning: {source}: RSS/Atom parsing not implemented", file=sys.stderr)
        return []
    else:
        print(f"Warning: {source}: unknown feed_type '{feed_type}'", file=sys.stderr)
        return []
