package main

import (
	"net"

	"github.com/morya/utils/log"

	"github.com/morya/utils/errors"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/morya/drone/util"

	crypto_ssh "golang.org/x/crypto/ssh"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

type Repo struct {
	repo       *git.Repository
	lastCommit *object.Commit
}

func getRepoLastCommit(r *git.Repository) (*object.Commit, error) {
	logs, _ := r.Log(&git.LogOptions{})
	var lastCommit *object.Commit
	logs.ForEach(func(commit *object.Commit) error {
		if lastCommit == nil {
			lastCommit = commit
			log.Debugf("last commit hex = %v", lastCommit.Hash.String())
			return errors.New("done")
		}
		log.Debugf("commit hash = %v", commit.Hash.String())
		return nil
	})

	return lastCommit, nil
}

func newRepo(repoURL string, repoPath string, keyFile string) (*Repo, error) {
	var exist bool
	if !util.Exists(repoPath) || !util.IsDir(repoPath) {
		util.RemoveDir(repoPath)
		exist = false
	} else {
		exist = true
	}

	var repo *git.Repository
	var err error
	if exist {
		repo, err = git.PlainOpen(repoPath)
	} else {
		// TODO 'git' 是git-server默认用户名，暂时不考虑支持其它用户名
		auth, err := ssh.NewPublicKeysFromFile("git", keyFile, "")
		if err != nil {
			return nil, err
		}
		auth.HostKeyCallback = func(hostname string, remote net.Addr, key crypto_ssh.PublicKey) error {
			// ignore host key
			return nil
		}

		repo, err = git.PlainClone(repoPath, false, &git.CloneOptions{
			URL:  repoURL,
			Auth: auth,
		})
	}

	if err != nil {
		return nil, err
	}
	lastCommit, err := getRepoLastCommit(repo)
	log.Debugf("last commit is %v, err = %v", lastCommit.Hash.String(), err)
	return &Repo{repo: repo, lastCommit: lastCommit}, nil
}

func (r *Repo) HasUpdate() (bool, error) {
	r.repo.Fetch(&git.FetchOptions{RemoteName: "origin"})

	tree, err := r.repo.Worktree()
	if err != nil {
		return false, err
	}

	err = tree.Pull(&git.PullOptions{RemoteName: "origin"})
	switch err {
	case git.NoErrAlreadyUpToDate:
		return false, nil

	case nil:
		return true, nil
	default:

	}
	return false, err
}
