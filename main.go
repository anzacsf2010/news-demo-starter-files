package main

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/freshman-tech/news-demo-starter-files/news"
	"github.com/joho/godotenv"
	"github.com/newrelic/go-agent/v3/newrelic"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Search struct {
	Query      string
	NextPage   int
	TotalPages int
	Results    *news.Results
}

var temp = template.Must(template.ParseFiles("index.html"))

func indexHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "<html><head>")
	txn := newrelic.FromContext(r.Context())
	hdr := txn.BrowserTimingHeader()
	if js := hdr.WithTags(); js != nil {
		w.Write(js)
	}
	io.WriteString(w, "</head><body></body></html>")
	temp.Execute(w, nil)
}

func searchHandler(newsapi *news.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><head>")
		txn := newrelic.FromContext(r.Context())
		hdr := txn.BrowserTimingHeader()
		if js := hdr.WithTags(); js != nil {
			w.Write(js)
		}
		io.WriteString(w, "</head><body></body></html>")
		txn.NoticeError(errors.New("segment errors"))
		segment := newrelic.Segment{}
		segment.Name = "searchStart"
		segment.StartTime = txn.StartSegmentNow()
		u, err := url.Parse(r.URL.String())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		params := u.Query()
		searchQuery := params.Get("q")
		page := params.Get("page")
		if page == "" {
			page = "1"
		}

		txn.Application().RecordCustomEvent("searchString", map[string]interface{}{
			"searchQuery": searchQuery,
		})

		results, err := newsapi.FetchEverything(searchQuery, page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		nextPage, err := strconv.Atoi(page)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		search := &Search{
			Query:      searchQuery,
			NextPage:   nextPage,
			TotalPages: int(math.Ceil(float64(results.TotalResults) / float64(newsapi.PageSize))),
			Results:    results,
		}

		if ok := !search.IsLastPage(); ok {
			search.NextPage++
		}

		buf := &bytes.Buffer{}
		err = temp.Execute(buf, search)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = buf.WriteTo(w)
		if err != nil {
			return
		}

		fmt.Printf("%+v", results)
		fmt.Println("Search query is: ", searchQuery)
		fmt.Println("Page is: ", page)
		segment.End()
	}
}

func (s *Search) IsLastPage() bool {
	return s.NextPage >= s.TotalPages
}

func (s *Search) CurrentPage() int {
	if s.NextPage == 1 {
		return s.NextPage
	}
	return s.NextPage - 1
}

func (s *Search) PreviousPage() int {
	return s.CurrentPage() - 1
}

func main() {
	_err := godotenv.Load()
	if _err != nil {
		log.Println("Error loading .env file")
	}

	newRelicKey := os.Getenv("NEWRELIC_API_KEY")
	newRelicAppName := os.Getenv("NEWRELIC_APP_NAME")
	if newRelicAppName == "" {
		newRelicAppName = "goapp_news_demo"
		log.Println("Warning: New Relic app name (NEWRELIC_APP_NAME) not defined in environment. The app name was hardcoded for this instance. Please set in environment.")
	}
	if newRelicKey == "" {
		log.Fatalf("Error: New Relic API key (NEWRELIC_API_KEY) not found in environment. Please check and try again!")
	}

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(newRelicAppName),
		newrelic.ConfigLicense(newRelicKey),
		newrelic.ConfigDebugLogger(os.Stdout),
	)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	apiKey := os.Getenv("NEWS_API_KEY")
	if apiKey == "" {
		log.Fatalf("Env: API key (apiKey) must be set but is not. Please check and try again!")
	}

	myClient := &http.Client{
		Timeout: 10 * time.Second,
	}
	newsapi := news.NewClient(myClient, apiKey, 20)

	fs := http.FileServer(http.Dir("assets"))

	mux := http.NewServeMux()
	mux.Handle(newrelic.WrapHandle(app, "/assets/", http.StripPrefix("/assets/", fs)))
	mux.HandleFunc(newrelic.WrapHandleFunc(app, "/", indexHandler))
	mux.HandleFunc(newrelic.WrapHandleFunc(app, "/search", searchHandler(newsapi)))

	err = http.ListenAndServe(":"+port, mux)
	if err != nil {
		return
	}
}
