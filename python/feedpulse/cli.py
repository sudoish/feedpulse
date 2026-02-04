"""CLI interface for feedpulse"""

import sys
import click
from rich.console import Console
from rich.table import Table

from . import __version__
from .config import load_config
from .fetcher import fetch_feeds_sync
from .storage import FeedDatabase, DatabaseError


console = Console()


@click.group()
@click.version_option(version=__version__, prog_name='feedpulse')
def cli():
    """feedpulse - Concurrent Feed Aggregator CLI"""
    pass


@cli.command()
@click.option('--config', default='config.yaml', help='Path to config file')
def fetch(config):
    """Fetch all feeds and store results"""
    
    # Load configuration
    cfg = load_config(config)
    
    # Print header
    console.print(f"\nFetching {len(cfg.feeds)} feeds (max concurrency: {cfg.settings.max_concurrency})...")
    
    # Fetch all feeds
    try:
        results = fetch_feeds_sync(cfg.feeds, cfg.settings)
    except KeyboardInterrupt:
        console.print("\n[yellow]Cancelled by user[/yellow]")
        sys.exit(130)
    
    if not results:
        console.print("[yellow]No results (cancelled?)[/yellow]")
        return
    
    # Store results
    try:
        db = FeedDatabase(cfg.settings.database_path)
        total_new = db.store_results(results)
    except DatabaseError as e:
        console.print(f"[red]Error: {e}[/red]", file=sys.stderr)
        sys.exit(1)
    
    # Print results
    success_count = 0
    error_count = 0
    total_items = 0
    
    for result in results:
        if result.status == 'success':
            success_count += 1
            total_items += len(result.items)
            new_count = result.items_new
            console.print(
                f"  [green]✓[/green] {result.source:<30} — {len(result.items)} items "
                f"({new_count} new) in {result.duration_ms}ms"
            )
        else:
            error_count += 1
            console.print(
                f"  [red]✗[/red] {result.source:<30} — error: {result.error_message}"
            )
    
    # Summary
    console.print(
        f"\nDone: {success_count}/{len(cfg.feeds)} succeeded, "
        f"{total_items} items ({total_new} new), "
        f"{error_count} error{'s' if error_count != 1 else ''}"
    )


@cli.command()
@click.option('--config', default='config.yaml', help='Path to config file')
@click.option('--format', type=click.Choice(['table', 'json', 'csv']), default='table',
              help='Output format')
@click.option('--source', help='Filter by source name')
@click.option('--since', help='Filter items newer than (e.g., "24h", "7d")')
def report(config, format, source, since):
    """Generate summary report"""
    
    # Load configuration
    cfg = load_config(config)
    
    # Get report data
    try:
        db = FeedDatabase(cfg.settings.database_path)
        report_data = db.get_report_data(source_filter=source)
    except DatabaseError as e:
        console.print(f"[red]Error: {e}[/red]", file=sys.stderr)
        sys.exit(1)
    
    if not report_data:
        console.print("No data available")
        return
    
    # Output based on format
    if format == 'table':
        table = Table(title=None, show_header=True, header_style="bold")
        table.add_column("Source", style="cyan")
        table.add_column("Items", justify="right")
        table.add_column("Errors", justify="right")
        table.add_column("Error Rate", justify="right")
        table.add_column("Last Success")
        
        total_items = 0
        for row in report_data:
            table.add_row(
                row['source'],
                str(row['items']),
                str(row['errors']),
                f"{row['error_rate']:.1f}%",
                row['last_success'][:16] if row['last_success'] != 'never' else 'never'
            )
            total_items += row['items']
        
        console.print("\n")
        console.print(table)
        console.print(f"\nTotal: {total_items} items across {len(report_data)} sources\n")
    
    elif format == 'json':
        import json
        print(json.dumps(report_data, indent=2))
    
    elif format == 'csv':
        import csv
        import io
        output = io.StringIO()
        writer = csv.DictWriter(output, fieldnames=['source', 'items', 'errors', 'error_rate', 'last_success'])
        writer.writeheader()
        writer.writerows(report_data)
        print(output.getvalue())


@cli.command()
@click.option('--config', default='config.yaml', help='Path to config file')
def sources(config):
    """List configured sources and their status"""
    
    # Load configuration
    cfg = load_config(config)
    
    # Get sources status
    try:
        db = FeedDatabase(cfg.settings.database_path)
        sources_status = db.get_sources_status()
    except DatabaseError as e:
        # Database might not exist yet
        sources_status = []
    
    # Create lookup for status
    status_map = {s['source']: s for s in sources_status}
    
    # Print configured sources
    table = Table(title="Configured Sources", show_header=True, header_style="bold")
    table.add_column("Name", style="cyan")
    table.add_column("Type")
    table.add_column("Status")
    table.add_column("Last Fetch")
    table.add_column("URL", style="dim")
    
    for feed in cfg.feeds:
        status_info = status_map.get(feed.name)
        
        if status_info:
            status = "✓" if status_info['status'] == 'success' else "✗"
            last_fetch = status_info['last_fetch'][:16] if status_info['last_fetch'] else 'never'
        else:
            status = "—"
            last_fetch = "never"
        
        table.add_row(
            feed.name,
            feed.feed_type,
            status,
            last_fetch,
            feed.url[:50] + "..." if len(feed.url) > 50 else feed.url
        )
    
    console.print("\n")
    console.print(table)
    console.print()


def main():
    """Main entry point"""
    try:
        cli()
    except Exception as e:
        console.print(f"[red]Error: {e}[/red]", file=sys.stderr)
        sys.exit(1)


if __name__ == '__main__':
    main()
