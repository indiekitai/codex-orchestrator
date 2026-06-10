package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestLedgerTaskLifecycle(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "repo")
	worker := filepath.Join(root, "worker")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, project, "init", "-q")
	git(t, project, "config", "user.email", "test@example.com")
	git(t, project, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, project, "add", "README.md")
	git(t, project, "commit", "-q", "-m", "base")

	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/api-auth", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "API-AUTH-LOCAL",
		"--title", "API auth local",
		"--worktree", worker,
		"--branch", "codex/api-auth",
		"--allowed", "services/api",
		"--forbidden", "secrets",
		"--gate", "go test ./...",
		"--evidence", "local",
	}); err != nil {
		t.Fatal(err)
	}

	f, err := os.OpenFile(filepath.Join(worker, "README.md"), os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("worker\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "README.md")
	git(t, worker, "commit", "-q", "-m", "worker change")

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.Observations) != 1 {
		t.Fatalf("expected one observation, got %d", len(summary.Observations))
	}
	if got := summary.Observations[0].Status; got != "completed-unreviewed" {
		t.Fatalf("expected completed-unreviewed, got %q", got)
	}
	if got := summary.OverallStatus; got != "review-needed" {
		t.Fatalf("expected review-needed, got %q", got)
	}
	if err := cmdAppendEvent([]string{
		"--ledger", ledger,
		"--type", "review",
		"--task-id", "API-AUTH-LOCAL",
		"--status", "completed-unreviewed",
		"--note", "ready for orchestrator review",
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if got := updated.Tasks[0].Status; got != "completed-unreviewed" {
		t.Fatalf("expected updated task status, got %q", got)
	}
	if len(updated.Tasks[0].History) != 2 {
		t.Fatalf("expected two history events, got %d", len(updated.Tasks[0].History))
	}
	if _, err := os.Stat(filepath.Join(project, ".codex-orchestrator", "events.jsonl")); err != nil {
		t.Fatal(err)
	}
}

func TestObserveClassifications(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(root, "missing")
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "MISSING", "--worktree", missing, "--branch", "codex/missing", "--base-commit", base}); err != nil {
		t.Fatal(err)
	}
	dirty := filepath.Join(root, "dirty")
	git(t, project, "worktree", "add", "-q", "-b", "codex/dirty", dirty, "HEAD")
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "DIRTY", "--worktree", dirty, "--branch", "codex/dirty", "--base-commit", base}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dirty, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	wrong := filepath.Join(root, "wrong")
	git(t, project, "worktree", "add", "-q", "-b", "codex/actual", wrong, "HEAD")
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "WRONG", "--worktree", wrong, "--branch", "codex/expected", "--base-commit", base}); err != nil {
		t.Fatal(err)
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	for _, item := range summary.Observations {
		statuses[item.ID] = item.Status
	}
	if statuses["MISSING"] != "pending-setup" {
		t.Fatalf("expected pending setup, got %q", statuses["MISSING"])
	}
	if statuses["DIRTY"] != "stale-needs-inspection" {
		t.Fatalf("expected stale dirty worktree, got %q", statuses["DIRTY"])
	}
	if statuses["WRONG"] != "blocked" {
		t.Fatalf("expected blocked branch mismatch, got %q", statuses["WRONG"])
	}
	if summary.OverallStatus != "blocked" {
		t.Fatalf("expected overall blocked, got %q", summary.OverallStatus)
	}
}

func TestHeartbeatWritesReportAndEvent(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	report := filepath.Join(project, ".codex-orchestrator", "heartbeat-report.json")
	summary := filepath.Join(project, ".codex-orchestrator", "heartbeat-summary.md")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdHeartbeat([]string{"--ledger", ledger, "--interval", "0", "--count", "1", "--write-report", report, "--write-summary", summary}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var observed ObserveSummary
	if err := json.Unmarshal(data, &observed); err != nil {
		t.Fatal(err)
	}
	if observed.OverallStatus != "dispatch-possible" {
		t.Fatalf("expected dispatch-possible, got %q", observed.OverallStatus)
	}
	text, err := os.ReadFile(summary)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(text), "overallStatus") {
		t.Fatalf("expected Markdown summary to include overallStatus")
	}
	events, err := os.ReadFile(filepath.Join(project, ".codex-orchestrator", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(events), `"type":"heartbeat"`) {
		t.Fatalf("expected heartbeat event, got %s", string(events))
	}
}

func createRepo(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, path, "init", "-q")
	git(t, path, "config", "user.email", "test@example.com")
	git(t, path, "config", "user.name", "Test")
	if err := os.WriteFile(filepath.Join(path, "README.md"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, path, "add", "README.md")
	git(t, path, "commit", "-q", "-m", "base")
	return path
}

func git(t *testing.T, cwd string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}

func gitOutputForTest(t *testing.T, cwd string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
	return strings.TrimSpace(string(out))
}
