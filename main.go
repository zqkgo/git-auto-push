package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type Repository struct {
	Path   string `json:"path"`
	Remote string `json:"remote"`
	Branch string `json: "branch"`
}

type Config struct {
	Repositories []Repository `json:"repositories"`
	Interval     int          `json:"interval"`
}

func main() {
	f, err := os.OpenFile("config.json", os.O_RDONLY, 0766)
	if err != nil {
		log.Fatalf("failed to open config file, err: %+v\n", err)
	}
	defer f.Close()

	bs, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatalf("failed to read config file, err: %+v\n", err)
	}

	var c Config
	err = json.Unmarshal(bs, &c)
	if err != nil {
		log.Fatalf("failed to decode config file, err: %+v\n", err)
	}
	itvl, repos := c.Interval, c.Repositories
	if itvl == 0 {
		itvl = 10
	}
	for {
		autoPush(repos)
		time.Sleep(time.Duration(itvl) * time.Second)
	}
}

func autoPush(repos []Repository) {
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
			log.Printf("ERROR: failed to get directory stat, err: %+v, path: %s\n", err, repo.Path)
			continue
		}
		if !s.IsDir() {
			log.Printf("ERROR: %s is not a directory\n", repo.Path)
			continue
		}
		os.Chdir(p)

		if err != nil {
			log.Printf("ERROR: failed to change working dir, err: %+v, path: %s\n", err, repo.Path)
			continue
		}

		cmd := exec.Command("git", "add", ".")
		bs, err := cmd.Output()
		if err != nil {
			log.Printf("ERROR: failed to run 'git add', err: %+v, path: %s, output: %s\n", err, repo.Path, string(bs))
			continue
		}
		curTime := time.Now().Format("2006/01/02 15:04:05")
		cmd = exec.Command("git", "commit", "-m", fmt.Sprintf("%s auto commit", curTime))
		bs, err = cmd.Output()
		if err != nil {
			log.Printf("ERROR: failed to run 'git commit', err: %+v, path: %s, output: %s\n", err, repo.Path, string(bs))
			continue
		}

		cmd = exec.Command("git", "push", repo.Remote, repo.Branch)
		bs, err = cmd.Output()
		if err != nil {
			log.Printf("ERROR: failed to run 'git push', err: %+v, path: %s, output: %s\n", err, repo.Path, string(bs))
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
	fmt.Printf("Successfully pushed repositories: \n %s", s)
}
