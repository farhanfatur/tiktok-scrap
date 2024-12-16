package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/chromedp/chromedp"
)

type Video struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Thumbnail string `json:"thumbnail"`
}

func main() {
	http.HandleFunc("/search", handleSearch)
	log.Println("Server is running on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	// Extract query parameters
	keyword := r.URL.Query().Get("q")
	skipStr := r.URL.Query().Get("skip")
	takeStr := r.URL.Query().Get("take")

	if keyword == "" || skipStr == "" || takeStr == "" {
		http.Error(w, "Missing required query parameters (q, skip, take)", http.StatusBadRequest)
		return
	}

	// Convert skip and take to integers
	skip, err := strconv.Atoi(skipStr)
	if err != nil {
		http.Error(w, "Invalid skip value", http.StatusBadRequest)
		return
	}

	take, err := strconv.Atoi(takeStr)
	if err != nil {
		http.Error(w, "Invalid take value", http.StatusBadRequest)
		return
	}

	// Perform scraping
	results, err := scrapeTikTokVideos(keyword, skip, take)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error scraping videos: %v", err), http.StatusInternalServerError)
		return
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"results": %+v}`, results)
}

func scrapeTikTokVideos(keyword string, skip, take int) ([]Video, error) {
	// Create a new Chromedp context
	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	// Timeout for the scraping
	ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	// TikTok search URL
	searchURL := fmt.Sprintf("https://www.tiktok.com/search?q=%s", keyword)

	var htmlContent string

	// Perform Chromedp tasks
	err := chromedp.Run(ctx,
		chromedp.Navigate(searchURL),
		// chromedp.Sleep(5*time.Second), // Allow time for initial load
		scrollPage(ctx, 3), // Simulate scrolling to load more content
		chromedp.OuterHTML(`div#tabs-0-panel-search_top`, &htmlContent),
	)
	if err != nil {
		return nil, fmt.Errorf("chromedp failed: %w", err)
	}
	fmt.Println("PRINT HTML:")
	// Extract videos from HTML using GoQuery
	videos := extractVideos(htmlContent)

	// Apply pagination (skip and take)
	start := skip * take
	end := start + take
	if start > len(videos) {
		return []Video{}, nil
	}
	if end > len(videos) {
		end = len(videos)
	}

	return videos[start:end], nil
}

func scrollPage(ctx context.Context, scrollCount int) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		for i := 0; i < scrollCount; i++ {
			if err := chromedp.Run(ctx, chromedp.Evaluate(`window.scrollBy(0, window.innerHeight)`, nil)); err != nil {
				return err
			}
			time.Sleep(2 * time.Second) // Allow time for content to load
		}
		return nil
	}
}

func extractVideos(htmlContent string) []Video {
	fmt.Println("MLAKUK")
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		log.Printf("Error parsing HTML: %v", err)
		return nil
	}

	var videos []Video
	data := doc.Find("div.css-x6y88p-DivItemContainerV2")
	if data.Length() == 0 {
		log.Println("No video cards found!")
		return nil
	}
	data.Each(func(i int, s *goquery.Selection) {
		fmt.Println("I =>", i)
		fmt.Println("S =>", s.Text())
		title := s.Find("span[data-e2e='new-desc-span']").Text()
		url, _ := s.Find("a").Attr("href")
		thumbnail, _ := s.Find("img").Attr("src")

		videos = append(videos, Video{
			Title:     title,
			URL:       url,
			Thumbnail: thumbnail,
		})
	})

	return videos
}
