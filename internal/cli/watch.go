package cli

import (
	"os"
	"path/filepath"
	"time"

	storemod "github.com/rufus-SD/maind/internal/store"
)

type contextRefresher struct {
	store *storemod.Store
	paths []string
	stop  chan struct{}
}

func newContextRefresher(s *storemod.Store) *contextRefresher {
	return &contextRefresher{
		store: s,
		stop:  make(chan struct{}),
	}
}

func (r *contextRefresher) addProject(dir string) {
	r.paths = append(r.paths, dir)
}

func (r *contextRefresher) start() {
	go func() {
		r.refreshAll()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.stop:
				return
			case <-ticker.C:
				r.refreshAll()
			}
		}
	}()
}

func (r *contextRefresher) refreshAll() {
	for _, dir := range r.paths {
		refreshContextFile(r.store, dir)
	}
}

func (r *contextRefresher) cleanup() {
	for _, dir := range r.paths {
		os.Remove(filepath.Join(dir, ".maind", "context.md"))
	}
}

func (r *contextRefresher) close() {
	close(r.stop)
}

func findConnectedProjects() []string {
	var projects []string
	home, _ := os.UserHomeDir()

	common := []string{
		"Desktop", "Documents", "Projects", "dev", "src", "code", "repos",
	}

	for _, dir := range common {
		base := filepath.Join(home, dir)
		matches, _ := filepath.Glob(filepath.Join(base, "*", ".maind"))
		for _, m := range matches {
			projects = append(projects, filepath.Dir(m))
		}
		matches, _ = filepath.Glob(filepath.Join(base, "*", "*", ".maind"))
		for _, m := range matches {
			projects = append(projects, filepath.Dir(m))
		}
	}

	cwd, _ := os.Getwd()
	localMaind := filepath.Join(cwd, ".maind")
	if _, err := os.Stat(localMaind); err == nil {
		found := false
		for _, p := range projects {
			if p == cwd {
				found = true
				break
			}
		}
		if !found {
			projects = append(projects, cwd)
		}
	}

	return projects
}
