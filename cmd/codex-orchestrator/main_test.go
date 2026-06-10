package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

func TestRecordRoutineRun(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "TASK-1",
		"--worktree", filepath.Join(root, "missing"),
		"--branch", "codex/task-1",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordRoutineRun([]string{
		"--ledger", ledger,
		"--routine", "pr-reviewer",
		"--task-id", "TASK-1",
		"--status", "passed",
		"--evidence-local", "go test ./...",
		"--action", "reviewed diff",
		"--next", "merge task branch",
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.RoutineRuns) != 1 {
		t.Fatalf("expected one routine run, got %d", len(updated.RoutineRuns))
	}
	run := updated.RoutineRuns[0]
	if run.RoutineID != "pr-reviewer" || run.TaskID != "TASK-1" || run.Status != "passed" {
		t.Fatalf("unexpected routine run: %#v", run)
	}
	if got := run.Evidence["local"]; len(got) != 1 || got[0] != "go test ./..." {
		t.Fatalf("expected local evidence, got %#v", run.Evidence)
	}
	events, err := os.ReadFile(filepath.Join(project, ".codex-orchestrator", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(events), `"type":"routine-run"`) {
		t.Fatalf("expected routine-run event, got %s", string(events))
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.RecentRoutineRuns) != 1 || summary.RecentRoutineRuns[0].RoutineID != "pr-reviewer" {
		t.Fatalf("expected recent routine run in observe summary, got %#v", summary.RecentRoutineRuns)
	}
	rendered := renderSummary(summary)
	if !strings.Contains(rendered, "Recent Routine Runs") {
		t.Fatalf("expected rendered summary to include routine runs:\n%s", rendered)
	}
}

func TestRunPRReviewerRoutineWritesPassedReport(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	reportPath := filepath.Join(root, "reports", "pr-reviewer.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/pr-review", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PR-REVIEW",
		"--worktree", worker,
		"--branch", "codex/pr-review",
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "feature.txt")
	git(t, worker, "commit", "-q", "-m", "feature")

	if err := cmdRunRoutine([]string{"pr-reviewer", "--ledger", ledger, "--task-id", "PR-REVIEW", "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report RoutineRunReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.RoutineID != "pr-reviewer" || report.TaskID != "PR-REVIEW" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	joined := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{"Task exists in ledger", "Worktree exists", "git diff --name-status", "A\tfeature.txt", "git diff --check"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, joined)
		}
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected only local evidence, got %#v", report.Evidence)
	}
}

func TestRunPRReviewerRoutineBlockedWhenTaskMissing(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	report, err := runPRReviewerRoutine(ledger, "UNKNOWN")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked missing-task report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "Task not found") {
		t.Fatalf("expected blocked task evidence, got %#v", report.Evidence)
	}
}

func TestRunPRReviewerRoutineBlockedOnBranchMismatch(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/actual", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "WRONG-BRANCH",
		"--worktree", worker,
		"--branch", "codex/expected",
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}
	report, err := runPRReviewerRoutine(ledger, "WRONG-BRANCH")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "blocked" || !strings.Contains(report.BlockedReason, "branch") {
		t.Fatalf("expected blocked branch report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "Expected branch codex/expected") {
		t.Fatalf("expected branch mismatch evidence, got %#v", report.Evidence)
	}
}

func TestRunPRReviewerRoutineFailsOnDirtyWorktree(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/dirty-review", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "DIRTY-REVIEW",
		"--worktree", worker,
		"--branch", "codex/dirty-review",
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := runPRReviewerRoutine(ledger, "DIRTY-REVIEW")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" {
		t.Fatalf("expected failed dirty report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["local"], "\n"); !strings.Contains(got, "uncommitted changes") {
		t.Fatalf("expected dirty evidence, got %#v", report.Evidence)
	}
}

func TestRunStaleTaskRescuerRoutineWritesPassedReport(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	reportPath := filepath.Join(root, "reports", "stale-task-rescuer.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/stale-rescue", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "STALE-RESCUE",
		"--worktree", worker,
		"--branch", "codex/stale-rescue",
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}
	ledgerData, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	ledgerData.Tasks[0].LastObservation["at"] = time.Now().Add(-45 * time.Minute).Format(time.RFC3339)
	ledgerData.Tasks[0].LastObservation["status"] = "stale-needs-inspection"
	ledgerData.Tasks[0].History = append(ledgerData.Tasks[0].History, map[string]string{
		"at":     time.Now().Add(-45 * time.Minute).Format(time.RFC3339),
		"type":   "observe",
		"status": "stale-needs-inspection",
		"note":   "stale",
	})
	if err := saveLedger(ledger, &ledgerData); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "feature.txt")
	git(t, worker, "commit", "-q", "-m", "feature")

	if err := cmdRunRoutine([]string{"stale-task-rescuer", "--ledger", ledger, "--task-id", "STALE-RESCUE", "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report RoutineRunReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.RoutineID != "stale-task-rescuer" || report.TaskID != "STALE-RESCUE" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	joined := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{"Last observation:", "Task history:", "git log --oneline -3", "git diff --name-status", "A\tfeature.txt"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, joined)
		}
	}
	if !strings.Contains(report.NextSuggestedAction, "orchestrator review") {
		t.Fatalf("expected review next action, got %q", report.NextSuggestedAction)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected only local evidence, got %#v", report.Evidence)
	}
}

