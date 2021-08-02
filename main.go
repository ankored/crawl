package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ankored/crawl/filter"
	"github.com/ankored/crawl/scrape"
)

func main() {
	var (
		workers = flag.Int("n", 1, "specify the number of workers to use")
	)
	flag.Parse()

	// Check that the user provided a positive value for workers
	if *workers < 1 {
		fmt.Print("number of workers must be greater than zero")
		os.Exit(1)
	}

	// Check that an argument was provided
	if flag.NArg() != 1 {
		fmt.Print("must specify a single url to start crawling, e.g. crawl https://google.com")
		os.Exit(1)
	}

	// Also check that the user provided a reasonable url
	baseURL, err := url.Parse(flag.Arg(0))
	if err != nil {
		fmt.Printf("error parsing starting url: %s", err)
		os.Exit(1)
	}

	// Listen for cancellation signals from the command line
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	realMain(ctx, baseURL, *workers)
}

func realMain(baseCtx context.Context, base *url.URL, numWorkers int) {
	ctx, cancel := context.WithCancel(baseCtx)

	// This is the queue of urls to fan-out to workers
	queue := make(chan *url.URL, 64)

	// We can use a "free-list", which is a channel acting as a pool
	// of available scrapers, which will have a capacity equal to number of
	// specified workers. With a limited number of scrapers to take, only n
	// number of api calls will take place at a time.
	freeList := make(chan scrape.Scraper, numWorkers)

	// Fill the free list with the pool of workers
	for i := 0; i < numWorkers; i++ {
		freeList <- scrape.NewScraper(&http.Client{
			Timeout: 5 * time.Second,
		})
	}

	// To signal when a worker has performed a unit of work,
	// it can send a struct over this channel.
	// That will cause the base case to be checked.
	done := make(chan struct{})

	// Filter does the logic for saying which urls to visit and which ones to throw away
	f := filter.NewFilter(base)

	go func() {
		// Either pull from the queue and start a worker when a scraper is
		// available from the list (limiting execution to n workers), or
		// check for the base case occuring, which is all scrapers free
		// and no more urls to process.
		for {
			select {
			case u := <-queue:
				if !f.Visit(u) {
					continue
				}
				// As per the requirements, print out the url
				fmt.Println(u.String())

				// Get a scraper to perform work,
				// this limits executing routines to n workers.
				scraper := <-freeList

				go func() {
					defer func() {
						done <- struct{}{}
					}()

					urls, _ := scraper.FetchLinks(u)

					// Release the scraper
					freeList <- scraper

					// Add any found urls back into the queue
					for _, u := range urls {
						queue <- u
					}
				}()
			case <-ctx.Done():
				return
			case <-done:
				// Check for the base case: no urls to process, no one is running
				if len(freeList) == numWorkers && len(queue) == 0 {
					cancel()
				}
			}
		}
	}()

	// Start the flow with the first url provided
	queue <- base

	// Wait until the event loop is done or cancellation is called
	<-ctx.Done()
}
