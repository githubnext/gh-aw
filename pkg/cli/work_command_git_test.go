package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workqueue"
)

type workQueueGitRemote struct {
	git, dir, pushLog string
}

// Redirect only the fixture repository to a disposable local remote. The CLI
// still uses its real isolated Git transport; REST mocks must not serve ledgers.
func newWorkQueueGitRemote(t *testing.T, ledger string) *workQueueGitRemote {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	remote := &workQueueGitRemote{git: git, dir: filepath.Join(root, "remote.git"), pushLog: filepath.Join(root, "pushes")}
	remote.run(t, "", "init", "--bare", "--initial-branch=main", remote.dir)
	tree := strings.TrimSpace(remote.run(t, "", "mktree"))
	head := strings.TrimSpace(remote.run(t, "Default branch fixture\n", "commit-tree", tree))
	remote.run(t, "", "update-ref", "refs/heads/main", head)
	if ledger != "" {
		remote.write(t, ledger)
	}
	script := `#!/bin/sh
for arg do
  if [ "$arg" = push ]; then
    printf 'push\n' >> "$WORK_QUEUE_TEST_PUSH_LOG"
  fi
  if [ "$arg" = ls-remote ] && [ "${WORK_QUEUE_TEST_INVALID_REF:-}" = true ]; then
    printf '1\trefs/heads/work-queue\n'
    exit 0
  fi
done
exec "$WORK_QUEUE_TEST_GIT" -c "url.$WORK_QUEUE_TEST_REMOTE.insteadOf=https://github.com/owner/repo.git" -c protocol.file.allow=always "$@"
`
	bin := filepath.Join(root, "bin")
	if err := os.Mkdir(bin, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bin, "git"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("WORK_QUEUE_TEST_GIT", git)
	t.Setenv("WORK_QUEUE_TEST_REMOTE", remote.dir)
	t.Setenv("WORK_QUEUE_TEST_PUSH_LOG", remote.pushLog)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	return remote
}

func (r *workQueueGitRemote) run(t *testing.T, input string, args ...string) string {
	t.Helper()
	cmd := exec.Command(r.git, append([]string{"--git-dir=" + r.dir, "-c", "user.name=Queue Test", "-c", "user.email=queue@example.com", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Stdin = strings.NewReader(input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fixture git %s failed: %v (%s)", strings.Join(args, " "), err, output)
	}
	return string(output)
}

func (r *workQueueGitRemote) head(t *testing.T) string {
	t.Helper()
	refs := r.run(t, "", "for-each-ref", "--format=%(objectname)", "refs/heads/"+workqueue.DefaultBranch)
	return strings.TrimSpace(refs)
}

func (r *workQueueGitRemote) read(t *testing.T) string {
	t.Helper()
	if r.head(t) == "" {
		return ""
	}
	return r.run(t, "", "show", "refs/heads/"+workqueue.DefaultBranch+":"+workqueue.FileName)
}

func (r *workQueueGitRemote) write(t *testing.T, ledger string) {
	t.Helper()
	blob := strings.TrimSpace(r.run(t, ledger, "hash-object", "-w", "--stdin"))
	tree := strings.TrimSpace(r.run(t, fmt.Sprintf("100644 blob %s\t%s\n", blob, workqueue.FileName), "mktree"))
	args := []string{"commit-tree", tree}
	if head := r.head(t); head != "" {
		args = append(args, "-p", head)
	}
	head := strings.TrimSpace(r.run(t, "Queue fixture\n", args...))
	r.run(t, "", "update-ref", "refs/heads/"+workqueue.DefaultBranch, head)
}

func (r *workQueueGitRemote) pushes(t *testing.T) int {
	t.Helper()
	data, err := os.ReadFile(r.pushLog)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return strings.Count(string(data), "push\n")
}
