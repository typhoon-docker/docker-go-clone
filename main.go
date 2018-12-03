package main

import (
	"github.com/labstack/echo"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type hook interface {
	Ref() string
	CloneUrl(token string) string
	User() string
}

type githubHook struct {
	GitRef string `json:"ref"`
	Repository struct {
		CloneUrl string `json:"clone_url"`
		Owner struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
}

func (s *githubHook) Ref() string {
	return s.GitRef
}

func (s *githubHook) CloneUrl(token string) string {
	i := strings.Index(s.Repository.CloneUrl, "//")
	if i == -1 {
		return s.Repository.CloneUrl
	}
	i += len("//")
	return s.Repository.CloneUrl[:i] + token + "@" + s.Repository.CloneUrl[i:]
}

func (s *githubHook) User() string {
	return s.Repository.Owner.Login
}

type gitlabHook struct {
	GitRef string `json:"ref"`
	Repository struct {
		GitHttpUrl string `json:"git_http_url"`
	} `json:"repository"`
	Project struct {
		Namespace string `json:"namespace"`
	} `json:"project"`
}

func (s *gitlabHook) Ref() string {
	return s.GitRef
}

func (s *gitlabHook) CloneUrl(token string) string {
	i := strings.Index(s.Repository.GitHttpUrl, "//")
	if i == -1 {
		return s.Repository.GitHttpUrl
	}
	i += len("//")
	return s.Repository.GitHttpUrl[:i] + "oauth2:" + token + "@" + s.Repository.GitHttpUrl[i:]
}

func (s *gitlabHook) User() string {
	return s.Project.Namespace
}

func getToken(user string) string {
	return "###tokentodo###"
}

func main() {
	e := echo.New()
	e.POST("/hook", func(c echo.Context) error {
		func() {
			var hook hook
			if c.Request().Header.Get("X-GitHub-Event") == "push" {
				var h githubHook
				if err := c.Bind(&h); err != nil {
					e.Logger.Warn(err)
					return
				}
				hook = &h
			} else if c.Request().Header.Get("X-Gitlab-Event") == "Push Hook" {
				var h gitlabHook
				if err := c.Bind(&h); err != nil {
					e.Logger.Warn(err)
					return
				}
				hook = &h
			} else {
				return
			}
			if hook.Ref() != "refs/heads/master" {
				return
			}
			token := getToken(hook.User())
			cmdGit := exec.Command("git", "clone", "-q", "--depth", "1", "--", hook.CloneUrl(token), "repo")
			cmdGit.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
			if err := cmdGit.Run(); err != nil {
				log.Fatal(err)
			}
		}()
		return c.String(http.StatusOK, "" )
	})
	e.Logger.Fatal(e.Start(":80"))
}
