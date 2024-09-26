package main

import (
	"encoding/json"
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
	var success []string
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
		err = os.Chdir(p)
		if err != nil {
			log.Printf("WARN: failed to change working dir, err: %+v, path: %s", err, repo.Path)
			continue
		}

		ok := syncGit(repo)
		if !ok {
			log.Printf("repo %+v not synced", repo)
			continue
		}

		success = append(success, repo.Path)
	}
	os.Chdir(oriDir)
	if len(success) == 0 {
		fmt.Println("No repository pushed")
		return
	}
	s := strings.Join(success, "\n")
	fmt.Printf("Successfully pushed repositories: %s\n", s)
}

var notStaged = "Changes not staged for commit"

func syncGit(repo Repository) (ok bool) {
	var err error
	defer func() {
		if err == nil {
			ok = true
		}
	}()

	cmd := exec.Command("git", "status")
	bs, err := cmd.Output()
	if err != nil {
		log.Printf("WARN: failed to run 'git status', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
		return
	}

	// check if there are files need to be staged.
	var stashed bool
	if strings.Contains(string(bs), notStaged) {
		cmd = exec.Command("git", "stash")
		bs, err = cmd.Output()
		if err != nil {
			log.Printf("WARN: failed to run 'git stash', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
			return
		}
		stashed = true
	}

	cmd = exec.Command("git", "pull", repo.Remote, repo.Branch)
	bs, err = cmd.Output()
	if err != nil {
		log.Printf("WARN: failed to run 'git stash', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
		return
	}

	// push local updated files.
	if stashed {
		cmd = exec.Command("git", "stash", "pop")
		bs, err = cmd.Output()
		if err != nil {
			log.Printf("WARN: failed to run 'git stash', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
			return
		}
		cmd = exec.Command("git", "add", ".")
		bs, err = cmd.Output()
		if err != nil {
			log.Printf("WARN: failed to run 'git add', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
			return
		}

		curTime := time.Now().Format("2006/01/02 15:04:05")
		cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("%s auto commit", curTime))
		bs, err = cmd.Output()
		if err != nil {
			log.Printf("WARN: failed to run 'git commit', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
			return
		}

		cmd = exec.Command("git", "push", repo.Remote, repo.Branch)
		bs, err = cmd.Output()
		if err != nil {
			log.Printf("WARN: failed to run 'git push', err: %+v, path: %s, output: %s", err, repo.Path, string(bs))
			return
		}
	}
	return
}
