"""Feed data normalization and parsing"""

import sys
import json
from typing import List, Optional, Any, Dict
from datetime import datetime

from .models import FeedItem


def coerce_to_string(value: Any) -> Optional[str]:
    """Try to convert value to string"""
    if isinstance(value, str):
        return value
    try:
        return str(value)
    except Exception:
        return None


def parse_timestamp(value: Any) -> Optional[str]:
    """Parse various timestamp formats to ISO 8601"""
    if not value:
        return None
    
    try:
        # Unix timestamp (int or float)
        if isinstance(value, (int, float)):
            dt = datetime.utcfromtimestamp(value)
            return dt.isoformat()
        
        # ISO 8601 string
        if isinstance(value, str):
            # Try parsing common formats
            for fmt in ['%Y-%m-%dT%H:%M:%S', '%Y-%m-%dT%H:%M:%SZ', 
                       '%Y-%m-%d %H:%M:%S', '%Y-%m-%d']:
                try:
                    dt = datetime.strptime(value.replace('Z', '').split('+')[0].split('.')[0], fmt)
                    return dt.isoformat()
                except ValueError:
                    continue
            # If it's already a valid ISO string, return as-is
            return value
    except Exception:
        pass
    
    return None


def parse_hackernews(data: Any, source: str) -> List[FeedItem]:
    """
    Parse HackerNews API response.
    Top stories returns an array of IDs. Store IDs as items.
    """
    items = []
    
    try:
        if not isinstance(data, list):
            print(f"Warning: {source}: expected list, got {type(data).__name__}", file=sys.stderr)
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


def parse_github(data: Any, source: str) -> List[FeedItem]:
    """
    Parse GitHub API response.
    Extract from items[]: full_name → title, html_url → url, 
    topics → tags, updated_at → timestamp
    """
    items = []
    
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


def parse_reddit(data: Any, source: str) -> List[FeedItem]:
    """
    Parse Reddit API response.
    Extract from data.children[].data: title → title, url → url,
    created_utc → timestamp, link_flair_text → tags
    """
    items = []
    
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


def parse_lobsters(data: Any, source: str) -> List[FeedItem]:
    """
    Parse Lobsters API response.
    Extract from root array: title → title, url → url (fallback to comments_url),
    created_at → timestamp, tags → tags
    """
    items = []
    
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


def parse_feed(response_body: str, feed_type: str, source: str) -> List[FeedItem]:
    """
    Parse feed response based on feed_type.
    Returns list of normalized FeedItems.
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
    
    # Route to appropriate parser
    if feed_type == 'json':
        # Auto-detect based on structure
        # This is a simplification - in production we'd need better detection
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
        
        print(f"Warning: {source}: unable to auto-detect JSON feed structure", file=sys.stderr)
        return []
    
    elif feed_type in ('rss', 'atom'):
        # Not implemented for this experiment
        print(f"Warning: {source}: RSS/Atom parsing not implemented", file=sys.stderr)
        return []
    
    else:
        print(f"Warning: {source}: unknown feed_type '{feed_type}'", file=sys.stderr)
        return []
