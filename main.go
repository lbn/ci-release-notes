package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
)

var (
	repo  = flag.String("repo", "", "repository name")
	owner = flag.String("owner", "", "owner (org or user)")
	token = flag.String("token", "", "OAuth2 token")
	tag   = flag.String("tag", "", "this release tag")

	errNoRelease = errors.New("no release for this tag")
)

func createReleaseNotes(prs []*github.PullRequest) string {
	var lines []string
	for _, pr := range prs {
		lines = append(lines, fmt.Sprintf("[#%d](%s) - %s [%s]", *pr.Number, *pr.HTMLURL, *pr.Title, *pr.User.Login))
	}
	return strings.Join(lines, "\n")
}

func addReleaseNotes(client *github.Client, prs []*github.PullRequest) error {
	releases, _, err := client.Repositories.ListReleases(*owner, *repo, nil)
	if err != nil {
		log.Fatal(err)
	}
	var thisRelease *github.RepositoryRelease
	for _, release := range releases {
		if *release.TagName == *tag {
			thisRelease = release
		}
	}

	if thisRelease == nil {
		return errNoRelease
	}

	*thisRelease.Body = *thisRelease.Body + "\n" + createReleaseNotes(prs)
	log.Println(*thisRelease.Body)
	_, _, err = client.Repositories.EditRelease(*owner, *repo, *thisRelease.ID, thisRelease)
	if err != nil {
		return err
	}

	return nil
}

func main() {
	flag.Parse()

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token},
	)
	tc := oauth2.NewClient(oauth2.NoContext, ts)

	client := github.NewClient(tc)

	var prs []*github.PullRequest

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		txt := scanner.Text()
		if txt[0] == '#' {
			txt = txt[1:]
		}
		prNumber, err := strconv.Atoi(txt)
		if err != nil {
			log.Printf("Cannot parse PR number '%s'", txt)
			continue
		}
		pr, _, err := client.PullRequests.Get(*owner, *repo, prNumber)
		if err != nil {
			log.Printf("Cannot get pull request: %s", err.Error())
		}
		log.Println(*pr.Title)
		log.Println(*pr.User.Login)
		log.Println(*pr.URL)
		prs = append(prs, pr)
	}
	err := addReleaseNotes(client, prs)
	if err == errNoRelease {
		log.Println(err)
	} else if err != nil {
		log.Fatal(err)
	}
}