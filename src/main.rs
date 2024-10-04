use std::collections::HashMap;

use dotenv::dotenv;
use reqwest::header::{ACCEPT, AUTHORIZATION, USER_AGENT};
use serde::{Deserialize, Serialize};

#[derive(Debug, Serialize, Deserialize)]
struct PullRequest {}

#[derive(Debug, Serialize, Deserialize)]
struct Issue {
    number: usize,
    title: String,
    pull_request: Option<PullRequest>,
}

#[derive(Debug, Serialize, Deserialize)]
struct IssueReaction {
    content: String,
}

fn get_api_url(owner: &str, repo: &str) -> String {
    format!("https://api.github.com/repos/{owner}/{repo}/issues?state=open&per_page=100")
}

async fn send_request(url: &str) -> std::result::Result<reqwest::Response, reqwest::Error> {
    let github_personal_access_token = std::env::var("GITHUB_TOKEN").expect("Missing GIHHUB_TOKEN");
    let client = reqwest::Client::new();

    client
        .get(url)
        .header(AUTHORIZATION, github_personal_access_token)
        .header(USER_AGENT, "cli-learning-rust")
        .header(ACCEPT, "application/vnd.github+json")
        .send()
        .await
}

async fn get_issues(owner: &str, repo: &str) -> Vec<Issue> {
    let url = get_api_url(owner, repo);

    let response = send_request(&url).await;
    let response = match response {
        Ok(res) if res.status().is_success() => res,
        _ => return Vec::new(),
    };

    response
        .json::<Vec<Issue>>()
        .await
        .expect("Something went wrong while parsing.")
        .into_iter()
        .filter(|issue| issue.pull_request.is_none())
        .collect::<Vec<_>>()
}

async fn get_reactions(issue: &Issue, owner: &str, repo: &str) -> Vec<IssueReaction> {
    let request_url = format!(
        "https://api.github.com/repos/{owner}/{repo}/issues/{issue_number}/reactions",
        issue_number = issue.number
    );

    let response = send_request(&request_url).await;
    let response = match response {
        Ok(res) if res.status().is_success() => res,
        _ => return Vec::new(),
    };

    response
        .json::<Vec<IssueReaction>>()
        .await
        .expect("Something went wrong while parsting issue reactions.")
        .into_iter()
        .collect::<Vec<_>>()
}

#[tokio::main]
async fn main() {
    // Loading the environment variables in program.
    dotenv().ok();

    let owner: &str = "facebook";
    let repo: &str = "react";
    let mut map = HashMap::new();

    let issues = get_issues(owner, repo).await;
    println!("Fetched {:?} issues!", issues.len());

    for issue in &issues {
        println!("Gathering reactions for issue: {:?}", issue.number);
        let reactions = get_reactions(issue, owner, repo).await;

        for reaction in &reactions {
            let content: &str = &reaction.content.clone();

            let count = map.entry(issue.number).or_insert(0);
            *count += {
                match content {
                    "+1" => 1,
                    "-1" => -1,
                    _ => 0,
                }
            };
        }
    }

    let mut map_vec: Vec<(&usize, &i32)> = map.iter().collect();
    map_vec.sort_by(|a, b| b.1.cmp(a.1));

    println!();

    let mut position: u32 = 0;
    for issue in &map_vec {
        position += 1;
        println!(
            "#{:?} – {:?} with {:?} upvotes!",
            position, issue.0, issue.1
        );
    }
}
