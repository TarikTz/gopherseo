package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/tariktz/gopherseo/internal/crawler"
	"github.com/tariktz/gopherseo/internal/output"
)

type crawlOptions struct {
	output          string
	issuesOutput    string
	canonicalOutput string
	threads         int
	depth           int
	userAgent       string
	excludePatterns []string
	timeout         time.Duration
}

func init() {
	opts := &crawlOptions{}

	crawlCmd := &cobra.Command{
		Use:   "crawl <url>",
		Short: "Crawl a domain and export a sitemap.xml",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			rootURL := strings.TrimSpace(args[0])

			spinnerStop := make(chan struct{})
			spinnerDone := make(chan struct{})
			go func() {
				defer close(spinnerDone)
				frames := []rune{'|', '/', '-', '\\'}
				i := 0
				ticker := time.NewTicker(200 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-spinnerStop:
						fmt.Fprint(os.Stderr, "\r")
						return
					case <-ticker.C:
						fmt.Fprintf(os.Stderr, "\rCrawling... %c", frames[i%len(frames)])
						i++
					}
				}
			}()

			result, err := crawler.Crawl(crawler.Options{
				RootURL:         rootURL,
				MaxDepth:        opts.depth,
				Threads:         opts.threads,
				UserAgent:       opts.userAgent,
				ExcludePatterns: opts.excludePatterns,
				RequestTimeout:  opts.timeout,
			})
			close(spinnerStop)
			<-spinnerDone
			if err != nil {
				return err
			}

			if err := output.WriteSitemap(opts.output, result.ValidURLs, result.LastModified); err != nil {
				return err
			}

			if err := output.WriteIssueTasks(opts.issuesOutput, result.BrokenLinkTasks); err != nil {
				return err
			}

			if err := output.WriteCanonicalIssues(opts.canonicalOutput, result.CanonicalIssues); err != nil {
				return err
			}

			fmt.Printf("\nCrawl complete\n")
			fmt.Printf("  Discovered:    %d\n", result.Discovered)
			fmt.Printf("  Valid URLs:    %d\n", len(result.ValidURLs))
			fmt.Printf("  Broken links:  %d\n", len(result.BrokenLinks))
			fmt.Printf("  Excluded URLs: %d\n", result.ExcludedURLs)
			fmt.Printf("  Canonical issues: %d\n", len(result.CanonicalIssues))
			fmt.Printf("  Missing canonical: %d\n", len(result.MissingCanonicalPages))
			fmt.Printf("  Multiple canonical: %d\n", len(result.MultipleCanonicalPages))
			fmt.Printf("\nSitemap written to %s\n", opts.output)
			fmt.Printf("Broken-link task report written to %s\n", opts.issuesOutput)
			fmt.Printf("Canonical issue report written to %s\n", opts.canonicalOutput)

			if len(result.BrokenLinks) > 0 {
				fmt.Fprintf(os.Stderr, "\nBroken links found (%d):\n", len(result.BrokenLinks))
				for link, status := range result.BrokenLinks {
					fmt.Fprintf(os.Stderr, "  [%d] %s\n", status, link)
				}
			}

			return nil
		},
	}

	crawlCmd.Flags().StringVarP(&opts.output, "output", "o", "./sitemap.xml", "Output sitemap file path")
	crawlCmd.Flags().StringVar(&opts.issuesOutput, "issues-output", "./broken-link-tasks.md", "Output file for broken-link cleanup tasks")
	crawlCmd.Flags().StringVar(&opts.canonicalOutput, "canonical-report-output", "./canonical-issues.md", "Output file for canonical URL issues")
	crawlCmd.Flags().IntVar(&opts.threads, "threads", 5, "Maximum concurrent crawler workers")
	crawlCmd.Flags().IntVar(&opts.depth, "depth", 0, "Max crawl depth (0 = unlimited)")
	crawlCmd.Flags().StringVar(&opts.userAgent, "user-agent", "GopherSEO-Bot/1.0", "Crawler user-agent")
	crawlCmd.Flags().StringSliceVar(&opts.excludePatterns, "exclude", []string{}, "Glob pattern to skip (repeatable)")
	crawlCmd.Flags().DurationVar(&opts.timeout, "timeout", 30*time.Second, "Timeout per HTTP request (e.g. 10s, 1m)")

	rootCmd.AddCommand(crawlCmd)
}
