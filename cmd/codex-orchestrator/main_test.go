package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if summary.ReviewPressure.Blocked != 1 {
		t.Fatalf("expected one blocked task, got %#v", summary.ReviewPressure)
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

func TestIntegrationDirtyBlocksDispatch(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "unrelated.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OverallStatus != "blocked" {
		t.Fatalf("expected blocked dirty integration checkout, got %q", summary.OverallStatus)
	}
	if !summary.Integration.Dirty {
		t.Fatalf("expected integration dirty")
	}
}

func TestStaleTimeoutClassification(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/stale", worker, "HEAD")
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "STALE", "--worktree", worker, "--branch", "codex/stale"}); err != nil {
		t.Fatal(err)
	}
	ledgerData, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	ledgerData.Tasks[0].LastObservation["at"] = time.Now().Add(-30 * time.Minute).Format(time.RFC3339)
	if err := saveLedger(ledger, &ledgerData); err != nil {
		t.Fatal(err)
	}
	summary, err := observeWithOptions(ledger, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if got := summary.Observations[0].Status; got != "stale-needs-inspection" {
		t.Fatalf("expected stale task, got %q", got)
	}
	if summary.OverallStatus != "stale" {
		t.Fatalf("expected overall stale, got %q", summary.OverallStatus)
	}
}

func TestTerminalMergedTaskRequiresCleanupWhenWorktreeRemains(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/merged", worker, "HEAD")
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "MERGED", "--worktree", worker, "--branch", "codex/merged"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "merge", "--task-id", "MERGED", "--status", "merged", "--note", "merged but not cleaned"}); err != nil {
		t.Fatal(err)
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if got := summary.Observations[0].Status; got != "cleanup-needed" {
		t.Fatalf("expected cleanup-needed, got %q", got)
	}
	if summary.OverallStatus != "cleanup-needed" {
		t.Fatalf("expected overall cleanup-needed, got %q", summary.OverallStatus)
	}
}

func TestReviewQueueSaturation(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"READY-1", "READY-2"} {
		worker := filepath.Join(root, strings.ToLower(id))
		git(t, project, "worktree", "add", "-q", "-b", "codex/"+strings.ToLower(id), worker, "HEAD")
		if err := cmdRecordTask([]string{"--ledger", ledger, "--id", id, "--worktree", worker, "--branch", "codex/" + strings.ToLower(id)}); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(worker, id+".txt"), []byte("done\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		git(t, worker, "add", ".")
		git(t, worker, "commit", "-q", "-m", id)
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.OverallStatus != "review-needed" {
		t.Fatalf("expected review-needed, got %q", summary.OverallStatus)
	}
	if summary.ReviewPressure.ReviewNeeded != 2 {
		t.Fatalf("expected two review-needed tasks, got %#v", summary.ReviewPressure)
	}
	if len(summary.RecommendedActions) == 0 || !strings.Contains(summary.RecommendedActions[0], "saturated") {
		t.Fatalf("expected saturated review queue action, got %#v", summary.RecommendedActions)
	}
}

func TestBadLedgerAndUnknownTaskErrors(t *testing.T) {
	root := t.TempDir()
	badLedger := filepath.Join(root, "bad.json")
	if err := os.WriteFile(badLedger, []byte("{bad json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := observe(badLedger); err == nil {
		t.Fatal("expected bad ledger error")
	}
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "review", "--task-id", "UNKNOWN", "--status", "blocked"}); err == nil {
		t.Fatal("expected unknown task error")
	}
}

func TestValidateRoutineSpecs(t *testing.T) {
	report := validateRoutines(filepath.Join("..", "..", "routines"))
	if !report.Valid {
		t.Fatalf("expected bundled routines to validate: %#v", report)
	}
	if len(report.Specs) < 3 {
		t.Fatalf("expected at least three routine specs, got %d", len(report.Specs))
	}
}

func TestValidateRoutineSpecRejectsWeakContract(t *testing.T) {
	issues := validateRoutineSpec(RoutineSpec{
		SchemaVersion: 1,
		ID:            "weak",
		Title:         "Weak",
		Purpose:       "too weak",
		Trigger:       "manual",
		Inputs:        []string{"git status"},
		AllowedActions: []string{
			"inspect",
		},
		ForbiddenActions: []string{"merge"},
		Gates:            []string{"git diff --check"},
		EvidenceLabels:   []string{"direct"},
		OutputSchema: RoutineOutputSpec{
			RequiredFields: []string{"status"},
			StatusValues:   []string{"passed"},
		},
		Escalation: []string{"ask human"},
	})
	if len(issues) == 0 {
		t.Fatal("expected weak routine contract to fail validation")
	}
	joined := strings.Join(issues, "\n")
	for _, want := range []string{"evidenceLabels must include blocked", "outputSchema.requiredFields must include evidence", "outputSchema.statusValues must include blocked"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected issue %q in:\n%s", want, joined)
		}
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
