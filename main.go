package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"time"
    "html/template"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/gorilla/mux"
)

type Subtitle struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Code string `json:"code,omitempty"`
}

type Response struct {
	State              int        `json:"state"`
	Title              string     `json:"title"`
	Thumbnail          string     `json:"thumbnail"`
	Duration           string     `json:"duration"`
	Subtitles          []Subtitle `json:"subtitles"`
	SubtitlesAutoTrans []Subtitle `json:"subtitlesAutoTrans"`
}

func getSubtitles(videoID string) (string, error) {
	downsubURL := fmt.Sprintf("https://downsub.com/?url=https://www.youtube.com/watch?v=%s", videoID)

	ctx, cancel := chromedp.NewContext(context.Background())
	defer cancel()

	ctx, cancel = context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var requests []string
	var subtitleURL string

	err := chromedp.Run(ctx,
		network.Enable(),
		chromedp.ActionFunc(func(ctx context.Context) error {
			chromedp.ListenTarget(ctx, func(ev interface{}) {
				if req, ok := ev.(*network.EventRequestWillBeSent); ok {
					requests = append(requests, req.Request.URL)
				}
			})
			return nil
		}),
		chromedp.Navigate(downsubURL),
		chromedp.WaitVisible("body"),
	)

	if err != nil {
		return "", fmt.Errorf("failed to navigate: %v", err)
	}

	re, err := regexp.Compile(`https://get-info\.downsub\.com/.+`)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %v", err)
	}

	for _, req := range requests {
		if re.MatchString(req) {
			subtitleURL = req
			break
		}
	}

	if subtitleURL == "" {
		return "Subtitle link not found", nil
	}

	var subtitleContent string
	err = chromedp.Run(ctx,
		chromedp.Navigate(subtitleURL),
		chromedp.OuterHTML("html", &subtitleContent),
	)

	if err != nil {
		return "", fmt.Errorf("failed to fetch subtitle: %v", err)
	}

	re = regexp.MustCompile(`<pre>(.+)</pre>`)
	subtitle := re.FindStringSubmatch(subtitleContent)
	if len(subtitle) < 2 {
		return "Subtitle not found", nil
	}

	return subtitle[1], nil
}

func subtitlesHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	videoID := vars["videoID"]

	// Assuming getSubtitles returns a JSON string
	subtitlesJSON, err := getSubtitles(videoID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var response Response
	err = json.Unmarshal([]byte(subtitlesJSON), &response)
	if err != nil {
		http.Error(w, "Failed to parse JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        tmpl := template.Must(template.ParseFiles("ui/index.html"))
        tmpl.Execute(w, nil)

	})

	r.HandleFunc("/subtitles/{videoID}", subtitlesHandler).Methods("GET")

	fmt.Println("Starting server on :8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