func TestRunStaleTaskRescuerRoutineFailsOnDirtyWorktree(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/dirty-rescue", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "DIRTY-RESCUE",
		"--worktree", worker,
		"--branch", "codex/dirty-rescue",
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := runStaleTaskRescuerRoutine(ledger, "DIRTY-RESCUE")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" {
		t.Fatalf("expected failed dirty report, got %#v", report)
	}
	joined := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{"uncommitted changes", "git ls-files --others --exclude-standard", "dirty.txt"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected dirty evidence %q in:\n%s", want, joined)
		}
	}
	if !strings.Contains(report.NextSuggestedAction, "same-task takeover") {
		t.Fatalf("expected same-task rescue next action, got %q", report.NextSuggestedAction)
	}
}

func TestRunStaleTaskRescuerRoutineBlockedOnMissingBaseCommit(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/no-base", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "NO-BASE",
		"--worktree", worker,
		"--branch", "codex/no-base",
	}); err != nil {
		t.Fatal(err)
	}
	ledgerData, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	ledgerData.Tasks[0].BaseCommit = ""
	if err := saveLedger(ledger, &ledgerData); err != nil {
		t.Fatal(err)
	}
	report, err := runStaleTaskRescuerRoutine(ledger, "NO-BASE")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "blocked" || report.BlockedReason != "task baseCommit is missing" {
		t.Fatalf("expected blocked missing-base report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "no comparable baseCommit") {
		t.Fatalf("expected baseCommit blocked evidence, got %#v", report.Evidence)
	}
}

func TestRunCIFixerRoutineWritesPassedReport(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	reportPath := filepath.Join(root, "reports", "ci-fixer.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/ci-pass", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "CI-PASS",
		"--worktree", worker,
		"--branch", "codex/ci-pass",
		"--base-commit", base,
		"--gate", "test -f feature.txt",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "feature.txt")
	git(t, worker, "commit", "-q", "-m", "feature")

	if err := cmdRunRoutine([]string{"ci-fixer", "--ledger", ledger, "--task-id", "CI-PASS", "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report RoutineRunReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.RoutineID != "ci-fixer" || report.TaskID != "CI-PASS" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	joined := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{"Recorded task gates", "git diff --name-status", "A\tfeature.txt", "gate test -f feature.txt passed", "post-gate git status"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, joined)
		}
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected only local evidence, got %#v", report.Evidence)
	}
	if !strings.Contains(report.NextSuggestedAction, "no automatic code changes") {
		t.Fatalf("expected no-auto-fix next action, got %q", report.NextSuggestedAction)
	}
}

func TestRunCIFixerRoutineFailsOnGateFailure(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/ci-fail", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "CI-FAIL",
		"--worktree", worker,
		"--branch", "codex/ci-fail",
		"--base-commit", base,
		"--gate", "printf gate-failed && exit 7",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "feature.txt")
	git(t, worker, "commit", "-q", "-m", "feature")

	report, err := runCIFixerRoutine(ledger, "CI-FAIL")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	joined := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{"gate printf gate-failed && exit 7 failed", "gate-failed"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected failed gate evidence %q in:\n%s", want, joined)
		}
	}
	if !strings.Contains(report.NextSuggestedAction, "same task worker") {
		t.Fatalf("expected same worker next action, got %q", report.NextSuggestedAction)
	}
}

