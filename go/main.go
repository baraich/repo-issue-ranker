package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type PullRequest struct{}

type Issue struct {
	Number      int          `json:"number"`
	Title       string       `json:"title"`
	PullRequest *PullRequest `json:"pull_request,omitempty"`
}

type IssueReaction struct {
	Content string `json:"content"`
}

func sendRequest(url string) (*http.Response, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		log.Fatal("Missing GITHUB_TOKEN! Make sure you have configured it.")
	}

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+githubToken)
	req.Header.Set("User-Agent", "cli-learning-go")
	req.Header.Set("Accept", "application/vnd.github+json")

	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func getApiUrl(owner, repository string) string {
	return fmt.Sprintf("https://api.github.com/repos/%s/%s/issues?state=open", owner, repository)
}

func getIssues(owner, repository string) []Issue {
	url := getApiUrl(owner, repository)
	response, error := sendRequest(url)
	if error != nil || response.StatusCode != http.StatusOK {
		fmt.Printf("Exited with HTTP status code: %d\n", response.StatusCode)

		if response.StatusCode == http.StatusForbidden || response.StatusCode == http.StatusTooManyRequests {
			resetTime := response.Header.Get("x-ratelimit-reset")
			if timestamp, err := strconv.ParseInt(resetTime, 10, 64); err == nil {
				waitDuration := time.Unix(timestamp, 0).Sub(time.Now())
				fmt.Printf("Please try again later after %v!\n", waitDuration)
			} else {
				fmt.Println("Please try agian later in sometime!")
			}
		}
	}
	defer response.Body.Close()

	var issues []Issue
	if err := json.NewDecoder(response.Body).Decode(&issues); err != nil {
		log.Fatal(err)
	}

	valid_issues := make([]Issue, 0)
	for _, issue := range issues {
		if issue.PullRequest == nil {
			valid_issues = append(valid_issues, issue)
		}
	}

	return valid_issues
}

func getReactions(issueNumber int, owner, repository string) []IssueReaction {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/reactions", owner, repository, issueNumber)
	resp, err := sendRequest(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	var reactions []IssueReaction
	if err := json.NewDecoder(resp.Body).Decode(&reactions); err != nil {
		log.Fatal(err)
	}

	return reactions
}

func main() {
	// Load the environment variables into the program.
	godotenv.Load()

	// Defining repository and it's owner
	owner := "facebook"
	repository := "react"

	// A map that would store all the reaction for individual issue.
	reactionMap := make(map[int]int)

	// Fetching the issues from the Github API.
	issues := getIssues(owner, repository)
	fmt.Printf("Fetched %d issues!\n", len(issues))

	for _, issue := range issues {
		fmt.Printf("Gathering reactions for issue: %d\n", issue.Number)
		reactions := getReactions(issue.Number, owner, repository)

		for _, reaction := range reactions {
			switch reaction.Content {
			case "+1":
				reactionMap[issue.Number]++
			case "-1":
				reactionMap[issue.Number]--
			}
		}
	}

	type issueScore struct {
		Number int
		Score  int
	}

	var issueScores []issueScore
	for number, score := range reactionMap {
		issueScores = append(issueScores, issueScore{Number: number, Score: score})
	}

	sort.Slice(issueScores, func(i, j int) bool {
		return issueScores[i].Score > issueScores[j].Score
	})

	fmt.Println()
	for i, issue := range issueScores {
		fmt.Printf("#%d – Issue #%d with %d upvotes!\n", i+1, issue.Number, issue.Score)
	}
}
