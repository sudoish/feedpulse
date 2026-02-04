"""Concurrent feed fetching with retries and backoff"""

import sys
import asyncio
import random
from typing import List
from datetime import datetime
import aiohttp

from .models import FeedConfig, Settings, FetchResult
from .parser import parse_feed


async def fetch_feed_with_retry(
    session: aiohttp.ClientSession,
    feed: FeedConfig,
    settings: Settings
) -> FetchResult:
    """
    Fetch a single feed with retry logic and exponential backoff.
    """
    start_time = datetime.utcnow()
    
    for attempt in range(settings.retry_max + 1):
        try:
            timeout = aiohttp.ClientTimeout(total=settings.default_timeout_secs)
            
            async with session.get(
                feed.url,
                headers=feed.headers,
                timeout=timeout
            ) as response:
                # Handle HTTP errors
                if response.status == 404:
                    # No retry for 404
                    duration = int((datetime.utcnow() - start_time).total_seconds() * 1000)
                    return FetchResult(
                        source=feed.name,
                        status='error',
                        error_message=f"HTTP 404",
                        duration_ms=duration
                    )
                
                if response.status >= 400:
                    # 4xx/5xx - will retry
                    error_msg = f"HTTP {response.status}"
                    
                    if attempt < settings.retry_max:
                        # Calculate backoff delay with jitter
                        delay_ms = settings.retry_base_delay_ms * (2 ** attempt)
                        jitter = random.uniform(0.8, 1.2)
                        delay_sec = (delay_ms * jitter) / 1000
                        await asyncio.sleep(delay_sec)
                        continue
                    else:
                        # Final attempt failed
                        duration = int((datetime.utcnow() - start_time).total_seconds() * 1000)
                        return FetchResult(
                            source=feed.name,
                            status='error',
                            error_message=f"{error_msg} after {settings.retry_max} retries",
                            duration_ms=duration
                        )
                
                # Success - parse response
                body = await response.text()
                items = parse_feed(body, feed.feed_type, feed.name)
                
                duration = int((datetime.utcnow() - start_time).total_seconds() * 1000)
                return FetchResult(
                    source=feed.name,
                    status='success',
                    items=items,
                    duration_ms=duration
                )
        
        except asyncio.TimeoutError:
            error_msg = "timeout"
            
            if attempt < settings.retry_max:
                delay_ms = settings.retry_base_delay_ms * (2 ** attempt)
                jitter = random.uniform(0.8, 1.2)
                delay_sec = (delay_ms * jitter) / 1000
                await asyncio.sleep(delay_sec)
                continue
            else:
                duration = int((datetime.utcnow() - start_time).total_seconds() * 1000)
                return FetchResult(
                    source=feed.name,
                    status='error',
                    error_message=f"{error_msg} after {settings.retry_max} retries",
                    duration_ms=duration
                )
        
        except aiohttp.ClientError as e:
            # Network errors (DNS, connection refused, etc.)
            error_msg = str(e)
            
            if attempt < settings.retry_max:
                delay_ms = settings.retry_base_delay_ms * (2 ** attempt)
                jitter = random.uniform(0.8, 1.2)
                delay_sec = (delay_ms * jitter) / 1000
                await asyncio.sleep(delay_sec)
                continue
            else:
                duration = int((datetime.utcnow() - start_time).total_seconds() * 1000)
                return FetchResult(
                    source=feed.name,
                    status='error',
                    error_message=f"{error_msg} after {settings.retry_max} retries",
                    duration_ms=duration
                )
        
        except Exception as e:
            # Unexpected error
            duration = int((datetime.utcnow() - start_time).total_seconds() * 1000)
            return FetchResult(
                source=feed.name,
                status='error',
                error_message=f"unexpected error: {e}",
                duration_ms=duration
            )
    
    # Should not reach here
    duration = int((datetime.utcnow() - start_time).total_seconds() * 1000)
    return FetchResult(
        source=feed.name,
        status='error',
        error_message="retry loop exhausted",
        duration_ms=duration
    )


async def fetch_all_feeds(feeds: List[FeedConfig], settings: Settings) -> List[FetchResult]:
    """
    Fetch all feeds concurrently with semaphore to limit concurrency.
    """
    semaphore = asyncio.Semaphore(settings.max_concurrency)
    
    async def fetch_with_semaphore(feed: FeedConfig) -> FetchResult:
        async with semaphore:
            return await fetch_feed_with_retry(session, feed, settings)
    
    # Create session with reasonable defaults
    timeout = aiohttp.ClientTimeout(total=settings.default_timeout_secs)
    connector = aiohttp.TCPConnector(limit=settings.max_concurrency)
    
    try:
        async with aiohttp.ClientSession(
            timeout=timeout,
            connector=connector
        ) as session:
            # Fetch all feeds concurrently
            tasks = [fetch_with_semaphore(feed) for feed in feeds]
            results = await asyncio.gather(*tasks, return_exceptions=True)
            
            # Convert exceptions to error results
            final_results = []
            for i, result in enumerate(results):
                if isinstance(result, BaseException):
                    final_results.append(FetchResult(
                        source=feeds[i].name,
                        status='error',
                        error_message=f"unexpected error: {result}",
                        duration_ms=0
                    ))
                elif isinstance(result, FetchResult):
                    final_results.append(result)
            
            return final_results
    
    except Exception as e:
        # Session creation failed
        print(f"Error: failed to create HTTP session: {e}", file=sys.stderr)
        return [
            FetchResult(
                source=feed.name,
                status='error',
                error_message=f"session error: {e}",
                duration_ms=0
            )
            for feed in feeds
        ]


def fetch_feeds_sync(feeds: List[FeedConfig], settings: Settings) -> List[FetchResult]:
    """Synchronous wrapper for async fetch_all_feeds"""
    try:
        loop = asyncio.get_event_loop()
        if loop.is_running():
            # Create new event loop if one is already running
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
    except RuntimeError:
        loop = asyncio.new_event_loop()
        asyncio.set_event_loop(loop)
    
    try:
        return loop.run_until_complete(fetch_all_feeds(feeds, settings))
    except KeyboardInterrupt:
        # Handle Ctrl+C gracefully
        print("\nCancelling pending fetches...", file=sys.stderr)
        # Cancel all tasks
        pending = asyncio.all_tasks(loop)
        for task in pending:
            task.cancel()
        # Wait for cancellation
        loop.run_until_complete(asyncio.gather(*pending, return_exceptions=True))
        return []
    finally:
        # Don't close the loop as it might be the main event loop
        pass