func TestRunCIFixerRoutineBlockedWhenGatesMissing(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/ci-no-gates", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "CI-NO-GATES",
		"--worktree", worker,
		"--branch", "codex/ci-no-gates",
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}

	report, err := runCIFixerRoutine(ledger, "CI-NO-GATES")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "blocked" || report.BlockedReason != "task gates are missing" {
		t.Fatalf("expected blocked missing-gates report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "no recorded gates") {
		t.Fatalf("expected missing-gates blocked evidence, got %#v", report.Evidence)
	}
}

func TestRunReleaseVerifierRoutineWritesPassedAlphaReport(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	git(t, project, "tag", "v0.3.0-alpha.1")
	reportPath := filepath.Join(root, "reports", "release-verifier.json")
	withFakeGH(t, fakeReleaseViewJSON("v0.3.0-alpha.1", true, defaultReleaseVerifierAssets()))

	if err := cmdRunRoutine([]string{"release-verifier", "--repo", project, "--tag", "v0.3.0-alpha.1", "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report RoutineRunReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.RoutineID != "release-verifier" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{"Local git tag v0.3.0-alpha.1 resolves", "Expected release assets"} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
	proxy := strings.Join(report.Evidence["proxy"], "\n")
	for _, want := range []string{"GitHub release exists", "prerelease=true", "codex-orchestrator_windows_amd64.exe.zip"} {
		if !strings.Contains(proxy, want) {
			t.Fatalf("expected proxy evidence %q in:\n%s", want, proxy)
		}
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected only local/proxy evidence, got %#v", report.Evidence)
	}
}

func TestRunReleaseVerifierRoutineFailsWhenAlphaIsNotPrerelease(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	git(t, project, "tag", "v0.3.0-alpha.2")
	withFakeGH(t, fakeReleaseViewJSON("v0.3.0-alpha.2", false, defaultReleaseVerifierAssets()))

	report := runReleaseVerifierRoutine(project, "v0.3.0-alpha.2", nil)
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["proxy"], "\n"); !strings.Contains(got, "prerelease mismatch") {
		t.Fatalf("expected prerelease mismatch evidence, got %#v", report.Evidence)
	}
}

func TestRunReleaseVerifierRoutineFailsWhenAssetsMissing(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	git(t, project, "tag", "v0.3.0")
	withFakeGH(t, fakeReleaseViewJSON("v0.3.0", false, []string{"codex-orchestrator_linux_amd64"}))

	report := runReleaseVerifierRoutine(project, "v0.3.0", []string{"codex-orchestrator_linux_amd64", "codex-orchestrator_darwin_arm64.tar.gz"})
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["proxy"], "\n"); !strings.Contains(got, "Missing expected release assets: codex-orchestrator_darwin_arm64.tar.gz") {
		t.Fatalf("expected missing asset evidence, got %#v", report.Evidence)
	}
}

func TestRunDocsDriftCheckerRoutineWritesPassedReport(t *testing.T) {
	root := t.TempDir()
	project := createDocsDriftFixture(t, root, []string{"pr-reviewer", "docs-drift-checker"})
	reportPath := filepath.Join(root, "reports", "docs-drift-checker.json")

	if err := cmdRunRoutine([]string{"docs-drift-checker", "--repo", project, "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report RoutineRunReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.RoutineID != "docs-drift-checker" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Runnable routines from cmd/codex-orchestrator/main.go: docs-drift-checker, pr-reviewer",
		"Routine specs in routines/: docs-drift-checker, pr-reviewer",
		"README.md mentions all runnable routines",
		"docs/roadmap.md mentions all runnable routines",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected only local evidence, got %#v", report.Evidence)
	}
}

func TestRunDocsDriftCheckerRoutineFailsOnMissingDocReference(t *testing.T) {
	root := t.TempDir()
	project := createDocsDriftFixture(t, root, []string{"pr-reviewer"})

	report := runDocsDriftCheckerRoutine(project)
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "README.md is missing runnable routine reference(s): docs-drift-checker") {
		t.Fatalf("expected missing docs-drift-checker evidence, got:\n%s", local)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected local-only failed evidence, got %#v", report.Evidence)
	}
}

func TestRunDocsDriftCheckerRoutineBlockedWhenSourceMissing(t *testing.T) {
	root := t.TempDir()
	report := runDocsDriftCheckerRoutine(root)
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "Could not read runnable routine source") {
		t.Fatalf("expected source blocked evidence, got %#v", report.Evidence)
	}
}

