package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"feedpulse/internal/config"
	"feedpulse/internal/fetcher"
	"feedpulse/internal/storage"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var (
	configPath string
	version    = "1.0.0"
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "feedpulse",
		Short:   "Concurrent feed aggregator CLI",
		Version: version,
		Long:    `feedpulse fetches multiple data feeds, validates and normalizes the data, stores results in SQLite, and generates summary reports.`,
	}

	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.yaml", "path to config file")

	rootCmd.AddCommand(newFetchCmd())
	rootCmd.AddCommand(newReportCmd())
	rootCmd.AddCommand(newSourcesCmd())

	return rootCmd
}

// newFetchCmd creates the fetch command
func newFetchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fetch",
		Short: "Fetch all feeds and store results",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch()
		},
	}
}

// newReportCmd creates the report command
func newReportCmd() *cobra.Command {
	var format string
	var sourceName string
	var since string

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Generate summary report",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReport(format, sourceName, since)
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format (table, json, csv)")
	cmd.Flags().StringVar(&sourceName, "source", "", "filter by source name")
	cmd.Flags().StringVar(&since, "since", "", "filter items newer than (e.g., '24h', '7d')")

	return cmd
}

// newSourcesCmd creates the sources command
func newSourcesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sources",
		Short: "List configured sources and their status",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSources()
		},
	}
}

// runFetch executes the fetch command
func runFetch() error {
	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return fmt.Errorf("config error")
	}

	// Open database
	store, err := storage.NewStorage(cfg.Settings.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open database: %v\n", err)
		return fmt.Errorf("database error")
	}
	defer store.Close()

	// Set up context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle Ctrl+C
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\nCancelling...\n")
		cancel()
	}()

	// Fetch feeds
	fmt.Printf("Fetching %d feeds (max concurrency: %d)...\n", len(cfg.Feeds), cfg.Settings.MaxConcurrency)

	f := fetcher.NewFetcher(cfg)
	results := f.FetchAll(ctx)

	// Process results
	successCount := 0
	errorCount := 0
	totalItems := 0
	totalNew := 0

	for _, result := range results {
		if result.Success {
			successCount++
			totalItems += result.ItemsCount

			// Get existing count before saving
			existingCount, err := store.GetItemCount(result.Source)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to get existing count for %s: %v\n", result.Source, err)
			}

			// Save items
			if len(result.Items) > 0 {
				if err := store.SaveItems(result.Items); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to save items for %s: %v\n", result.Source, err)
				} else {
					// Calculate new items
					newCount, _ := store.GetItemCount(result.Source)
					result.NewItems = newCount - existingCount
					totalNew += result.NewItems
				}
			}

			// Log success
			if err := store.LogFetch(storage.FetchLog{
				Source:     result.Source,
				FetchedAt:  time.Now(),
				Status:     "success",
				ItemsCount: result.ItemsCount,
				DurationMs: result.DurationMs,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log fetch for %s: %v\n", result.Source, err)
			}

			fmt.Printf("  ✓ %-30s — %d items (%d new) in %dms\n", result.Source, result.ItemsCount, result.NewItems, result.DurationMs)
		} else {
			errorCount++

			// Log error
			if err := store.LogFetch(storage.FetchLog{
				Source:       result.Source,
				FetchedAt:    time.Now(),
				Status:       "error",
				ErrorMessage: &result.Error,
				DurationMs:   result.DurationMs,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to log fetch for %s: %v\n", result.Source, err)
			}

			fmt.Printf("  ✗ %-30s — error: %s\n", result.Source, result.Error)
		}
	}

	fmt.Printf("\nDone: %d/%d succeeded, %d items (%d new)", successCount, len(results), totalItems, totalNew)
	if errorCount > 0 {
		fmt.Printf(", %d error(s)", errorCount)
	}
	fmt.Println()

	return nil
}

