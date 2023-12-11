package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
	"golang.org/x/oauth2"
)

const (
	githubToken    = "YOUR_GITHUB_TOKEN"
	host           = "localhost"
	port           = "19530"
	collectionName = "github_issues"
	dimension      = 300 // Adjust based on the dimensionality of your vectors
	issuesDaysBack = 90
)

func main() {
	// Connect to Milvus
	ctx := context.Background()
	cli, err := client.NewGrpcClient(ctx, host, port)
	if err != nil {
		log.Fatal(err)
	}
	defer cli.Close(ctx)

	// Authenticate with GitHub
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// Specify the repositories
	repos := []string{"facebook/react", "angular/angular", "tensorflow/tensorflow"}

	// Fetch and store GitHub issues for each repository
	for _, repo := range repos {
		// Fetch GitHub issues
		issues, err := fetchGitHubIssues(client, repo, issuesDaysBack)
		if err != nil {
			log.Printf("Error fetching issues for %s: %v\n", repo, err)
			continue
		}

		// Create a collection if it does not exist
		collectionParam := entity.NewCollectionParam(collectionName, dimension)
		collection, err := cli.HasCollection(ctx, collectionName)
		if err != nil {
			log.Fatal(err)
		}
		if !collection {
			err := cli.CreateCollection(ctx, collectionName, collectionParam)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Insert vectors into Milvus
		for _, issue := range issues {
			vector := getVectorFromIssue(issue)
			_, err := cli.Insert(ctx, collectionName, [][]float32{vector})
			if err != nil {
				log.Printf("Error inserting vector for %s: %v\n", repo, err)
				continue
			}
		}

		fmt.Printf("Vectors for %s inserted successfully!\n", repo)
	}
}

func fetchGitHubIssues(client *github.Client, repo string, daysBack int) ([]*github.Issue, error) {
	// Calculate the date 90 days ago
	since := time.Now().AddDate(0, 0, -daysBack)

	// Fetch GitHub issues
	issues, _, err := client.Issues.ListByRepo(context.Background(), ownerFromRepo(repo), repoNameFromRepo(repo), &github.IssueListByRepoOptions{
		Since: &since,
	})
	if err != nil {
		return nil, err
	}

	return issues, nil
}

func ownerFromRepo(repo string) string {
	return repo[:strings.Index(repo, "/")]
}

func repoNameFromRepo(repo string) string {
	return repo[strings.Index(repo, "/")+1:]
}

func getVectorFromIssue(issue *github.Issue) []float32 {
	// Implement a function to convert the issue text into vectors
	// This can involve using pre-trained word embeddings or other techniques
	// Here, we're using a placeholder implementation for illustration
	return make([]float32, dimension)
}