func TestRecordRoutineRunFromJSONReport(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "routine-report.json")
	report := RoutineRunReport{
		RoutineID: "api-proof",
		Status:    "blocked",
		Evidence: map[string][]string{
			"blocked": {"auth unavailable"},
		},
		ActionsTaken:        []string{"checked endpoint contract"},
		BlockedReason:       "missing token",
		NextSuggestedAction: "ask human for token",
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordRoutineRun([]string{"--ledger", ledger, "--report-json", reportPath}); err != nil {
		t.Fatal(err)
	}
	updated, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.RoutineRuns) != 1 {
		t.Fatalf("expected one routine run, got %d", len(updated.RoutineRuns))
	}
	run := updated.RoutineRuns[0]
	if run.RoutineID != "api-proof" || run.Status != "blocked" || run.BlockedReason != "missing token" {
		t.Fatalf("unexpected routine run from report: %#v", run)
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	rendered := renderSummary(summary)
	if !strings.Contains(rendered, "No tasks recorded") || !strings.Contains(rendered, "Recent Routine Runs") {
		t.Fatalf("expected empty-task summary to still include recent routine runs:\n%s", rendered)
	}
}

func TestRecordRoutineRunValidation(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordRoutineRun([]string{
		"--ledger", ledger,
		"--routine", "ci-fixer",
		"--status", "blocked",
		"--evidence-blocked", "CI logs unavailable",
		"--action", "checked workflow run",
		"--next", "ask human for CI access",
	}); err == nil {
		t.Fatal("expected blocked routine run to require blocked reason")
	}
	if err := cmdRecordRoutineRun([]string{
		"--ledger", ledger,
		"--routine", "ci-fixer",
		"--task-id", "UNKNOWN",
		"--status", "passed",
		"--evidence-local", "go test ./...",
		"--action", "ran tests",
		"--next", "continue",
	}); err == nil {
		t.Fatal("expected unknown task error")
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

func withFakeGH(t *testing.T, output string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "bin")
	if err := os.MkdirAll(bin, 0o755); err != nil {
		t.Fatal(err)
	}
	name := "gh"
	if runtime.GOOS == "windows" {
		name = "gh.bat"
	}
	path := filepath.Join(bin, name)
	var script string
	if runtime.GOOS == "windows" {
		script = "@echo off\r\necho " + output + "\r\n"
	} else {
		script = "#!/bin/sh\ncat <<'JSON'\n" + output + "\nJSON\n"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func fakeReleaseViewJSON(tag string, prerelease bool, assets []string) string {
	release := githubReleaseView{
		TagName:         tag,
		IsPrerelease:    prerelease,
		IsDraft:         false,
		URL:             "https://github.com/example/codex-orchestrator/releases/tag/" + tag,
		TargetCommitish: "main",
		Assets:          make([]githubReleaseAsset, 0, len(assets)),
	}
	for _, asset := range assets {
		release.Assets = append(release.Assets, githubReleaseAsset{Name: asset})
	}
	data, err := json.Marshal(release)
	if err != nil {
		panic(err)
	}
	return string(data)
}

func createDocsDriftFixture(t *testing.T, root string, readmeRoutineMentions []string) string {
	t.Helper()
	project := filepath.Join(root, "repo")
	for _, dir := range []string{
		filepath.Join(project, "cmd", "codex-orchestrator"),
		filepath.Join(project, "docs", "routines"),
		filepath.Join(project, "routines"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	source := `package main

func cmdRunRoutine(args []string) error {
	switch args[0] {
	case "pr-reviewer":
		return nil
	case "docs-drift-checker":
		return nil
	default:
		return nil
	}
}
`
	if err := os.WriteFile(filepath.Join(project, "cmd", "codex-orchestrator", "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"pr-reviewer", "docs-drift-checker"} {
		data, err := json.Marshal(RoutineSpec{ID: id})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, "routines", id+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	allMentions := "pr-reviewer docs-drift-checker"
	docs := map[string]string{
		"README.md":       strings.Join(readmeRoutineMentions, " "),
		"README.zh-CN.md": allMentions,
		"SKILL.md":        allMentions,
		filepath.Join("docs", "routines", "README.md"): allMentions,
		filepath.Join("docs", "roadmap.md"):            allMentions,
	}
	for path, text := range docs {
		if err := os.WriteFile(filepath.Join(project, path), []byte(text+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return project
}
