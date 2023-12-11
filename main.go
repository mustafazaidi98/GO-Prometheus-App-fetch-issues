package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	host     = "34.123.89.96"
	port     = 5432
	user     = "postgres"
	password = "28899"
	dbname   = "Assignment5"
)

var (
	githubAPICalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "github_api_calls_total",
			Help: "Total number of GitHub API calls.",
		},
		[]string{"repository"},
	)

	stackOverflowAPICalls = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "stackoverflow_api_calls_total",
			Help: "Total number of StackOverflow API calls.",
		},
		[]string{"tag"},
	)

	dataCollected = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "data_collected_total",
			Help: "Total amount of data collected.",
		},
		[]string{"source"},
	)

	totalDataSize float64
)

type gitIssue struct {
	Title string `json:"title"`
	Link  string `json:"html_url"`
	Size  float64
}

type StackOverflowItem struct {
	Items []struct {
		QuestionID int    `json:"question_id"`
		Title      string `json:"title"`
		IsAnswered bool   `json:"is_answered"`
		Answers    []struct {
			AnswerID   int  `json:"answer_id"`
			IsAccepted bool `json:"is_accepted"`
		} `json:"answers"`
	} `json:"items"`
}

func main() {
	prometheus.MustRegister(githubAPICalls)
	prometheus.MustRegister(stackOverflowAPICalls)
	prometheus.MustRegister(dataCollected)
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":8081", nil)
	}()
	go func() {
		for {
			fetchAndProcessData()
			time.Sleep(1000 * time.Microsecond)
		}
	}()
	select {}
}

func fetchAndProcessData() {
	token := "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"

	urls := []string{
		"https://api.github.com/repos/golang/go/issues",
		"https://api.github.com/repos/prometheus/prometheus/issues",
		"https://api.github.com/repos/seleniumhq/selenium/issues",
		"https://api.github.com/repos/openai/openai-python/issues",
		"https://api.github.com/repos/docker/docs/issues",
		"https://api.github.com/repos/milvus-io/milvus/issues",
	}

	stackoverflowLinks := []string{
		"https://api.stackexchange.com/2.3/search/advanced?site=stackoverflow&tagged=go",
		"https://api.stackexchange.com/2.3/search/advanced?site=stackoverflow&tagged=prometheus",
		"https://api.stackexchange.com/2.3/search/advanced?site=stackoverflow&tagged=selenium",
		"https://api.stackexchange.com/2.3/search/advanced?site=stackoverflow&tagged=openai",
		"https://api.stackexchange.com/2.3/search/advanced?site=stackoverflow&tagged=docker",
		"https://api.stackexchange.com/2.3/search/advanced?site=stackoverflow&tagged=milvus",
	}

	connStr := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		return
	}
	defer db.Close()

	for _, url := range urls {
		githubAPICalls.WithLabelValues(repositoryName(url)).Inc()

		issues, err := fetchGitHubIssues(url, token)
		if err != nil {
			fmt.Println("Error fetching GitHub issues:", err)
			continue
		}
		const githubIssueSize = 0.1
		for i := range issues {
			issues[i].Size = githubIssueSize
		}
		for _, issue := range issues {
			totalDataSize += issue.Size
		}

		err = insertIntoDatabase(db, issues)
		if err != nil {
			fmt.Println("Error inserting into the database:", err)
			continue
		}
		dataCollected.WithLabelValues("github").Add(totalDataSize)
	}

	for _, link := range stackoverflowLinks {
		stackOverflowAPICalls.WithLabelValues(tagName(link)).Inc()
		fmt.Println(link)
		item, err := fetchDataFromStackOverflow(link)
		if err != nil {
			log.Println("Error fetching data from Stack Overflow:", err)
			continue
		}
		for _, i := range item.Items {
			fmt.Sprintf("Question ID: %d\nTitle: %s\nIs Answered: %t\n", i.QuestionID, i.Title, i.IsAnswered)
			err := insertDataStackOverFLow(db, i.QuestionID, i.Title, i.IsAnswered)
			if err != nil {
				fmt.Println("Error inserting into the database:", err)
			}
		}
		dataCollected.WithLabelValues("stackoverflow").Add(totalDataSize)
	}
}
func repositoryName(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 4 {
		return parts[4] + "/" + parts[5]
	}
	return "unknown"
}
func tagName(link string) string {
	parts := strings.Split(link, "=")
	if len(parts) >= 2 {
		return parts[2]
	}
	return "unknown"
}

func fetchGitHubIssues(url, token string) ([]gitIssue, error) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var issues []gitIssue
	err = json.Unmarshal(body, &issues)
	if err != nil {
		return nil, err
	}

	return issues, nil
}

func fetchDataFromStackOverflow(link string) (*StackOverflowItem, error) {
	req, err := http.NewRequest("GET", link+"", nil)
	if err != nil {
		return nil, err
	}
	q := req.URL.Query()
	q.Add("client_id", "Krf9*ha)ZXEeQmLlVEgubQ((")
	req.URL.RawQuery = q.Encode()

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var item StackOverflowItem
	err = json.Unmarshal(body, &item)
	fmt.Println(item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func insertIntoDatabase(db *sql.DB, issues []gitIssue) error {
	for _, issue := range issues {
		_, err := db.Exec(`INSERT INTO public."GitIssues"("Title", "url") VALUES ($1, $2)`, issue.Title, issue.Link)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertDataStackOverFLow(db *sql.DB, questionID int, title string, isAnswered bool) error {
	sqlStatement := `
		INSERT INTO public."StackOverFlow"("QuestionID", "Title", "isAnswered")
		VALUES ($1, $2, $3)`
	_, err := db.Exec(sqlStatement, questionID, title, isAnswered)
	return err
}
