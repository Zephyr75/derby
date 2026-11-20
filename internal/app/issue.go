package app

// ISSUE.md: derby copies the task's GitHub issue, comments included, to the
// run directory, so every agent reads the same text without calling GitHub.
// It is copied when the task is created and before each run of the first agent.

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"derby/internal/artifacts"
	"derby/internal/github"
	"derby/internal/settings"
	"derby/internal/store"
)

const issueFile = "ISSUE.md"

// issueMarkdown renders an issue and its comments.
func issueMarkdown(d github.IssueDoc, repo string) string {
	var b strings.Builder
	var labels []string
	for _, l := range d.Labels {
		labels = append(labels, l.Name)
	}
	fmt.Fprintf(&b, "---\nissue: %d\nrepo: %s\nurl: %s\nstate: %s\nauthor: %s\nlabels: %s\ncreated: %s\n---\n\n",
		d.Number, repo, d.URL, strings.ToLower(d.State), d.Author.Login, strings.Join(labels, ", "), d.CreatedAt.Format("2006-01-02"))
	fmt.Fprintf(&b, "# %s\n\n", d.Title)
	if body := strings.TrimSpace(d.Body); body != "" {
		b.WriteString(body + "\n")
	} else {
		b.WriteString("(no description)\n")
	}
	if len(d.Comments) > 0 {
		b.WriteString("\n## Comments\n")
		for _, c := range d.Comments {
			fmt.Fprintf(&b, "\n### %s · %s\n\n%s\n", c.Author.Login, c.CreatedAt.Format("2006-01-02 15:04"), strings.TrimSpace(c.Body))
		}
	}
	return b.String()
}

// writeIssueFile copies the task's issue to ISSUE.md; a task without an
// issue has none.
func (a *App) writeIssueFile(p settings.Project, t store.Task) error {
	path := filepath.Join(a.runDir(p, t), issueFile)
	if t.Issue == "" {
		_ = os.Remove(path)
		return nil
	}
	d, err := a.gh.Issue(a.ctx, a.issueRepo(t), t.Issue)
	if err != nil {
		return err
	}
	return os.WriteFile(path, []byte(issueMarkdown(d, a.issueRepo(t))), 0o644)
}

// IssuePreview is an issue as New task shows it before there is a task:
// ISSUE.md's text, with its images loading through derby (see previewImages).
func (a *App) IssuePreview(projectID, issue string) (artifacts.Artifact, error) {
	p, ok := a.settings.Project(projectID)
	if !ok {
		return artifacts.Artifact{}, fmt.Errorf("unknown project %q", projectID)
	}
	repo := a.settings.Config().IssuesOf(p)
	if repo == "" {
		return artifacts.Artifact{}, errors.New("the project has no GitHub repo")
	}
	d, err := a.gh.Issue(a.ctx, repo, issue)
	if err != nil {
		return artifacts.Artifact{}, err
	}
	art := artifacts.Artifact{File: issueFile, Exists: true}
	art.Front, art.Body = artifacts.Parse(issueMarkdown(d, repo))
	art.Body = previewImages(art.Body)
	return art, nil
}

// TaskIssue is a task's issue as its description pop-up shows it: fetched
// again for the latest comments, else the copy in ISSUE.md (Body only).
type TaskIssue struct {
	Number   int            `json:"number"`
	Repo     string         `json:"repo"`
	Title    string         `json:"title"`
	URL      string         `json:"url"`
	State    string         `json:"state"`
	Author   string         `json:"author"`
	Created  string         `json:"created"`
	Labels   []string       `json:"labels"`
	Body     string         `json:"body"`
	Comments []IssueComment `json:"comments"`
	Offline  string         `json:"offline"` // why GitHub could not be read: Body is ISSUE.md
}

// IssueComment is a comment of a TaskIssue.
type IssueComment struct {
	Author  string `json:"author"`
	Created string `json:"created"`
	Body    string `json:"body"`
}

