package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Repository struct {
	// Absolute path of local directory.
	Path string `json:"path"`
	// Name of remote repository which mostly is origin.
	Remote string `json:"remote"`
	// Name of remote target branch.
	Branch string `json:"branch"`

	// Absolute path of local files.
	Files []string
}

type Config struct {
	Repositories []Repository `json:"repositories"`
	IntervalMs   int          `json:"interval_ms"`
}

var (
	dfgConf = "config.json"
	conf    = flag.String("conf", dfgConf, "-conf=/path/to/config.json")
)

func init() {
	log.SetOutput(os.Stdout)
}

func main() {
	flag.Parse()
	if *conf == "" {
		*conf = dfgConf
	}

	c, err := parseConfig(*conf)
	if err != nil {
		log.Fatalf("failed to parse config, err: %v", err)
		return
	}
	itvl, repos := c.IntervalMs, c.Repositories
	if itvl == 0 {
		itvl = 10 * 1000
	}
	for {
		autoSync(repos)
		time.Sleep(time.Duration(itvl) * time.Millisecond)
		log.Println("start to sync")
	}
}

func parseConfig(path string) (*Config, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0766)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file, err: %+v", err)
	}
	defer f.Close()

	bs, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file, err: %+v", err)
	}

	var c Config
	err = json.Unmarshal(bs, &c)
	if err != nil {
		return nil, fmt.Errorf("failed to decode config file, err: %+v", err)
	}

	return &c, nil
}

func autoSync(repos []Repository) {
	ex, err := os.Executable()
	if err != nil {
		log.Fatalf("failed to find current working dir, err: %+v\n", err)
	}
	oriDir := filepath.Dir(ex)
	var finished []string
	for i := 0; i < len(repos); i++ {
		repo := repos[i]
		p := repo.Path
		if len(p) == 0 {
			log.Println("WARNING: encounter an empty path")
			continue
		}

		s, err := os.Stat(p)
		if err != nil {
			log.Printf("WARN: failed to get directory stat, err: %+v, path: %s", err, repo.Path)
			continue
		}
		if !s.IsDir() {
			log.Printf("WARN: %s is not a directors", repo.Path)
			continue
		}
		log.Printf("%s is dir", p)
		err = os.Chdir(p)
		if err != nil {
			log.Printf("WARN: failed to change working dir, err: %+v, path: %s", err, repo.Path)
			continue
		}
		log.Printf("cwd changed to %s", p)

		ok := syncGit(repo)
		if !ok {
			log.Printf("repo %+v not synced", repo)
			continue
		}
		log.Printf("%+v synced to git", repo)

		finished = append(finished, repo.Path)
	}

	os.Chdir(oriDir)

	if len(finished) == 0 {
		log.Println("No repository pushed")
		return
	}

	s := strings.Join(finished, "\n")
	log.Printf("finish syncing repositories: %s\n", s)
}

var (
	notStaged = "Changes not staged for commit"
	untracked = "Untracked files"
)

func syncGit(repo Repository) bool {
	cmd := exec.Command("git", "pull", repo.Remote, repo.Branch)
	log.Println("run git pull and wait for output")

	// For some reason, the process may stuck when execute git pull,
	// here we run git pull in separate goroutine and wait for a max timeout.
	pullDone := make(chan struct{}, 1)
	go func() {
		bs, err := cmd.Output()
		if err != nil {
			log.Printf("WARN: failed to run 'git pull', err: %+v, path: %s", err, repo.Path)
		}
		log.Printf("git pull: %s", string(bs))
		pullDone <- struct{}{}
	}()

	maxWait := 30 * time.Second
	waitDone := time.After(maxWait)
	select {
	case <-pullDone:
		log.Print("finish git pull")
	case <-waitDone:
		err := cmd.Process.Kill()
		if err != nil {
			log.Printf("kill git process after %v waiting for pulling, err: %s", maxWait, err.Error())
		}
		return false
	}

	cmd = exec.Command("git", "status")
	bs, err := cmd.Output()
	if err != nil {
		log.Printf("WARN: failed to run 'git status', err: %+v, path: %s", err, repo.Path)
		return false
	}
	log.Printf("git status: %s", string(bs))
	if !needCommit(string(bs)) {
		err = errors.New("no change")
		log.Println(err)
		return false
	}

	cmd = exec.Command("git", "add", ".")
	bs, err = cmd.Output()
	if err != nil {
		log.Printf("WARN: failed to run 'git add', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
		return false
	}

	curTime := time.Now().Format("2006/01/02 15:04:05")
	cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("%s auto commit", curTime))
	bs, err = cmd.Output()
	if err != nil {
		log.Printf("WARN: failed to run 'git commit', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
		return false
	}

	cmd = exec.Command("git", "push", repo.Remote, repo.Branch)
	bs, err = cmd.Output()
	if err != nil {
		log.Printf("WARN: failed to run 'git push', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
		return false
	}
	log.Printf("git push: %s", string(bs))

	return true
}

func needCommit(msg string) bool {
	return strings.Contains(msg, notStaged) ||
		strings.Contains(msg, untracked)
}
