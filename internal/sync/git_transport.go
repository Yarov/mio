package sync

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitTransport implements Transport using a git repository as remote storage.
// Chunks are stored as gzipped JSON files in a chunks/ subdirectory of the repo.
type GitTransport struct {
	repoDir string
	remote  string
	branch  string
}

// NewGitTransport creates a GitTransport. It clones or initializes the repo
// at ~/.mio/sync-repo/ and ensures the chunks/ directory exists.
func NewGitTransport(remote, branch string) (*GitTransport, error) {
	if branch == "" {
		branch = "main"
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}
	repoDir := filepath.Join(home, ".mio", "sync-repo")

	t := &GitTransport{
		repoDir: repoDir,
		remote:  remote,
		branch:  branch,
	}

	if err := t.ensureRepo(); err != nil {
		return nil, fmt.Errorf("ensure repo: %w", err)
	}

	return t, nil
}

func (t *GitTransport) ensureRepo() error {
	gitDir := filepath.Join(t.repoDir, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		// Repo exists, pull latest
		_ = t.git("pull", "--rebase", "origin", t.branch)
		return nil
	}

	if t.remote != "" {
		// Clone from remote
		if err := os.MkdirAll(filepath.Dir(t.repoDir), 0755); err != nil {
			return err
		}
		cmd := exec.Command("git", "clone", "-b", t.branch, t.remote, t.repoDir)
		if out, err := cmd.CombinedOutput(); err != nil {
			// If branch doesn't exist yet, clone default and create branch
			cmd2 := exec.Command("git", "clone", t.remote, t.repoDir)
			if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
				return fmt.Errorf("git clone: %s: %w", string(append(out, out2...)), err2)
			}
			_ = t.git("checkout", "-b", t.branch)
		}
	} else {
		// Init a new local repo
		if err := os.MkdirAll(t.repoDir, 0755); err != nil {
			return err
		}
		if err := t.git("init"); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
		_ = t.git("checkout", "-b", t.branch)
	}

	// Ensure chunks directory
	chunksDir := filepath.Join(t.repoDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return err
	}

	return nil
}

func (t *GitTransport) git(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = t.repoDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
	}
	return nil
}

func (t *GitTransport) Write(chunkID string, data []byte) error {
	chunksDir := filepath.Join(t.repoDir, "chunks")
	if err := os.MkdirAll(chunksDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(chunksDir, chunkID+".jsonl.gz")
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create chunk file: %w", err)
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	if _, err := gz.Write(data); err != nil {
		gz.Close()
		return fmt.Errorf("write compressed data: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("close gzip writer: %w", err)
	}

	// Stage, commit, push
	relPath := filepath.Join("chunks", chunkID+".jsonl.gz")
	if err := t.git("add", relPath); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	if err := t.git("commit", "-m", fmt.Sprintf("sync: add chunk %s", chunkID)); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}
	if t.remote != "" {
		if err := t.git("push", "origin", t.branch); err != nil {
			return fmt.Errorf("git push: %w", err)
		}
	}

	return nil
}

func (t *GitTransport) Read(chunkID string) ([]byte, error) {
	// Pull latest before reading
	if t.remote != "" {
		_ = t.git("pull", "--rebase", "origin", t.branch)
	}

	path := filepath.Join(t.repoDir, "chunks", chunkID+".jsonl.gz")
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open chunk file: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gz.Close()

	return io.ReadAll(gz)
}

func (t *GitTransport) List() ([]string, error) {
	// Pull latest before listing
	if t.remote != "" {
		_ = t.git("pull", "--rebase", "origin", t.branch)
	}

	chunksDir := filepath.Join(t.repoDir, "chunks")
	entries, err := os.ReadDir(chunksDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".jsonl.gz") {
			name := e.Name()
			id := name[:len(name)-len(".jsonl.gz")]
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (t *GitTransport) Exists(chunkID string) (bool, error) {
	path := filepath.Join(t.repoDir, "chunks", chunkID+".jsonl.gz")
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	return err == nil, err
}