// runReport executes the report command
func runReport(format, sourceName, since string) error {
	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return fmt.Errorf("config error")
	}

	// Open database
	store, err := storage.NewStorage(cfg.Settings.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open database: %v\n", err)
		return fmt.Errorf("database error")
	}
	defer store.Close()

	// Get stats
	stats, err := store.GetFetchStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get stats: %v\n", err)
		return fmt.Errorf("stats error")
	}

	// Filter by source if requested
	if sourceName != "" {
		var filtered []storage.FetchStats
		for _, stat := range stats {
			if stat.Source == sourceName {
				filtered = append(filtered, stat)
			}
		}
		stats = filtered
	}

	// Get total items count
	totalItems, err := store.GetAllItemsCount()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to get total items: %v\n", err)
	}

	// Output based on format
	switch format {
	case "json":
		return outputJSON(stats, totalItems)
	case "csv":
		return outputCSV(stats)
	case "table":
		return outputTable(stats, totalItems)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

// runSources executes the sources command
func runSources() error {
	// Load config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return fmt.Errorf("config error")
	}

	// Open database
	store, err := storage.NewStorage(cfg.Settings.DatabasePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to open database: %v\n", err)
		return fmt.Errorf("database error")
	}
	defer store.Close()

	// Get stats
	stats, err := store.GetFetchStats()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to get stats: %v\n", err)
		return fmt.Errorf("stats error")
	}

	// Create a map for quick lookup
	statsMap := make(map[string]storage.FetchStats)
	for _, stat := range stats {
		statsMap[stat.Source] = stat
	}

	// Display configured sources
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Source", "URL", "Type", "Status")

	for _, feed := range cfg.Feeds {
		status := "never fetched"
		if stat, ok := statsMap[feed.Name]; ok {
			if stat.LastSuccess != nil {
				status = "✓ active"
			} else {
				status = "✗ failing"
			}
		}

		table.Append(feed.Name, feed.URL, feed.FeedType, status)
	}

	table.Render()
	return nil
}

// outputTable outputs stats in table format
func outputTable(stats []storage.FetchStats, totalItems int) error {
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("Source", "Items", "Errors", "Error Rate", "Last Success")

	for _, stat := range stats {
		errorRate := "0.0%"
		if stat.TotalFetches > 0 {
			rate := float64(stat.ErrorCount) / float64(stat.TotalFetches) * 100
			errorRate = fmt.Sprintf("%.1f%%", rate)
		}

		lastSuccess := "never"
		if stat.LastSuccess != nil {
			// Parse and format timestamp
			if t, err := time.Parse(time.RFC3339, *stat.LastSuccess); err == nil {
				lastSuccess = t.Format("2006-01-02 15:04")
			} else {
				lastSuccess = *stat.LastSuccess
			}
		}

		table.Append(
			stat.Source,
			fmt.Sprintf("%d", stat.ItemsCount),
			fmt.Sprintf("%d", stat.ErrorCount),
			errorRate,
			lastSuccess,
		)
	}

	table.Render()
	fmt.Printf("\nTotal: %d items across %d sources\n", totalItems, len(stats))
	return nil
}

// outputJSON outputs stats in JSON format
func outputJSON(stats []storage.FetchStats, totalItems int) error {
	output := map[string]interface{}{
		"sources":     stats,
		"total_items": totalItems,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}

// outputCSV outputs stats in CSV format
func outputCSV(stats []storage.FetchStats) error {
	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()

	// Write header
	if err := writer.Write([]string{"Source", "Items", "Errors", "Error Rate", "Last Success"}); err != nil {
		return err
	}

	// Write rows
	for _, stat := range stats {
		errorRate := "0.0"
		if stat.TotalFetches > 0 {
			errorRate = fmt.Sprintf("%.1f", float64(stat.ErrorCount)/float64(stat.TotalFetches)*100)
		}

		lastSuccess := "never"
		if stat.LastSuccess != nil {
			lastSuccess = *stat.LastSuccess
		}

		if err := writer.Write([]string{
			stat.Source,
			fmt.Sprintf("%d", stat.ItemsCount),
			fmt.Sprintf("%d", stat.ErrorCount),
			errorRate,
			lastSuccess,
		}); err != nil {
			return err
		}
	}

	return nil
}