// GetTaskIssue returns a task's issue with its comments; images load
// through derby (see previewImages).
func (a *App) GetTaskIssue(taskID string) (TaskIssue, error) {
	t, err := a.db.Task(taskID)
	if err != nil {
		return TaskIssue{}, err
	}
	if t.Issue == "" {
		return TaskIssue{}, errors.New("this task has no GitHub issue")
	}
	repo := a.issueRepo(t)
	d, err := a.gh.Issue(a.ctx, repo, t.Issue)
	if err != nil {
		b, rerr := os.ReadFile(filepath.Join(a.runDir(a.projectOf(t), t), issueFile))
		if rerr != nil {
			return TaskIssue{}, err
		}
		_, body := artifacts.Parse(string(b))
		return TaskIssue{Repo: repo, Title: t.Title, Body: previewImages(body), Offline: err.Error(), Comments: []IssueComment{}}, nil
	}
	v := TaskIssue{
		Number: d.Number, Repo: repo, Title: d.Title, URL: d.URL, State: strings.ToLower(d.State), Author: d.Author.Login,
		Created: d.CreatedAt.Format("2006-01-02"), Body: previewImages(strings.TrimSpace(d.Body)), Labels: []string{}, Comments: []IssueComment{},
	}
	for _, l := range d.Labels {
		v.Labels = append(v.Labels, l.Name)
	}
	for _, c := range d.Comments {
		v.Comments = append(v.Comments, IssueComment{Author: c.Author.Login, Created: c.CreatedAt.Format("2006-01-02 15:04"), Body: previewImages(strings.TrimSpace(c.Body))})
	}
	return v, nil
}

var (
	imgTag     = regexp.MustCompile(`(?is)<img\b[^>]*>`)
	imgAttr    = regexp.MustCompile(`(?is)\b(src|alt)\s*=\s*"([^"]*)"`)
	attachment = regexp.MustCompile(`(!\[[^\]]*\]\()(https://github\.com/user-attachments/[^)\s]+)\)`)
)

// previewImages makes an issue's images show in derby: GitHub writes them as
// <img> tags, which derby's markdown leaves out, and a private repository's
// attachments only load signed in, so derby fetches them (ghFilePath). Other
// attachments stay links, opened in the browser where the user is signed in.
func previewImages(md string) string {
	md = imgTag.ReplaceAllStringFunc(md, func(tag string) string {
		src, alt := "", "image"
		for _, m := range imgAttr.FindAllStringSubmatch(tag, -1) {
			if strings.EqualFold(m[1], "src") {
				src = m[2]
			} else if m[2] != "" {
				alt = m[2]
			}
		}
		if src == "" {
			return tag
		}
		return fmt.Sprintf("![%s](%s)", alt, src)
	})
	return attachment.ReplaceAllStringFunc(md, func(m string) string {
		parts := attachment.FindStringSubmatch(m)
		return parts[1] + ghFilePath + "?u=" + url.QueryEscape(parts[2]) + ")"
	})
}

// ghFilePath serves a file attached to a GitHub issue (?u=<its URL>),
// fetched with the gh user's token.
const ghFilePath = "/gh-file"

func (a *App) serveGHFile(w http.ResponseWriter, r *http.Request) {
	u := r.URL.Query().Get("u")
	if r.Method != http.MethodGet || !strings.HasPrefix(u, "https://github.com/user-attachments/") {
		http.NotFound(w, r)
		return
	}
	token, err := a.gh.Token(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u, nil)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	// Go drops the header on the redirect to the file's storage host.
	req.Header.Set("Authorization", "token "+strings.TrimSpace(token))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		http.Error(w, res.Status, res.StatusCode)
		return
	}
	w.Header().Set("Content-Type", res.Header.Get("Content-Type"))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("Content-Security-Policy", "sandbox") // shown as an image, never run as a page
	w.Header().Set("X-Content-Type-Options", "nosniff")
	_, _ = io.Copy(w, res.Body)
}
