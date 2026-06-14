package main

import (
	"encoding/json"
	"errors"
	"io"
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

func TestWriteJSONUsesReplaceableTargetWithoutTempLeak(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "state", "ledger.json")
	if err := writeJSON(target, map[string]any{"version": 1, "status": "ok"}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("expected valid json, got %q: %v", string(data), err)
	}
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(target), ".ledger.json.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no temporary files, got %#v", matches)
	}
	if err := writeJSON(target, map[string]any{"version": 2, "status": "updated"}); err != nil {
		t.Fatal(err)
	}
	updated, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), `"version": 2`) {
		t.Fatalf("expected replaced json, got %s", string(updated))
	}
}

func TestShellCommandUsesNonLoginShell(t *testing.T) {
	shell, args := shellCommand("echo ok")
	if runtime.GOOS == "windows" {
		if !strings.EqualFold(filepath.Base(shell), "cmd") && !strings.EqualFold(filepath.Base(shell), "cmd.exe") {
			t.Fatalf("expected cmd shell on windows, got %q", shell)
		}
		if len(args) != 2 || args[0] != "/C" {
			t.Fatalf("expected cmd /C args, got %#v", args)
		}
		return
	}
	if shell == "" {
		t.Fatal("expected shell")
	}
	if len(args) != 2 || args[0] != "-c" {
		t.Fatalf("expected non-login shell args, got %#v", args)
	}
}

func TestCompletionScriptsMentionCoreCommands(t *testing.T) {
	cases := []struct {
		name string
		text string
	}{
		{name: "bash", text: completionBash()},
		{name: "zsh", text: completionZsh()},
		{name: "fish", text: completionFish()},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, want := range []string{
				"dispatch",
				"run-routine",
				"release-verifier",
				"roadmap-next-task-suggester",
				"rules",
				"misalignment",
				"watchdog",
				"self-update",
				"completion",
			} {
				if !strings.Contains(tc.text, want) {
					t.Fatalf("expected %s completion to mention %q", tc.name, want)
				}
			}
		})
	}
}

func TestCompletionRejectsUnknownShell(t *testing.T) {
	if err := cmdCompletion([]string{"powershell"}); err == nil {
		t.Fatal("expected unsupported shell to fail")
	}
}

func TestSelfUpdateScriptArgs(t *testing.T) {
	cases := []struct {
		name       string
		skillOnly  bool
		helperOnly bool
		withHelper bool
		noHelper   bool
		want       []string
	}{
		{name: "default"},
		{name: "skill", skillOnly: true, want: []string{"--skill-only"}},
		{name: "helper", helperOnly: true, want: []string{"--helper-only"}},
		{name: "with-helper", withHelper: true, want: []string{"--with-helper"}},
		{name: "no-helper", noHelper: true, want: []string{"--no-helper"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := selfUpdateScriptArgs(tc.skillOnly, tc.helperOnly, tc.withHelper, tc.noHelper)
			if strings.Join(got, "\x00") != strings.Join(tc.want, "\x00") {
				t.Fatalf("expected %#v, got %#v", tc.want, got)
			}
		})
	}
}

func TestResolveSelfUpdateSource(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "codex-orchestrator")
	if err := os.MkdirAll(filepath.Join(source, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(source, "scripts", "update-local.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := resolveSelfUpdateSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("expected %q, got %q", source, got)
	}
	if _, err := resolveSelfUpdateSource(filepath.Join(root, "missing")); err == nil {
		t.Fatal("expected missing source to fail")
	}
}

func TestPrepareSelfUpdateSourceFromGitRefreshesCache(t *testing.T) {
	root := t.TempDir()
	remote := filepath.Join(root, "remote")
	cache := filepath.Join(root, "cache")
	if err := os.MkdirAll(filepath.Join(remote, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, remote, "init", "-q", "-b", "main")
	git(t, remote, "config", "user.email", "test@example.com")
	git(t, remote, "config", "user.name", "Test")
	script := filepath.Join(remote, "scripts", "update-local.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho first\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, remote, "add", "scripts/update-local.sh")
	git(t, remote, "commit", "-q", "-m", "first")

	got, notes, err := prepareSelfUpdateSourceFromGit(remote, cache)
	if err != nil {
		t.Fatal(err)
	}
	if got != cache {
		t.Fatalf("expected cache %q, got %q", cache, got)
	}
	if len(notes) == 0 || !strings.Contains(notes[0], "Cloned") {
		t.Fatalf("expected clone note, got %#v", notes)
	}
	cachedScript := filepath.Join(cache, "scripts", "update-local.sh")
	data, err := os.ReadFile(cachedScript)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "first") {
		t.Fatalf("expected first script content, got %s", data)
	}

	if err := os.WriteFile(script, []byte("#!/bin/sh\necho second\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, remote, "add", "scripts/update-local.sh")
	git(t, remote, "commit", "-q", "-m", "second")
	if _, notes, err := prepareSelfUpdateSourceFromGit(remote, cache); err != nil {
		t.Fatal(err)
	} else if len(notes) == 0 || !strings.Contains(notes[0], "Refreshed") {
		t.Fatalf("expected refresh note, got %#v", notes)
	}
	updated, err := os.ReadFile(cachedScript)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(updated), "second") {
		t.Fatalf("expected refreshed script content, got %s", updated)
	}
}

func TestDefaultSelfUpdateCacheDirUsesEnv(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CODEX_ORCHESTRATOR_UPDATE_CACHE", root)
	if got := defaultSelfUpdateCacheDir(); got != root {
		t.Fatalf("expected env cache dir %q, got %q", root, got)
	}
}

func TestSelfUpdateRejectsConflictingModes(t *testing.T) {
	if err := cmdSelfUpdate([]string{"--skill-only", "--with-helper", "--dry-run"}); err == nil {
		t.Fatal("expected conflicting self-update modes to fail")
	}
}

func TestSelfUpdateRejectsConflictingSources(t *testing.T) {
	if err := cmdSelfUpdate([]string{"--source", t.TempDir(), "--from-github", "--dry-run"}); err == nil {
		t.Fatal("expected conflicting self-update sources to fail")
	}
}

func TestWatchdogStatusReadsLaunchAgentAndHeartbeatReport(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	previousInspect := inspectLaunchAgentLoadedFn
	inspectLaunchAgentLoadedFn = func(label string) (string, string) {
		return "loaded", ""
	}
	t.Cleanup(func() {
		inspectLaunchAgentLoadedFn = previousInspect
	})
	project := createRepo(t, filepath.Join(root, "repo"))
	stateDir := filepath.Join(project, defaultStateDir)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	suffix := "unit-test"
	label := "com.indiekitai.codex-orchestrator.watchdog." + suffix
	plistPath := filepath.Join(home, "Library", "LaunchAgents", label+".plist")
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(plistPath, []byte(`<plist><dict><key>Label</key><string>`+label+`</string><key>REPO</key><string>`+project+`</string></dict></plist>`), 0o644); err != nil {
		t.Fatal(err)
	}
	report := ObserveSummary{
		ObservedAt: "2026-06-12T10:00:00+08:00",
		HeartbeatStatus: &HeartbeatStatus{
			EvidenceLabel:       "local/static",
			Status:              "missed",
			CurrentHeartbeatAt:  "2026-06-12T10:00:00+08:00",
			ExpectedInterval:    "20m0s",
			MissedAfter:         "45m0s",
			Gap:                 "5h0m0s",
			EstimatedMissedRuns: 14,
			Note:                "Possible missed heartbeat.",
		},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "watchdog-heartbeat-report.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "watchdog-heartbeat-summary.md"), []byte("# watchdog\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "watchdog-last-error.log"), []byte("sample failure\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	status, err := inspectWatchdogStatus(project, suffix)
	if err != nil {
		t.Fatal(err)
	}
	if status.EvidenceLabel != "local/static" {
		t.Fatalf("expected local/static evidence, got %q", status.EvidenceLabel)
	}
	if !status.Installed {
		t.Fatal("expected installed LaunchAgent plist")
	}
	if status.LoadedStatus != "loaded" {
		t.Fatalf("expected mocked loaded status, got %q", status.LoadedStatus)
	}
	if !status.ReportExists || !status.SummaryExists || !status.LastErrorExists {
		t.Fatalf("expected watchdog artifacts, got %#v", status)
	}
	if status.HeartbeatStatus == nil || status.HeartbeatStatus.Status != "missed" {
		t.Fatalf("expected missed heartbeat status, got %#v", status.HeartbeatStatus)
	}
	if !strings.Contains(strings.Join(status.RecommendedActions, "\n"), "missed") {
		t.Fatalf("expected missed heartbeat recommended action, got %#v", status.RecommendedActions)
	}
}

func TestWatchdogStatusMissingInstallRecommendsInstall(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	t.Setenv("HOME", home)
	project := createRepo(t, filepath.Join(root, "repo"))
	status, err := inspectWatchdogStatus(project, "not-installed")
	if err != nil {
		t.Fatal(err)
	}
	if status.Installed {
		t.Fatal("expected no LaunchAgent plist")
	}
	text := strings.Join(status.RecommendedActions, "\n")
	if !strings.Contains(text, "install") || !strings.Contains(text, "local/static") {
		t.Fatalf("expected install/local-static recommendation, got %q", text)
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

func TestPendingWorktreeIDTaskLifecycle(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PENDING",
		"--title", "Pending setup",
		"--thread-id", "thread_123",
		"--pending-worktree-id", "pwt_123",
	}); err != nil {
		t.Fatal(err)
	}

	stored, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Tasks) != 1 {
		t.Fatalf("expected one task, got %d", len(stored.Tasks))
	}
	task := stored.Tasks[0]
	if task.PendingWorktreeID != "pwt_123" {
		t.Fatalf("expected pending worktree id, got %#v", task)
	}
	if task.Worktree != "" || task.Branch != "" {
		t.Fatalf("expected no worktree or branch before setup completes, got %#v", task)
	}
	if task.Status != "pending-setup" {
		t.Fatalf("expected pending-setup ledger status, got %q", task.Status)
	}

	statusJSON := captureStdout(t, func() error {
		return cmdStatus([]string{"--ledger", ledger, "--json"})
	})
	var statusPayload struct {
		Tasks         []Task              `json:"tasks"`
		RuntimeStatus RuntimeStatusReport `json:"runtimeStatus"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &statusPayload); err != nil {
		t.Fatalf("expected status JSON, got %q: %v", statusJSON, err)
	}
	if len(statusPayload.Tasks) != 1 || statusPayload.Tasks[0].PendingWorktreeID != "pwt_123" {
		t.Fatalf("expected status JSON to expose pendingWorktreeId, got %#v", statusPayload.Tasks)
	}
	if len(statusPayload.RuntimeStatus.PendingSetup) != 1 || statusPayload.RuntimeStatus.PendingSetup[0].PendingWorktreeID != "pwt_123" {
		t.Fatalf("expected runtime status pending setup bucket, got %#v", statusPayload.RuntimeStatus)
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if got := summary.Observations[0].Status; got != "pending-setup" {
		t.Fatalf("expected pending setup observation, got %q", got)
	}
	if got := summary.Observations[0].State.Setup; got != "pending-worktree-id" {
		t.Fatalf("expected pending setup state, got %#v", summary.Observations[0].State)
	}
	if got := summary.Observations[0].PendingWorktreeID; got != "pwt_123" {
		t.Fatalf("expected observation pendingWorktreeId, got %q", got)
	}
	if summary.ReviewPressure.PendingSetup != 1 || summary.ReviewPressure.Blocked != 0 {
		t.Fatalf("expected pending setup pressure without blocker, got %#v", summary.ReviewPressure)
	}

	worker := filepath.Join(root, "worker")
	git(t, project, "worktree", "add", "-q", "-b", "codex/pending", worker, "HEAD")
	if err := cmdAppendEvent([]string{
		"--ledger", ledger,
		"--type", "setup-complete",
		"--task-id", "PENDING",
		"--status", "active",
		"--worktree", worker,
		"--branch", "codex/pending",
		"--note", "Codex App worktree setup completed.",
	}); err != nil {
		t.Fatal(err)
	}
	reconciled, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if reconciled.Tasks[0].Worktree != worker || reconciled.Tasks[0].Branch != "codex/pending" {
		t.Fatalf("expected reconciled worktree and branch, got %#v", reconciled.Tasks[0])
	}
	if reconciled.Tasks[0].PendingWorktreeID != "pwt_123" {
		t.Fatalf("expected opaque pending id to remain recorded, got %#v", reconciled.Tasks[0])
	}
	summary, err = observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if got := summary.Observations[0].Status; got != "active" {
		t.Fatalf("expected active after worktree reconciliation, got %q", got)
	}
	if state := summary.Observations[0].State; state.Setup != "worktree-present" || state.Worktree != "present" || state.Branch != "matched" {
		t.Fatalf("expected reconciled real worktree state, got %#v", state)
	}
}

func TestObserveReconcilesPendingSetupFromGitWorktreeTruthReadOnly(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	branch := "codex/pending-reconcile"
	worker := filepath.Join(root, "worker")
	git(t, project, "worktree", "add", "-q", "-b", branch, worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PENDING-RECONCILE",
		"--title", "Pending reconcile",
		"--package-id", "PKG-ONE",
		"--pending-worktree-id", "pwt_reconcile",
		"--branch", branch,
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := observeWithOptions(ledger, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if summary.ReviewPressure.PendingSetup != 0 {
		t.Fatalf("expected pending setup to reconcile to worktree truth, got %#v", summary.ReviewPressure)
	}
	if summary.ReviewPressure.Active != 1 || len(summary.RuntimeStatus.ActiveWorkers) != 1 {
		t.Fatalf("expected active worker from git worktree truth, got pressure=%#v runtime=%#v", summary.ReviewPressure, summary.RuntimeStatus)
	}
	item := summary.RuntimeStatus.ActiveWorkers[0]
	expectedWorker, err := filepath.EvalSymlinks(worker)
	if err != nil {
		t.Fatal(err)
	}
	actualWorker, err := filepath.EvalSymlinks(item.Worktree)
	if err != nil {
		t.Fatal(err)
	}
	if actualWorker != expectedWorker || item.State.Setup != "worktree-found-ledger-missing" || item.State.Worktree != "present-unrecorded" {
		t.Fatalf("expected read-only reconciled worktree state, got %#v", item)
	}
	if !strings.Contains(item.Action, "dispatch reconcile") || !strings.Contains(item.Note, "ledger still has no worktree path") {
		t.Fatalf("expected reconcile action/note, got action=%q note=%q", item.Action, item.Note)
	}
	if summary.OverallStatus != "quiet" || strings.Contains(strings.Join(summary.RecommendedActions, "\n"), "dispatch the next safe roadmap task") {
		t.Fatalf("expected active worker to suppress generic dispatch recommendation, got status=%q actions=%v", summary.OverallStatus, summary.RecommendedActions)
	}
	stored, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Tasks[0].Worktree != "" {
		t.Fatalf("observe must not mutate ledger worktree, got %#v", stored.Tasks[0])
	}
}

func TestObserveReconcileWriteUpdatesLedgerFromGitTruth(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	branch := "codex/observe-reconcile-write"
	worker := filepath.Join(root, "worker")
	git(t, project, "worktree", "add", "-q", "-b", branch, worker, "HEAD")
	if err := os.WriteFile(filepath.Join(worker, "worker.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "worker.txt")
	git(t, worker, "commit", "-q", "-m", "worker done")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "OBS-RECONCILE",
		"--title", "Observe reconcile",
		"--package-id", "PKG-OBS",
		"--pending-worktree-id", "pwt_observe",
		"--branch", branch,
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() error {
		return cmdObserve([]string{"--ledger", ledger, "--reconcile", "--write", "--json"})
	})
	var summary ObserveSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("expected observe JSON, got %q: %v", output, err)
	}
	if summary.Reconciliation == nil || summary.Reconciliation.Status != "updated" || summary.Reconciliation.Updated != 1 {
		t.Fatalf("expected reconciliation update in summary, got %#v", summary.Reconciliation)
	}
	stored, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	task := stored.Tasks[0]
	if task.Status != "completed-unreviewed" {
		t.Fatalf("expected completed-unreviewed ledger status, got %#v", task)
	}
	if cleanAbsPath(task.Worktree) != cleanAbsPath(worker) || task.Branch != branch {
		t.Fatalf("expected worktree and branch to be written, got %#v", task)
	}
	if len(task.History) != 2 || task.History[1]["type"] != "observe-reconcile" {
		t.Fatalf("expected observe-reconcile history event, got %#v", task.History)
	}
	events, err := os.ReadFile(eventsPathForLedger(ledger))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(events), `"type":"observe-reconcile"`) {
		t.Fatalf("expected observe-reconcile event, got %s", string(events))
	}
}

func TestObserveReconcileRequiresWrite(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdObserve([]string{"--ledger", ledger, "--reconcile"}); err == nil || !strings.Contains(err.Error(), "--write") {
		t.Fatalf("expected observe --reconcile to require --write, got %v", err)
	}
	if err := cmdObserve([]string{"--ledger", ledger, "--write"}); err == nil || !strings.Contains(err.Error(), "--reconcile") {
		t.Fatalf("expected observe --write to require --reconcile, got %v", err)
	}
}

func TestDispatchRecordStoresPendingSetupContract(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, defaultLedger)
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() error {
		return cmdDispatch([]string{
			"record",
			"--repo", project,
			"--task-id", "DISPATCH-PENDING",
			"--title", "Dispatch pending",
			"--thread-id", "thread_456",
			"--pending-worktree-id", "pwt_456",
			"--branch", "codex/dispatch-pending",
			"--base-commit", gitOutputForTest(t, project, "rev-parse", "HEAD"),
			"--allowed", "cmd/codex-orchestrator/**",
			"--forbidden", ".github/workflows/**",
			"--gate", "go test ./...",
			"--json",
		})
	})
	var payload DispatchResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected dispatch record JSON, got %q: %v", output, err)
	}
	if payload.Command != "dispatch record" || payload.EvidenceLabel != "local/static" {
		t.Fatalf("unexpected payload identity: %#v", payload)
	}
	if !strings.Contains(strings.Join(payload.Warnings, "\n"), "not proof that a worker is running") {
		t.Fatalf("expected honest pending setup warning, got %#v", payload.Warnings)
	}
	if payload.Task.ID != "DISPATCH-PENDING" || payload.Task.Status != "pending-setup" {
		t.Fatalf("expected pending setup task, got %#v", payload.Task)
	}
	if payload.Task.PendingWorktreeID != "pwt_456" || payload.Task.ThreadID != "thread_456" {
		t.Fatalf("expected App setup identifiers to be stored, got %#v", payload.Task)
	}
	if payload.Task.Branch != "codex/dispatch-pending" {
		t.Fatalf("expected branch to be preserved before worktree resolution, got %#v", payload.Task)
	}
	if got := payload.Task.WriteSet["allowed"]; len(got) != 1 || got[0] != "cmd/codex-orchestrator/**" {
		t.Fatalf("expected allowed path to be stored, got %#v", payload.Task.WriteSet)
	}
	if got := payload.Task.WriteSet["forbidden"]; len(got) != 1 || got[0] != ".github/workflows/**" {
		t.Fatalf("expected forbidden path to be stored, got %#v", payload.Task.WriteSet)
	}
	if len(payload.Task.Gates) != 1 || payload.Task.Gates[0] != "go test ./..." {
		t.Fatalf("expected gate to be stored, got %#v", payload.Task.Gates)
	}
}

func TestDispatchReconcileResolvesPendingTaskFromGitWorktreeTruth(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, defaultLedger)
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	if err := cmdDispatch([]string{
		"record",
		"--repo", project,
		"--task-id", "DISPATCH-RECONCILE",
		"--pending-worktree-id", "pwt_789",
		"--branch", "codex/dispatch-reconcile",
		"--base-commit", base,
	}); err != nil {
		t.Fatal(err)
	}
	worker := filepath.Join(root, "worker")
	git(t, project, "worktree", "add", "-q", "-b", "codex/dispatch-reconcile", worker, "HEAD")

	output := captureStdout(t, func() error {
		return cmdDispatch([]string{
			"reconcile",
			"--repo", project,
			"--task-id", "DISPATCH-RECONCILE",
			"--json",
		})
	})
	var payload DispatchResult
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("expected dispatch reconcile JSON, got %q: %v", output, err)
	}
	if payload.Command != "dispatch reconcile" || payload.EvidenceLabel != "local/static" {
		t.Fatalf("unexpected payload identity: %#v", payload)
	}
	if cleanAbsPath(payload.Task.Worktree) != cleanAbsPath(worker) || payload.Task.Branch != "codex/dispatch-reconcile" {
		t.Fatalf("expected reconciled worktree and branch, got %#v", payload.Task)
	}
	if payload.Task.PendingWorktreeID != "pwt_789" {
		t.Fatalf("expected pending id to remain durable, got %#v", payload.Task)
	}
	if payload.GitWorktree == nil || cleanAbsPath(payload.GitWorktree.Path) != cleanAbsPath(worker) || payload.GitWorktree.Branch != "codex/dispatch-reconcile" {
		t.Fatalf("expected git worktree truth in payload, got %#v", payload.GitWorktree)
	}
	if !strings.Contains(strings.Join(payload.Warnings, "\n"), "not proof of task correctness") {
		t.Fatalf("expected correctness warning, got %#v", payload.Warnings)
	}

	stored, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	task := stored.Tasks[0]
	if cleanAbsPath(task.Worktree) != cleanAbsPath(worker) || task.Branch != "codex/dispatch-reconcile" || task.Status != "active" {
		t.Fatalf("expected ledger task to be reconciled active, got %#v", task)
	}
	if len(task.History) != 2 || task.History[1]["type"] != "dispatch-reconcile" {
		t.Fatalf("expected reconcile history event, got %#v", task.History)
	}
}

func TestDispatchReconcileRejectsBranchMismatch(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, defaultLedger)
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdDispatch([]string{
		"record",
		"--repo", project,
		"--task-id", "DISPATCH-MISMATCH",
		"--pending-worktree-id", "pwt_mismatch",
		"--branch", "codex/expected",
	}); err != nil {
		t.Fatal(err)
	}
	worker := filepath.Join(root, "worker")
	git(t, project, "worktree", "add", "-q", "-b", "codex/actual", worker, "HEAD")

	err := cmdDispatch([]string{
		"reconcile",
		"--repo", project,
		"--task-id", "DISPATCH-MISMATCH",
		"--worktree", worker,
	})
	if err == nil {
		t.Fatal("expected branch mismatch error")
	}
	if !strings.Contains(err.Error(), "branch mismatch") {
		t.Fatalf("expected branch mismatch error, got %v", err)
	}
}

func TestDetachedWorkerBranchBlocksSetupTruth(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	worker := filepath.Join(root, "detached")
	git(t, project, "worktree", "add", "-q", "-b", "codex/detached", worker, "HEAD")
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "DETACHED", "--worktree", worker, "--branch", "codex/detached", "--base-commit", base}); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "checkout", "-q", "--detach", "HEAD")

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if got := summary.Observations[0].Status; got != "blocked" {
		t.Fatalf("expected detached worktree to be blocked, got %q", got)
	}
	state := summary.Observations[0].State
	if state.Branch != "detached" || state.Worktree != "present" || state.Setup != "worktree-present" {
		t.Fatalf("expected detached branch state, got %#v", state)
	}
	if len(summary.RuntimeStatus.Blockers) != 1 || summary.RuntimeStatus.Blockers[0].State.Branch != "detached" {
		t.Fatalf("expected runtime blocker with detached state, got %#v", summary.RuntimeStatus.Blockers)
	}
}

func TestLegacyLedgerWithoutPendingWorktreeIDStillLoads(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	worker := filepath.Join(root, "worker")
	git(t, project, "worktree", "add", "-q", "-b", "codex/legacy", worker, "HEAD")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	now := nowISO()
	legacy := map[string]any{
		"version":        1,
		"projectRoot":    project,
		"defaultBranch":  "master",
		"remote":         "origin",
		"pushPolicy":     "manual",
		"maxConcurrency": 2,
		"createdAt":      now,
		"updatedAt":      now,
		"tasks": []map[string]any{{
			"id":         "LEGACY",
			"title":      "Legacy task",
			"worktree":   worker,
			"branch":     "codex/legacy",
			"baseCommit": base,
			"status":     "active",
			"lastObservation": map[string]string{
				"at":     now,
				"result": "active",
			},
		}},
	}
	if err := writeJSON(ledger, legacy); err != nil {
		t.Fatal(err)
	}
	loaded, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Tasks[0].PendingWorktreeID != "" {
		t.Fatalf("expected missing pendingWorktreeId to load as empty, got %#v", loaded.Tasks[0])
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if got := summary.Observations[0].Status; got != "active" {
		t.Fatalf("expected legacy task to remain active, got %q", got)
	}
}

func TestObserveRepoFlagResolvesDefaultLedger(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, defaultLedger)
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	otherCWD := filepath.Join(root, "elsewhere")
	if err := os.MkdirAll(otherCWD, 0o755); err != nil {
		t.Fatal(err)
	}
	previousCWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(otherCWD); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousCWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
	if err := cmdObserve([]string{"--repo", project, "--json"}); err != nil {
		t.Fatal(err)
	}
}

func TestLedgerRepoFlagCompatibilityCommands(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, defaultLedger)
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	missingRepo := filepath.Join(root, "missing-repo")
	if err := cmdObserve([]string{"--repo", missingRepo, "--ledger", ledger, "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdStatus([]string{"--repo", project, "--json"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdHeartbeat([]string{"--repo", project, "--interval", "0", "--count", "1", "--json"}); err != nil {
		t.Fatal(err)
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

func TestHeartbeatCheckOnlyDoesNotAppendEvent(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	eventsPath := eventsPathForLedger(ledger)
	report := filepath.Join(project, ".codex-orchestrator", "watchdog-heartbeat-report.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := appendEvent(eventsPath, map[string]any{
		"at":     time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
		"type":   "heartbeat",
		"status": "quiet",
		"note":   "app heartbeat",
	}); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmdHeartbeat([]string{
		"--ledger", ledger,
		"--events", eventsPath,
		"--interval", "10m",
		"--missed-after", "25m",
		"--count", "1",
		"--check-only",
		"--write-report", report,
		"--json",
	}); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(eventsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("check-only heartbeat must not append events\nbefore:\n%s\nafter:\n%s", string(before), string(after))
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var observed ObserveSummary
	if err := json.Unmarshal(data, &observed); err != nil {
		t.Fatal(err)
	}
	if observed.HeartbeatStatus == nil || observed.HeartbeatStatus.Status != "missed" {
		t.Fatalf("expected check-only report to preserve missed heartbeat status, got %#v", observed.HeartbeatStatus)
	}
}

func TestTaskBudgetMetadataSurfacesInObserveAndHeartbeat(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	budgetWorker := filepath.Join(root, "worker-budget")
	legacyWorker := filepath.Join(root, "worker-legacy")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	report := filepath.Join(project, ".codex-orchestrator", "heartbeat-report.json")
	summaryPath := filepath.Join(project, ".codex-orchestrator", "heartbeat-summary.md")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/budget", budgetWorker, "HEAD")
	git(t, project, "worktree", "add", "-q", "-b", "codex/legacy", legacyWorker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "BUDGETED",
		"--worktree", budgetWorker,
		"--branch", "codex/budget",
		"--max-runtime-minutes", "90",
		"--review-budget-minutes", "25",
		"--budget-note", "local-only",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "LEGACY", "--worktree", legacyWorker, "--branch", "codex/legacy"}); err != nil {
		t.Fatal(err)
	}
	stored, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Tasks[0].Budget == nil {
		t.Fatal("expected budget metadata to be stored")
	}
	if stored.Tasks[0].Budget.MaxRuntimeMinutes != 90 || stored.Tasks[0].Budget.ReviewBudgetMinutes != 25 {
		t.Fatalf("unexpected stored budget: %#v", stored.Tasks[0].Budget)
	}
	if stored.Tasks[1].Budget != nil {
		t.Fatalf("expected legacy task to omit budget metadata, got %#v", stored.Tasks[1].Budget)
	}

	observed, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if observed.BudgetSummary.TasksWithBudget != 1 || observed.BudgetSummary.TasksMissingBudget != 1 {
		t.Fatalf("unexpected budget summary: %#v", observed.BudgetSummary)
	}
	if observed.BudgetSummary.TotalMaxRuntimeMinutes != 90 || observed.BudgetSummary.TotalReviewBudgetMinutes != 25 {
		t.Fatalf("unexpected budget totals: %#v", observed.BudgetSummary)
	}
	if observed.BudgetPressure.TasksMissingBudget != 1 {
		t.Fatalf("expected one missing-budget pressure warning, got %#v", observed.BudgetPressure)
	}
	var budgetObservation *Observation
	for index := range observed.Observations {
		if observed.Observations[index].ID == "BUDGETED" {
			budgetObservation = &observed.Observations[index]
			break
		}
	}
	if budgetObservation == nil || budgetObservation.Budget == nil {
		t.Fatalf("expected budgeted observation, got %#v", observed.Observations)
	}
	rendered := renderSummary(observed)
	for _, want := range []string{"tasksWithBudget: `1`", "totalMaxRuntimeMinutes: `90`", "budget: maxRuntime=90m review=25m note=local-only"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered summary to include %q:\n%s", want, rendered)
		}
	}

	if err := cmdHeartbeat([]string{"--ledger", ledger, "--interval", "0", "--count", "1", "--write-report", report, "--write-summary", summaryPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatal(err)
	}
	var heartbeat ObserveSummary
	if err := json.Unmarshal(data, &heartbeat); err != nil {
		t.Fatal(err)
	}
	if heartbeat.BudgetSummary.TasksWithBudget != 1 || heartbeat.BudgetSummary.TotalReviewBudgetMinutes != 25 {
		t.Fatalf("expected heartbeat report budget summary, got %#v", heartbeat.BudgetSummary)
	}
	summaryData, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(summaryData), "budget: maxRuntime=90m review=25m note=local-only") {
		t.Fatalf("expected heartbeat summary to include task budget:\n%s", string(summaryData))
	}
	if !strings.Contains(string(summaryData), "Budget Pressure") {
		t.Fatalf("expected heartbeat summary to include budget pressure:\n%s", string(summaryData))
	}
}

func TestBudgetPressureWarningsUseLocalLedgerTimestamps(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	runtimeWorker := filepath.Join(root, "worker-runtime")
	reviewWorker := filepath.Join(root, "worker-review")
	legacyWorker := filepath.Join(root, "worker-legacy")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/runtime", runtimeWorker, "HEAD")
	git(t, project, "worktree", "add", "-q", "-b", "codex/review", reviewWorker, "HEAD")
	git(t, project, "worktree", "add", "-q", "-b", "codex/legacy-budget-pressure", legacyWorker, "HEAD")

	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "RUNTIME",
		"--worktree", runtimeWorker,
		"--branch", "codex/runtime",
		"--max-runtime-minutes", "30",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "REVIEW",
		"--worktree", reviewWorker,
		"--branch", "codex/review",
		"--review-budget-minutes", "10",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "LEGACY", "--worktree", legacyWorker, "--branch", "codex/legacy-budget-pressure"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reviewWorker, "done.txt"), []byte("done\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, reviewWorker, "add", ".")
	git(t, reviewWorker, "commit", "-q", "-m", "ready for review")
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "review", "--task-id", "REVIEW", "--status", "completed-unreviewed", "--note", "ready"}); err != nil {
		t.Fatal(err)
	}

	ledgerData, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	runtimeAt := time.Now().Add(-31 * time.Minute).Format(time.RFC3339)
	reviewAt := time.Now().Add(-9 * time.Minute).Format(time.RFC3339)
	for index := range ledgerData.Tasks {
		switch ledgerData.Tasks[index].ID {
		case "RUNTIME":
			ledgerData.Tasks[index].History[0]["at"] = runtimeAt
			ledgerData.Tasks[index].LastObservation["at"] = runtimeAt
		case "REVIEW":
			ledgerData.Tasks[index].History[len(ledgerData.Tasks[index].History)-1]["at"] = reviewAt
			ledgerData.Tasks[index].LastObservation["at"] = reviewAt
		}
	}
	if err := saveLedger(ledger, &ledgerData); err != nil {
		t.Fatal(err)
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.BudgetPressure.EvidenceLabel != "local/static" {
		t.Fatalf("expected local/static budget pressure, got %#v", summary.BudgetPressure)
	}
	if summary.BudgetPressure.TasksExceeded != 1 || summary.BudgetPressure.TasksNearLimit != 1 || summary.BudgetPressure.TasksMissingBudget != 1 {
		t.Fatalf("unexpected budget pressure summary: %#v", summary.BudgetPressure)
	}
	byID := map[string]Observation{}
	for _, observation := range summary.Observations {
		byID[observation.ID] = observation
	}
	if byID["RUNTIME"].BudgetPressure == nil || byID["RUNTIME"].BudgetPressure.Status != "exceeded" {
		t.Fatalf("expected runtime budget exceeded, got %#v", byID["RUNTIME"].BudgetPressure)
	}
	if byID["REVIEW"].BudgetPressure == nil || byID["REVIEW"].BudgetPressure.Status != "near-limit" {
		t.Fatalf("expected review budget near-limit, got %#v", byID["REVIEW"].BudgetPressure)
	}
	if byID["LEGACY"].BudgetPressure == nil || byID["LEGACY"].BudgetPressure.Status != "missing" {
		t.Fatalf("expected legacy task missing budget pressure, got %#v", byID["LEGACY"].BudgetPressure)
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

func TestObserveRuntimeStatusReportCategories(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	activeWorker := filepath.Join(root, "active")
	dirtyWorker := filepath.Join(root, "dirty")
	reviewWorker := filepath.Join(root, "review")
	blockedWorker := filepath.Join(root, "blocked")
	cleanupWorker := filepath.Join(root, "cleanup")
	for _, worker := range []struct {
		path   string
		branch string
	}{
		{path: activeWorker, branch: "codex/active"},
		{path: dirtyWorker, branch: "codex/dirty"},
		{path: reviewWorker, branch: "codex/review"},
		{path: blockedWorker, branch: "codex/blocked"},
		{path: cleanupWorker, branch: "codex/cleanup"},
	} {
		git(t, project, "worktree", "add", "-q", "-b", worker.branch, worker.path, "HEAD")
	}

	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "ACTIVE", "--worktree", activeWorker, "--branch", "codex/active"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "PENDING", "--title", "Pending <setup>", "--pending-worktree-id", "pwt_runtime"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "DIRTY", "--worktree", dirtyWorker, "--branch", "codex/dirty"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "REVIEW", "--worktree", reviewWorker, "--branch", "codex/review"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "BLOCKED", "--worktree", blockedWorker, "--branch", "codex/expected"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "CLEANUP", "--worktree", cleanupWorker, "--branch", "codex/cleanup"}); err != nil {
		t.Fatal(err)
	}
	doneMissing := filepath.Join(root, "done-missing")
	if err := cmdRecordTask([]string{"--ledger", ledger, "--id", "DONE", "--worktree", doneMissing, "--branch", "codex/done"}); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(dirtyWorker, "dirty.txt"), []byte("dirty\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(reviewWorker, "review.txt"), []byte("review\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, reviewWorker, "add", "review.txt")
	git(t, reviewWorker, "commit", "-q", "-m", "review ready")

	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "merge", "--task-id", "CLEANUP", "--status", "merged", "--note", "merged but not cleaned"}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "cleanup", "--task-id", "DONE", "--status", "cleaned", "--note", "cleaned this cycle"}); err != nil {
		t.Fatal(err)
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	report := summary.RuntimeStatus
	if report.EvidenceLabel != "local/static" {
		t.Fatalf("expected local/static runtime report, got %#v", report)
	}
	if got := report.AvailableDispatchSlots; got != 0 {
		t.Fatalf("expected no available dispatch slots, got %d", got)
	}
	if len(report.ActiveWorkers) != 1 || report.ActiveWorkers[0].ID != "ACTIVE" {
		t.Fatalf("expected ACTIVE in activeWorkers, got %#v", report.ActiveWorkers)
	}
	if len(report.PendingSetup) != 1 || report.PendingSetup[0].ID != "PENDING" {
		t.Fatalf("expected PENDING in pendingSetup, got %#v", report.PendingSetup)
	}
	if report.PendingSetup[0].State.Setup != "pending-worktree-id" {
		t.Fatalf("expected pending setup state, got %#v", report.PendingSetup[0].State)
	}
	if len(report.DirtyUncommitted) != 1 || report.DirtyUncommitted[0].ID != "DIRTY" {
		t.Fatalf("expected DIRTY in dirtyUncommitted, got %#v", report.DirtyUncommitted)
	}
	if report.DirtyUncommitted[0].State.Diff != "dirty-uncommitted" {
		t.Fatalf("expected dirty diff state, got %#v", report.DirtyUncommitted[0].State)
	}
	if len(report.CompletedUnreviewed) != 1 || report.CompletedUnreviewed[0].ID != "REVIEW" {
		t.Fatalf("expected REVIEW in completedUnreviewed, got %#v", report.CompletedUnreviewed)
	}
	if state := report.CompletedUnreviewed[0].State; state.Diff != "clean-task-commit" || state.Review != "required" {
		t.Fatalf("expected clean commit review state, got %#v", state)
	}
	if len(report.Blockers) != 1 || report.Blockers[0].ID != "BLOCKED" {
		t.Fatalf("expected BLOCKED in blockers, got %#v", report.Blockers)
	}
	if len(report.CleanupNeeded) != 1 || report.CleanupNeeded[0].ID != "CLEANUP" {
		t.Fatalf("expected CLEANUP in cleanupNeeded, got %#v", report.CleanupNeeded)
	}
	if report.CleanupNeeded[0].State.Cleanup != "needed" {
		t.Fatalf("expected cleanup-needed state, got %#v", report.CleanupNeeded[0].State)
	}
	if len(report.RecentMergedOrCleaned) != 1 || report.RecentMergedOrCleaned[0].ID != "DONE" || report.RecentMergedOrCleaned[0].ObservedStatus != "cleaned" {
		t.Fatalf("expected DONE in recentMergedOrCleaned, got %#v", report.RecentMergedOrCleaned)
	}
	rendered := renderSummary(summary)
	for _, want := range []string{"## Runtime Status", "### Active Workers", "### Recent Merged Or Cleaned"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered summary to include %q:\n%s", want, rendered)
		}
	}

	statusJSON := captureStdout(t, func() error {
		return cmdStatus([]string{"--ledger", ledger, "--json"})
	})
	var statusPayload struct {
		RuntimeStatus          RuntimeStatusReport    `json:"runtimeStatus"`
		DispatchRecommendation DispatchRecommendation `json:"dispatchRecommendation"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &statusPayload); err != nil {
		t.Fatalf("expected status JSON, got %q: %v", statusJSON, err)
	}
	if len(statusPayload.RuntimeStatus.CleanupNeeded) != 1 || len(statusPayload.RuntimeStatus.RecentMergedOrCleaned) != 1 {
		t.Fatalf("expected status JSON runtime categories, got %#v", statusPayload.RuntimeStatus)
	}
	if !statusPayload.DispatchRecommendation.CapacityOnly || statusPayload.DispatchRecommendation.CapacityWarning == "" {
		t.Fatalf("expected dispatch capacity warning in status JSON, got %#v", statusPayload.DispatchRecommendation)
	}

	statusHTML := captureStdout(t, func() error {
		return cmdStatus([]string{"--ledger", ledger, "--html"})
	})
	for _, want := range []string{
		"<!doctype html>",
		"codex-orchestrator 本地静态状态页",
		"local/static evidence only",
		"集成区 / Integration",
		"活跃任务 / Active",
		"待 setup / Pending",
		"脏的未提交进度 / Dirty",
		"完成待审 / Review",
		"阻塞 / Blocked",
		"需要清理 / Cleanup",
		"最近合并/清理 / Recent",
		"派发槽",
		"证据标签 / Evidence Labels",
		"预算/审查压力 / Budget Pressure",
		"Pending &lt;setup&gt;",
		"id=PENDING",
		"pendingWorktreeId=pwt_runtime",
	} {
		if !strings.Contains(statusHTML, want) {
			t.Fatalf("expected status HTML to include %q:\n%s", want, statusHTML)
		}
	}
	if strings.Contains(statusHTML, "Pending <setup>") {
		t.Fatalf("expected status HTML to escape task title:\n%s", statusHTML)
	}
	statusHTMLPath := filepath.Join(root, "status.html")
	statusSummaryPath := filepath.Join(root, "status.md")
	if err := cmdStatus([]string{"--ledger", ledger, "--write-html", statusHTMLPath, "--write-summary", statusSummaryPath}); err != nil {
		t.Fatal(err)
	}
	writtenHTML, err := os.ReadFile(statusHTMLPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(writtenHTML), "codex-orchestrator 本地静态状态页") {
		t.Fatalf("expected written status HTML, got:\n%s", string(writtenHTML))
	}
	writtenSummary, err := os.ReadFile(statusSummaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(writtenSummary), "## Runtime Status") {
		t.Fatalf("expected written status summary, got:\n%s", string(writtenSummary))
	}
	writtenJSON, err := os.ReadFile(filepath.Join(root, "status.json"))
	if err != nil {
		t.Fatal(err)
	}
	var writtenPayload struct {
		RuntimeStatus          RuntimeStatusReport    `json:"runtimeStatus"`
		DispatchRecommendation DispatchRecommendation `json:"dispatchRecommendation"`
	}
	if err := json.Unmarshal(writtenJSON, &writtenPayload); err != nil {
		t.Fatalf("expected written status JSON, got %q: %v", string(writtenJSON), err)
	}
	if len(writtenPayload.RuntimeStatus.CleanupNeeded) != 1 || !writtenPayload.DispatchRecommendation.CapacityOnly {
		t.Fatalf("expected synchronized status JSON payload, got %#v", writtenPayload)
	}
	if err := cmdStatus([]string{"--ledger", ledger, "--json", "--html"}); err == nil {
		t.Fatal("expected mutually exclusive JSON/HTML status flags to fail")
	}
}

func TestStatusIncludesPackageSummary(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PKG-ACTIVE",
		"--title", "Package active worker",
		"--package-id", "PKG-CHECKOUT",
		"--pending-worktree-id", "pwt_pkg_active",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PKG-BLOCKED",
		"--title", "Package blocked worker",
		"--package-id", "PKG-CHECKOUT",
		"--pending-worktree-id", "pwt_pkg_blocked",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "blocker", "--task-id", "PKG-BLOCKED", "--status", "blocked", "--note", "needs owner input"}); err != nil {
		t.Fatal(err)
	}
	reviewFile := filepath.Join(root, "pi-review.md")
	if err := os.WriteFile(reviewFile, []byte("Verdict: pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdReviewImport([]string{"--ledger", ledger, "--package-id", "PKG-CHECKOUT", "--reviewer", "pi", "--file", reviewFile, "--task-id", "PKG-ACTIVE", "--status", "passed"}); err != nil {
		t.Fatal(err)
	}

	stored, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Tasks[0].PackageID != "PKG-CHECKOUT" {
		t.Fatalf("expected task package id to be recorded, got %#v", stored.Tasks[0])
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PackageSummary.Total != 1 || len(summary.PackageSummary.Rows) != 1 {
		t.Fatalf("expected one package summary row, got %#v", summary.PackageSummary)
	}
	pkg := summary.PackageSummary.Rows[0]
	if pkg.ID != "PKG-CHECKOUT" || pkg.Status != "blocked" || pkg.TaskCount != 2 {
		t.Fatalf("expected blocked package row, got %#v", pkg)
	}
	if !containsString(pkg.ActiveTaskIDs, "PKG-ACTIVE") || !containsString(pkg.BlockedTaskIDs, "PKG-BLOCKED") {
		t.Fatalf("expected active and blocked task ids, got %#v", pkg)
	}
	if pkg.ProgressLabel != "0/2 worker 已收口" || !strings.Contains(pkg.HumanSummary, "1 个阻塞") || pkg.ReviewStatus != "pi:passed" {
		t.Fatalf("expected package dashboard fields, got %#v", pkg)
	}

	statusJSON := captureStdout(t, func() error {
		return cmdStatus([]string{"--ledger", ledger, "--json"})
	})
	var statusPayload struct {
		PackageSummary PackageSummary `json:"packageSummary"`
		JobSummary     JobSummary     `json:"jobSummary"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &statusPayload); err != nil {
		t.Fatalf("expected status JSON, got %q: %v", statusJSON, err)
	}
	if len(statusPayload.PackageSummary.Rows) != 1 || statusPayload.PackageSummary.Rows[0].ID != "PKG-CHECKOUT" {
		t.Fatalf("expected status JSON package summary, got %#v", statusPayload.PackageSummary)
	}
	if len(statusPayload.JobSummary.Rows) != 2 || statusPayload.JobSummary.Rows[0].PackageID != "PKG-CHECKOUT" {
		t.Fatalf("expected job rows to include package id, got %#v", statusPayload.JobSummary.Rows)
	}

	rendered := renderSummary(summary)
	if !strings.Contains(rendered, "## 当前进度") || !strings.Contains(rendered, "当前主线: Pkg Checkout") || !strings.Contains(rendered, "派发模式: active：可以") || !strings.Contains(rendered, "需要你处理: 有 1 个阻塞项") {
		t.Fatalf("expected human-readable package progress in Markdown:\n%s", rendered)
	}
	if !strings.Contains(rendered, "## Package Summary") || !strings.Contains(rendered, "PKG-CHECKOUT") || !strings.Contains(rendered, "0/2 worker 已收口") || !strings.Contains(rendered, "pi:passed") {
		t.Fatalf("expected package summary in Markdown:\n%s", rendered)
	}
	statusHTML := captureStdout(t, func() error {
		return cmdStatus([]string{"--ledger", ledger, "--html"})
	})
	if !strings.Contains(statusHTML, "当前进度") || !strings.Contains(statusHTML, "Pkg Checkout") || !strings.Contains(statusHTML, "派发模式") || !strings.Contains(statusHTML, "active：可以") || !strings.Contains(statusHTML, "需要你处理") {
		t.Fatalf("expected human-readable package progress in HTML:\n%s", statusHTML)
	}
	if !strings.Contains(statusHTML, "功能包 / Packages") || !strings.Contains(statusHTML, "PKG-CHECKOUT") || !strings.Contains(statusHTML, "0/2 worker 已收口") || !strings.Contains(statusHTML, "external review: pi:passed") {
		t.Fatalf("expected package summary in HTML:\n%s", statusHTML)
	}

	unknownSummary := buildPackageSummary(
		[]Task{{ID: "PKG-REJECTED", PackageID: "PKG-UNKNOWN", Status: "rejected"}},
		[]Observation{{ID: "PKG-REJECTED", Status: "rejected", LastUpdatedAt: "2026-06-12T10:00:00+08:00"}},
		nil,
	)
	if len(unknownSummary.Rows) != 1 || unknownSummary.Rows[0].Status != "attention-needed" || !containsString(unknownSummary.Rows[0].OtherTaskIDs, "PKG-REJECTED") {
		t.Fatalf("expected unknown package task status to need attention, got %#v", unknownSummary.Rows)
	}
}

func TestLegacyTerminalUngroupedTasksDoNotTripLaneGuard(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "LEGACY-CLEANED",
		"--title", "Legacy cleaned task",
		"--pending-worktree-id", "pwt_legacy",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "cleanup", "--task-id", "LEGACY-CLEANED", "--status", "cleaned", "--note", "old task cleaned before package ids existed"}); err != nil {
		t.Fatal(err)
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PackageLaneGuard.Status != "passed" || len(summary.PackageLaneGuard.Warnings) != 0 {
		t.Fatalf("expected legacy terminal task to be ignored by lane guard, got %#v", summary.PackageLaneGuard)
	}
	if summary.JobSummary.LegacyTerminalUngrouped != 1 || len(summary.JobSummary.VisibleRows) != 0 {
		t.Fatalf("expected legacy task to be hidden from visible rows, got %#v", summary.JobSummary)
	}
	rendered := renderSummary(summary)
	if !strings.Contains(rendered, "legacyTerminalUngrouped") {
		t.Fatalf("expected rendered status to explain hidden legacy rows:\n%s", rendered)
	}

	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "UNGROUPED-ACTIVE",
		"--title", "Ungrouped active task",
		"--pending-worktree-id", "pwt_active",
	}); err != nil {
		t.Fatal(err)
	}
	summary, err = observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.PackageLaneGuard.Status != "warning" || summary.JobSummary.UngroupedNonTerminal != 1 {
		t.Fatalf("expected active ungrouped task to warn, got guard=%#v jobs=%#v", summary.PackageLaneGuard, summary.JobSummary)
	}
}

func TestInitCanWriteStarterTemplates(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project, "--write-templates"}); err != nil {
		t.Fatal(err)
	}
	templateNames := []string{"orchestration-policy.md", "package-plan.md", "project-map.md", "thread-map.md", "pulse-threads.md", "concepts.md", "inbox.md"}
	for _, name := range templateNames {
		path := filepath.Join(project, ".codex-orchestrator", name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected starter template %s: %v", name, err)
		}
		if !strings.Contains(string(data), "codex-orchestrator") && name == "orchestration-policy.md" {
			t.Fatalf("expected policy template to name codex-orchestrator, got:\n%s", string(data))
		}
		if name == "thread-map.md" && !strings.Contains(string(data), "Router") {
			t.Fatalf("expected thread map template to include Router role, got:\n%s", string(data))
		}
		if name == "pulse-threads.md" && !strings.Contains(string(data), "Project Pulse") {
			t.Fatalf("expected pulse template to include Project Pulse, got:\n%s", string(data))
		}
		if name == "concepts.md" && !strings.Contains(string(data), "Historical Pitfalls") {
			t.Fatalf("expected concepts template to include Historical Pitfalls, got:\n%s", string(data))
		}
		if name == "inbox.md" && !strings.Contains(string(data), "Intake") {
			t.Fatalf("expected inbox template to include Intake, got:\n%s", string(data))
		}
	}
	policy := filepath.Join(project, ".codex-orchestrator", "orchestration-policy.md")
	if err := os.WriteFile(policy, []byte("custom\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	created, skipped, err := writeStarterTemplates(project, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(created) != 0 || len(skipped) != len(templateNames) {
		t.Fatalf("expected all templates skipped, got created=%#v skipped=%#v", created, skipped)
	}
	data, err := os.ReadFile(policy)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "custom\n" {
		t.Fatalf("expected template not to overwrite without force, got:\n%s", string(data))
	}
}

func TestPackStatusReportsPackageCloseoutReadiness(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/package-worker", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PKG-WORKER",
		"--title", "Package worker",
		"--package-id", "PKG-CHECKOUT",
		"--worktree", worker,
		"--branch", "codex/package-worker",
		"--allowed", "README.md",
		"--gate", "go test ./...",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "README.md"), []byte("base\nworker\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "README.md")
	git(t, worker, "commit", "-q", "-m", "package worker")

	out := captureStdout(t, func() error {
		return cmdPackStatus([]string{"--ledger", ledger, "--package-id", "PKG-CHECKOUT", "--json"})
	})
	var report PackageCloseoutReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("expected package status JSON, got %q: %v", out, err)
	}
	if report.Status != "passed" || report.CloseoutDecision != "ready-for-orchestrator-acceptance" || report.Package == nil {
		t.Fatalf("expected package to be locally ready, got %#v", report)
	}
	if report.Acceptance.Command != "pack acceptance" || report.Acceptance.Status != "passed" {
		t.Fatalf("expected embedded acceptance report, got %#v", report.Acceptance)
	}
}

func TestPreflightReportsHandsOffReadinessWarnings(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() error {
		return cmdPreflight([]string{"--ledger", ledger, "--json"})
	})
	var report PreflightReport
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("expected preflight JSON, got %q: %v", out, err)
	}
	if report.Status != "warning" || report.EvidenceLabel != "local/static" {
		t.Fatalf("expected local/static warning preflight, got %#v", report)
	}
	checks := map[string]PreflightCheck{}
	for _, check := range report.Checks {
		checks[check.Name] = check
	}
	for _, name := range []string{"repo-git", "ledger", "heartbeat-gap", "watchdog", "project-map", "thread-map", "concepts", "inbox", "package-lane", "external-review-policy"} {
		if _, ok := checks[name]; !ok {
			t.Fatalf("expected preflight check %q in %#v", name, report.Checks)
		}
	}
	if checks["watchdog"].Status != "warning" || checks["heartbeat-gap"].Status != "warning" {
		t.Fatalf("expected watchdog and heartbeat warnings, got watchdog=%#v heartbeat=%#v", checks["watchdog"], checks["heartbeat-gap"])
	}

	reportPath := filepath.Join(root, "preflight.json")
	summaryPath := filepath.Join(root, "preflight.md")
	if err := cmdPreflight([]string{"--ledger", ledger, "--write-report", reportPath, "--write-summary", summaryPath}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected preflight JSON report: %v", err)
	}
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "codex-orchestrator preflight") || !strings.Contains(string(data), "watchdog") {
		t.Fatalf("expected preflight summary, got:\n%s", string(data))
	}
	if err := cmdPreflight([]string{"--ledger", ledger, "--fail-on-warning"}); err == nil {
		t.Fatal("expected --fail-on-warning to fail on warning status")
	}
	if err := cmdPreflight([]string{"--ledger", ledger, "--stale-after", "-1s"}); err == nil {
		t.Fatal("expected negative --stale-after to fail")
	}
}

func TestPackageSummaryReviewPolicyLaneGuardAndTimeline(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"PKG-A", "PKG-B", "PKG-C"} {
		if err := cmdRecordTask([]string{
			"--ledger", ledger,
			"--id", id,
			"--package-id", "PKG-CHECKOUT",
			"--pending-worktree-id", "pwt_" + strings.ToLower(id),
		}); err != nil {
			t.Fatal(err)
		}
		if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "cleanup", "--task-id", id, "--status", "cleaned", "--note", "package task cleaned"}); err != nil {
			t.Fatal(err)
		}
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.PackageSummary.Rows) != 1 {
		t.Fatalf("expected one package row, got %#v", summary.PackageSummary.Rows)
	}
	row := summary.PackageSummary.Rows[0]
	if !row.ReviewRequired || row.ReviewDecision != "one-reviewer" || row.ReviewNextAction == "" {
		t.Fatalf("expected package review requirement, got %#v", row)
	}
	if row.Status != "review-needed" {
		t.Fatalf("expected cleaned package to remain review-needed until external review is recorded, got %#v", row)
	}
	if summary.PackageLaneGuard.Status != "warning" || !strings.Contains(summary.PackageLaneGuard.RecommendedAction, "PKG-CHECKOUT") {
		t.Fatalf("expected lane guard warning for current package slot discipline, got %#v", summary.PackageLaneGuard)
	}
	if len(summary.Timeline) == 0 {
		t.Fatalf("expected timeline items, got %#v", summary.Timeline)
	}
	rendered := renderSummary(summary)
	for _, want := range []string{"## Package Lane Guard", "## Timeline", "one-reviewer", "Generate a package review pack"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered summary to include %q:\n%s", want, rendered)
		}
	}
	statusSummaryPath := filepath.Join(root, "status.md")
	if err := cmdStatus([]string{"--ledger", ledger, "--write-summary", statusSummaryPath}); err != nil {
		t.Fatal(err)
	}
	statusSummaryData, err := os.ReadFile(statusSummaryPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(statusSummaryData), "## Preflight") {
		t.Fatalf("expected status summary to include preflight:\n%s", string(statusSummaryData))
	}

	statusJSON := captureStdout(t, func() error {
		return cmdStatus([]string{"--ledger", ledger, "--json"})
	})
	var payload struct {
		PackageSummary   PackageSummary   `json:"packageSummary"`
		PackageLaneGuard PackageLaneGuard `json:"packageLaneGuard"`
		Preflight        *PreflightReport `json:"preflight"`
		Timeline         []TimelineItem   `json:"timeline"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &payload); err != nil {
		t.Fatalf("expected status JSON, got %q: %v", statusJSON, err)
	}
	if payload.Preflight == nil || len(payload.Timeline) == 0 || payload.PackageLaneGuard.Status == "" || !payload.PackageSummary.Rows[0].ReviewRequired {
		t.Fatalf("expected status JSON to expose new surfaces, got %#v", payload)
	}

	if packageReviewRisk(PackageStatusItem{ID: "preview-dashboard", TaskCount: 1}) != "low" {
		t.Fatal("expected preview package not to be promoted by bare pre substring")
	}
	if packageReviewRisk(PackageStatusItem{ID: "pre-prod-readback", TaskCount: 1}) != "high" {
		t.Fatal("expected pre-prod package to be high risk")
	}
	if statusClass("ready") != "ok" || statusClass("passed") != "ok" || statusClass("warning") != "warn" {
		t.Fatalf("expected status class mappings, got ready=%q passed=%q warning=%q", statusClass("ready"), statusClass("passed"), statusClass("warning"))
	}
}

func TestStatusAtAGlanceLinesCoverAttentionBranches(t *testing.T) {
	dirty := ObserveSummary{
		Integration: IntegrationState{Dirty: true},
		JobSummary:  JobSummary{Total: 1},
		RuntimeStatus: RuntimeStatusReport{
			AvailableDispatchSlots: 1,
			MaxConcurrency:         2,
		},
	}
	dirtyLines := strings.Join(statusAtAGlanceLines(dirty), "\n")
	if !strings.Contains(dirtyLines, "集成区有未提交变化") || strings.Contains(dirtyLines, "建议动作:") {
		t.Fatalf("expected dirty status without recommendation line, got:\n%s", dirtyLines)
	}

	activeMissed := ObserveSummary{
		Integration: IntegrationState{},
		HeartbeatStatus: &HeartbeatStatus{
			Status:              "missed",
			Gap:                 "5h0m0s",
			EstimatedMissedRuns: 14,
		},
		ReviewPressure: ReviewPressure{
			Active:       1,
			PendingSetup: 1,
		},
		RuntimeStatus: RuntimeStatusReport{
			AvailableDispatchSlots: 0,
			MaxConcurrency:         2,
		},
		RecommendedActions: []string{"Active tasks are within concurrency limit; continue monitoring."},
	}
	activeLines := strings.Join(statusAtAGlanceLines(activeMissed), "\n")
	for _, want := range []string{"heartbeat 可能漏跑", "派发模式: active：可以", "正在跑: active=1，pending setup=1", "等待当前 worker", "不要为了填满并发槽派无关模块任务", "需要你处理: 无。"} {
		if !strings.Contains(activeLines, want) {
			t.Fatalf("expected at-a-glance lines to include %q, got:\n%s", want, activeLines)
		}
	}
	if strings.Contains(activeLines, "建议动作:") || strings.Contains(activeLines, "并发槽已满") {
		t.Fatalf("expected human-facing lines to hide machine dispatch-slot jargon, got:\n%s", activeLines)
	}
}

func TestStatusLoadsRepoHeartbeatReportIntoHumanSummary(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	stateDir := filepath.Join(project, defaultStateDir)
	report := ObserveSummary{
		ObservedAt: "2026-06-12T18:00:00+08:00",
		HeartbeatStatus: &HeartbeatStatus{
			EvidenceLabel:       "local/static",
			Status:              "missed",
			CurrentHeartbeatAt:  "2026-06-12T18:00:00+08:00",
			ExpectedInterval:    "20m0s",
			MissedAfter:         "45m0s",
			Gap:                 "4h25m31s",
			EstimatedMissedRuns: 12,
			Note:                "Possible missed heartbeat.",
		},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "heartbeat-report.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	summary, err := observeWithOptions(ledger, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if summary.HeartbeatStatus == nil || summary.HeartbeatStatus.Status != "missed" {
		t.Fatalf("expected observe/status to load repo heartbeat report, got %#v", summary.HeartbeatStatus)
	}
	loaded, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	rendered := renderStatusHTML(summary, loaded, ledger)
	for _, want := range []string{"heartbeat 漏跑提示", "gap=4h25m31s", "estimatedMissedRuns=12", "heartbeat / watchdog"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected status HTML to include %q:\n%s", want, rendered)
		}
	}
}

func TestHumanProgressSummaryBranches(t *testing.T) {
	if got := humanIdentifier("TF-PRE-STAFF_RBAC-API-LOCAL"); got != "Staff RBAC API" {
		t.Fatalf("expected humanized identifier, got %q", got)
	}
	if got := humanIdentifier("TF-PRE-LOCAL"); got != "TF-PRE-LOCAL" {
		t.Fatalf("expected all-filtered identifier to fall back to raw id, got %q", got)
	}
	if got := humanStatusLabel("cleanup-needed"); got != "待清理" {
		t.Fatalf("expected cleanup label, got %q", got)
	}
	if got := humanObservationNote(RuntimeStatusItem{State: LocalTaskState{Diff: "clean-no-task-commit"}}); got != "worker 已创建，但还没有可验收 commit" {
		t.Fatalf("expected clean-no-task-commit human note, got %q", got)
	}

	dirty := buildHumanProgressSummary(ObserveSummary{
		Integration: IntegrationState{Dirty: true},
		RuntimeStatus: RuntimeStatusReport{
			EvidenceLabel: "local/static",
		},
	})
	if dirty.Headline != "本地有未分类改动" || !strings.Contains(dirty.NextStep, "先区分业务代码改动") {
		t.Fatalf("expected dirty repo branch, got %#v", dirty)
	}

	review := buildHumanProgressSummary(ObserveSummary{
		ReviewPressure: ReviewPressure{ReviewNeeded: 1},
		RuntimeStatus: RuntimeStatusReport{
			EvidenceLabel: "proxy",
		},
	})
	if review.Headline != "有 worker 等待验收" || !strings.Contains(strings.Join(review.Risks, "\n"), "不要自动把它当成 direct") {
		t.Fatalf("expected review-needed branch and guarded non-local evidence wording, got %#v", review)
	}

	cleanup := buildHumanProgressSummary(ObserveSummary{
		ReviewPressure: ReviewPressure{CleanupNeeded: 1},
	})
	if cleanup.Headline != "有已收口任务待清理" || !strings.Contains(cleanup.NextStep, "cleanup") {
		t.Fatalf("expected cleanup branch, got %#v", cleanup)
	}

	empty := buildHumanProgressSummary(ObserveSummary{})
	if empty.Headline != "当前空闲" || empty.CurrentLane != "暂无功能包" {
		t.Fatalf("expected empty ledger branch, got %#v", empty)
	}

	lane := currentLaneName(ObserveSummary{
		PackageSummary: PackageSummary{Rows: []PackageStatusItem{
			{ID: "staff-rbac-old-review", Status: "review-needed", TaskCount: 4, LatestUpdatedAt: "2026-06-12T08:00:00+08:00"},
			{ID: "pre-web-runtime-rewrite-unblock", Status: "active", TaskCount: 1, LatestUpdatedAt: "2026-06-12T09:00:00+08:00"},
		}},
	})
	if !strings.Contains(lane, "Web Runtime Rewrite Unblock") || strings.Contains(lane, "Staff") {
		t.Fatalf("expected current lane to prefer active package over old review debt, got %q", lane)
	}

	split := PackageSummary{Rows: []PackageStatusItem{
		{ID: "staff-rbac-old-review", Status: "review-needed", TaskCount: 4, LatestUpdatedAt: "2026-06-12T08:00:00+08:00"},
		{ID: "pre-web-runtime-rewrite-unblock", Status: "active", TaskCount: 1, LatestUpdatedAt: "2026-06-12T09:00:00+08:00"},
	}}
	current, ok := selectCurrentPackageLane(split.Rows)
	if !ok || current.ID != "pre-web-runtime-rewrite-unblock" {
		t.Fatalf("expected active current lane, got row=%#v ok=%t", current, ok)
	}
	split.CurrentLane = optionalPackageStatusItem(current, ok)
	split.HistoricalReviewDebt = packageRowsWithStatusExcept(split.Rows, "review-needed", current.ID)
	if split.CurrentLane == nil || split.CurrentLane.ID != "pre-web-runtime-rewrite-unblock" || len(split.HistoricalReviewDebt) != 1 || split.HistoricalReviewDebt[0].ID != "staff-rbac-old-review" {
		t.Fatalf("expected current lane and historical review debt split, got %#v", split)
	}

	rec := buildDispatchRecommendation(
		"active",
		IntegrationState{},
		ReviewPressure{Active: 1, AvailableSlots: 1},
		split,
		PackageLaneGuard{EvidenceLabel: "local/static", Status: "warning", CurrentPackageID: "pre-web-runtime-rewrite-unblock"},
	)
	if rec.Recommended || rec.Reason != "active or pending worker exists" || !strings.Contains(rec.NextAction, "current package worker") {
		t.Fatalf("expected active worker to suppress filler dispatch, got %#v", rec)
	}
}

func TestIntegrationStateDirOnlyAndDrainSlotDisplay(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	stateDir := filepath.Join(project, defaultStateDir)
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(stateDir, "status.md"), []byte("local status\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	integration := inspectIntegration(project)
	if integration.Dirty || !integration.StateDirChanges || !integration.StateDirOnly {
		t.Fatalf("expected state-dir-only integration state, got %#v", integration)
	}
	if integration.BusinessGitStatus != "" || !strings.Contains(integration.StateDirStatus, defaultStateDir) {
		t.Fatalf("expected state dir status separated from business status, got %#v", integration)
	}
	summary := ObserveSummary{
		DispatchMode: "drain",
		Integration:  integration,
		RuntimeStatus: RuntimeStatusReport{
			EvidenceLabel:          "local/static",
			AvailableDispatchSlots: 2,
			MaxConcurrency:         2,
		},
	}
	label, className := dispatchSlotDisplay(summary)
	if label != "队列已停，不派发" || className != "ok" {
		t.Fatalf("expected terminal drain dispatch slot label, got label=%q class=%q", label, className)
	}
	activeDrain := summary
	activeDrain.ReviewPressure.Active = 1
	label, className = dispatchSlotDisplay(activeDrain)
	if !strings.Contains(label, "排空中，不派发") || className != "warn" {
		t.Fatalf("expected active drain dispatch slot warning, got label=%q class=%q", label, className)
	}
	if got := humanDispatchModeLabel("drain"); got != "drain / 只收口，不派发" {
		t.Fatalf("expected human drain label, got %q", got)
	}
	progress := buildHumanProgressSummary(summary)
	if progress.Headline != "队列已停，不再派发" || !strings.Contains(progress.NextStep, "run-mode=drain") {
		t.Fatalf("expected terminal drain human summary, got %#v", progress)
	}
	if got := humanDispatchModeExplanation("paused"); !strings.Contains(got, "暂停编排") || !strings.Contains(got, "不派发") {
		t.Fatalf("expected human paused explanation, got %q", got)
	}
	risks := strings.Join(humanRiskLines(summary), "\n")
	for _, want := range []string{defaultStateDir + "/ 有本地编排状态变化", "run-mode=drain", "不应该继续派发"} {
		if !strings.Contains(risks, want) {
			t.Fatalf("expected risk %q in:\n%s", want, risks)
		}
	}
}

func TestGitOutputDisablesQuotePathForUnicodePaths(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	dir := filepath.Join(project, "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "中文路径.md"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, project, "add", "docs/中文路径.md")
	git(t, project, "commit", "-q", "-m", "add unicode path")
	out, err := gitOutput(project, "diff", "--name-status", base+"..HEAD")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "docs/中文路径.md") || strings.Contains(out, "\\344") {
		t.Fatalf("expected unquoted unicode path, got %q", out)
	}
}

func TestObserveBlockedPendingSetupAndDrainMode(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdDispatchRecord([]string{
		"--ledger", ledger,
		"--task-id", "SETUP-FAILED",
		"--pending-worktree-id", "pwt_failed",
		"--branch", "codex/setup-failed",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{
		"--ledger", ledger,
		"--type", "setup",
		"--task-id", "SETUP-FAILED",
		"--status", "blocked",
		"--note", "fatal: invalid reference",
	}); err != nil {
		t.Fatal(err)
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(summary.RuntimeStatus.PendingSetup) != 0 {
		t.Fatalf("blocked setup failure must not remain pending setup: %#v", summary.RuntimeStatus.PendingSetup)
	}
	if len(summary.RuntimeStatus.Blockers) != 1 || summary.RuntimeStatus.Blockers[0].ID != "SETUP-FAILED" {
		t.Fatalf("expected setup failure in blockers, got %#v", summary.RuntimeStatus.Blockers)
	}
	if summary.RuntimeStatus.Blockers[0].State.Setup != "blocked" {
		t.Fatalf("expected blocked setup state, got %#v", summary.RuntimeStatus.Blockers[0].State)
	}

	ledgerData, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	ledgerData.Tasks = nil
	if err := saveLedger(ledger, &ledgerData); err != nil {
		t.Fatal(err)
	}
	if err := cmdRunModeSet([]string{"--ledger", ledger, "--dispatch-mode", "drain", "--note", "finish current batch only"}); err != nil {
		t.Fatal(err)
	}
	summary, err = observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.DispatchMode != "drain" {
		t.Fatalf("expected drain dispatch mode, got %q", summary.DispatchMode)
	}
	if summary.OverallStatus != "dispatch-draining" {
		t.Fatalf("expected dispatch-draining status, got %q actions=%v", summary.OverallStatus, summary.RecommendedActions)
	}
	if strings.Contains(strings.Join(summary.RecommendedActions, "\n"), "dispatch the next safe roadmap task") {
		t.Fatalf("drain mode should not recommend dispatch, got %v", summary.RecommendedActions)
	}
}

func TestHeartbeatReportsMissedWakeupGap(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	events := eventsPathForLedger(ledger)
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := appendEvent(events, map[string]any{
		"at":     time.Now().Add(-5 * time.Hour).Format(time.RFC3339),
		"type":   "heartbeat",
		"status": "quiet",
		"note":   "old heartbeat",
	}); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "heartbeat-report.json")
	if err := cmdHeartbeat([]string{
		"--ledger", ledger,
		"--events", events,
		"--interval", "20m",
		"--missed-after", "45m",
		"--count", "1",
		"--write-report", reportPath,
		"--json",
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var summary ObserveSummary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if summary.HeartbeatStatus == nil {
		t.Fatal("expected heartbeat status in report")
	}
	if summary.HeartbeatStatus.Status != "missed" {
		t.Fatalf("expected missed heartbeat status, got %#v", summary.HeartbeatStatus)
	}
	if summary.HeartbeatStatus.EstimatedMissedRuns < 1 {
		t.Fatalf("expected estimated missed runs, got %#v", summary.HeartbeatStatus)
	}
}

func TestObserveJobSummaryAndProjectMap(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := os.MkdirAll(filepath.Join(project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "CODEBASE_MAP.md"), []byte("# Codebase map\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(project, ".codex-orchestrator"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".codex-orchestrator", "thread-map.md"), []byte("# Thread Map\n\nRouter\nInbox\nPulse\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".codex-orchestrator", "concepts.md"), []byte("# Concepts\n\nGlossary\nDecisions\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".codex-orchestrator", "inbox.md"), []byte("# Inbox\n\nIntake\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	worker := filepath.Join(root, "worker")
	git(t, project, "worktree", "add", "-q", "-b", "codex/status", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "STATUS-ROW",
		"--title", "Status row",
		"--worktree", worker,
		"--branch", "codex/status",
	}); err != nil {
		t.Fatal(err)
	}

	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.JobSummary.EvidenceLabel != "local/static" || summary.JobSummary.Total != 1 {
		t.Fatalf("expected local/static one-job summary, got %#v", summary.JobSummary)
	}
	if len(summary.JobSummary.Rows) != 1 || summary.JobSummary.Rows[0].ID != "STATUS-ROW" || summary.JobSummary.Rows[0].Title != "Status row" {
		t.Fatalf("expected job summary row, got %#v", summary.JobSummary.Rows)
	}
	if summary.JobSummary.Counts["active"] != 1 {
		t.Fatalf("expected active job count, got %#v", summary.JobSummary.Counts)
	}
	if summary.ProjectMap.Status != "present" || summary.ProjectMap.Path != filepath.Join("docs", "CODEBASE_MAP.md") {
		t.Fatalf("expected detected project map, got %#v", summary.ProjectMap)
	}
	if summary.ThreadMap.Status != "present" || summary.ThreadMap.Path != filepath.Join(".codex-orchestrator", "thread-map.md") {
		t.Fatalf("expected detected thread map, got %#v", summary.ThreadMap)
	}
	if summary.Concepts.Status != "present" || summary.Concepts.Path != filepath.Join(".codex-orchestrator", "concepts.md") {
		t.Fatalf("expected detected concepts, got %#v", summary.Concepts)
	}
	if summary.Inbox.Status != "present" || summary.Inbox.Path != filepath.Join(".codex-orchestrator", "inbox.md") {
		t.Fatalf("expected detected inbox, got %#v", summary.Inbox)
	}
	rendered := renderSummary(summary)
	for _, want := range []string{"## Job Summary", "| `STATUS-ROW` | `` | `active`", "## Project Map", "docs/CODEBASE_MAP.md", "## Thread Map", ".codex-orchestrator/thread-map.md", "## Concepts", ".codex-orchestrator/concepts.md", "## Inbox", ".codex-orchestrator/inbox.md"} {
		if !strings.Contains(rendered, want) {
			t.Fatalf("expected rendered summary to include %q:\n%s", want, rendered)
		}
	}

	statusJSON := captureStdout(t, func() error {
		return cmdStatus([]string{"--ledger", ledger, "--json"})
	})
	var statusPayload struct {
		JobSummary JobSummary       `json:"jobSummary"`
		ProjectMap ProjectMapStatus `json:"projectMap"`
		ThreadMap  ThreadMapStatus  `json:"threadMap"`
		Concepts   ConceptsStatus   `json:"concepts"`
		Inbox      InboxStatus      `json:"inbox"`
	}
	if err := json.Unmarshal([]byte(statusJSON), &statusPayload); err != nil {
		t.Fatalf("expected status JSON, got %q: %v", statusJSON, err)
	}
	if statusPayload.JobSummary.Total != 1 || statusPayload.ProjectMap.Status != "present" || statusPayload.ThreadMap.Status != "present" || statusPayload.Concepts.Status != "present" || statusPayload.Inbox.Status != "present" {
		t.Fatalf("expected status JSON jobSummary/projectMap/threadMap/concepts/inbox, got %#v", statusPayload)
	}
}

func TestTerminalReleasedAndCleanedTasksAreQuietAfterCleanup(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		id     string
		status string
	}{
		{id: "RELEASED", status: "released"},
		{id: "CLEANED", status: "cleaned"},
	} {
		missing := filepath.Join(root, strings.ToLower(tc.id))
		if err := cmdRecordTask([]string{"--ledger", ledger, "--id", tc.id, "--worktree", missing, "--branch", "codex/" + strings.ToLower(tc.id)}); err != nil {
			t.Fatal(err)
		}
		if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", tc.status, "--task-id", tc.id, "--status", tc.status, "--note", "post-merge terminal state"}); err != nil {
			t.Fatal(err)
		}
	}

	ledgerData, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	staleAt := time.Now().Add(-2 * time.Hour).Format(time.RFC3339)
	for index := range ledgerData.Tasks {
		ledgerData.Tasks[index].LastObservation["at"] = staleAt
	}
	if err := saveLedger(ledger, &ledgerData); err != nil {
		t.Fatal(err)
	}

	summary, err := observeWithOptions(ledger, 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	actions := map[string]string{}
	for _, observation := range summary.Observations {
		statuses[observation.ID] = observation.Status
		actions[observation.ID] = observation.Action
	}
	for _, tc := range []struct {
		id     string
		status string
	}{
		{id: "RELEASED", status: "released"},
		{id: "CLEANED", status: "cleaned"},
	} {
		if statuses[tc.id] != tc.status {
			t.Fatalf("expected %s status %q, got %q", tc.id, tc.status, statuses[tc.id])
		}
		if actions[tc.id] != "quiet" {
			t.Fatalf("expected %s to be quiet, got %q", tc.id, actions[tc.id])
		}
	}
	if summary.ReviewPressure.PendingSetup != 0 || summary.ReviewPressure.Stale != 0 || summary.ReviewPressure.Blocked != 0 || summary.ReviewPressure.CleanupNeeded != 0 {
		t.Fatalf("expected terminal tasks to create no setup/stale/blocker pressure, got %#v", summary.ReviewPressure)
	}
	if summary.Counts["pending-setup"] != 0 || summary.Counts["stale-needs-inspection"] != 0 || summary.Counts["blocked"] != 0 || summary.Counts["cleanup-needed"] != 0 {
		t.Fatalf("expected no pressure statuses for terminal tasks, got %#v", summary.Counts)
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

func TestRunPRReviewerRoutineChecksAllowedWriteSetAndSignals(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/checklist", worker, "HEAD")
	if err := os.MkdirAll(filepath.Join(worker, "docs", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "CHECKLIST",
		"--worktree", worker,
		"--branch", "codex/checklist",
		"--base-commit", base,
		"--allowed", "docs/reviews/**",
		"--forbidden", "secrets/**",
		"--gate", "go test ./...",
	}); err != nil {
		t.Fatal(err)
	}
	reviewPath := filepath.Join(worker, "docs", "reviews", "self-review-evidence.md")
	if err := os.WriteFile(reviewPath, []byte("Self-review: local evidence only.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "docs/reviews/self-review-evidence.md")
	git(t, worker, "commit", "-q", "-m", "add review evidence")

	report, err := runPRReviewerRoutine(ledger, "CHECKLIST")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected passed checklist report, got %#v", report)
	}
	joined := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Automated review checklist: ledger writeSet",
		"all changed paths fit the ledger allowed writeSet",
		"no changed paths matched the ledger forbidden writeSet",
		"review artifact signal found",
		"self-review or handoff filename signal found",
		"suggested narrow gate(s) from ledger task: go test ./...",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected checklist evidence %q in:\n%s", want, joined)
		}
	}
	if report.NeedsHuman {
		t.Fatalf("expected no checklist warning requiring human for complete local signals, got %#v", report)
	}
}

func TestRunPRReviewerRoutineFailsOnForbiddenWriteSet(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/forbidden", worker, "HEAD")
	if err := os.MkdirAll(filepath.Join(worker, "secrets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "FORBIDDEN",
		"--worktree", worker,
		"--branch", "codex/forbidden",
		"--base-commit", base,
		"--allowed", "docs/**",
		"--forbidden", "secrets/**",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "secrets", "token.txt"), []byte("placeholder\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "secrets/token.txt")
	git(t, worker, "commit", "-q", "-m", "touch forbidden path")

	report, err := runPRReviewerRoutine(ledger, "FORBIDDEN")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" {
		t.Fatalf("expected failed forbidden-path report, got %#v", report)
	}
	joined := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"changed path(s) outside ledger allowed writeSet: secrets/token.txt",
		"changed path(s) match ledger forbidden writeSet: secrets/token.txt",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected forbidden checklist evidence %q in:\n%s", want, joined)
		}
	}
}

func TestPackMergeReadinessWritesStandardLocalReport(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	reportPath := filepath.Join(root, "reports", "merge-readiness.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/pack", worker, "HEAD")
	if err := os.MkdirAll(filepath.Join(worker, "cmd", "codex-orchestrator"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(worker, "docs", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PACK",
		"--title", "Merge readiness pack",
		"--thread-id", "thread_pack",
		"--pending-worktree-id", "pwt_pack",
		"--worktree", worker,
		"--branch", "codex/pack",
		"--base-commit", base,
		"--allowed", "cmd/codex-orchestrator/**",
		"--allowed", "docs/**",
		"--forbidden", ".github/workflows/**",
		"--gate", "go test ./...",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "cmd", "codex-orchestrator", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	review := "Self-review: local/static evidence only; not runtime or production proof.\n"
	if err := os.WriteFile(filepath.Join(worker, "docs", "reviews", "self-review-evidence.md"), []byte(review), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "cmd/codex-orchestrator/main.go", "docs/reviews/self-review-evidence.md")
	git(t, worker, "commit", "-q", "-m", "pack-ready change")
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "review", "--task-id", "PACK", "--status", "completed-unreviewed", "--note", "ready for pack"}); err != nil {
		t.Fatal(err)
	}

	if err := cmdPackMergeReadiness([]string{"--ledger", ledger, "--task-id", "PACK", "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report MergeReadinessPack
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || report.NeedsHuman {
		t.Fatalf("expected passed pack without missing local signals, got %#v", report)
	}
	if report.Task.ID != "PACK" || report.Task.Title != "Merge readiness pack" || report.Task.ThreadID != "thread_pack" || report.Task.PendingWorktreeID != "pwt_pack" {
		t.Fatalf("expected task metadata in pack, got %#v", report.Task)
	}
	if report.Task.ActualBranch != "codex/pack" || report.ObservedStatus != "completed-unreviewed" {
		t.Fatalf("expected branch and observed status, got task=%#v observed=%q", report.Task, report.ObservedStatus)
	}
	if report.CommitCountAfterBase == nil || *report.CommitCountAfterBase != 1 {
		t.Fatalf("expected one commit after base, got %#v", report.CommitCountAfterBase)
	}
	if report.PathCheck.Status != "passed" || report.DiffCheck.Status != "passed" {
		t.Fatalf("expected path and diff checks to pass, got path=%#v diff=%#v", report.PathCheck, report.DiffCheck)
	}
	changed := strings.Join(report.ChangedPaths, "\n")
	for _, want := range []string{"cmd/codex-orchestrator/main.go", "docs/reviews/self-review-evidence.md"} {
		if !strings.Contains(changed, want) {
			t.Fatalf("expected changed path %q in %#v", want, report.ChangedPaths)
		}
	}
	if len(report.Signals.ReviewDocs) == 0 || len(report.Signals.Artifacts) == 0 || len(report.Signals.SelfReview) == 0 || len(report.Signals.EvidenceLabel) == 0 || len(report.Signals.DocsDrift) == 0 {
		t.Fatalf("expected complete merge-readiness signals, got %#v", report.Signals)
	}
	if got := strings.Join(report.SuggestedGates, "\n"); !strings.Contains(got, "go test ./...") || !strings.Contains(got, "docs-drift-checker") || !strings.Contains(got, "evidence-label-auditor") {
		t.Fatalf("expected recorded and suggested gates, got %#v", report.SuggestedGates)
	}
	if !strings.Contains(report.Boundary, "local/static review evidence only") || !strings.Contains(report.Boundary, "does not merge, push, cleanup, dispatch, or edit git state") {
		t.Fatalf("expected conservative evidence boundary, got %q", report.Boundary)
	}
	if report.AcceptanceReport.Decision != "review-ready" || !strings.Contains(report.AcceptanceReport.NextAction, "separate accept/reject decision") {
		t.Fatalf("expected review-ready acceptance report, got %#v", report.AcceptanceReport)
	}
	if report.ClaimVerification.Status != "passed" || report.AcceptanceReport.ClaimVerification.Status != "passed" {
		t.Fatalf("expected passed claim verification in pack and acceptance report, got pack=%#v acceptance=%#v", report.ClaimVerification, report.AcceptanceReport.ClaimVerification)
	}
	if len(report.ClaimVerification.Checks) == 0 {
		t.Fatalf("expected claim verification checks, got %#v", report.ClaimVerification)
	}
	if len(report.AuthorizationMatrix) == 0 || !containsAuthorizationStatus(report.AuthorizationMatrix, "merge", "requires-separate-orchestrator-acceptance") {
		t.Fatalf("expected merge authorization boundary, got %#v", report.AuthorizationMatrix)
	}
	if report.LiveProofGate.Status == "" || report.LiveProofGate.Boundary == "" || report.LiveProofGate.WaiverRequired {
		t.Fatalf("expected conservative non-waiver live proof gate, got %#v", report.LiveProofGate)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected only local evidence, got %#v", report.Evidence)
	}
}

func TestRecordTaskStoresConstraintStack(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "CONSTRAINT-SNAPSHOT",
		"--pending-worktree-id", "local:constraint",
		"--package-id", "checkout-package",
		"--allowed", "apps/customer-web/**",
		"--forbidden", "services/payment/**",
		"--gate", "npm test -- --runInBand",
		"--constraint", "do not touch payment",
		"--authority", "AGENTS.md",
		"--user-instruction", "Continue checkout package only.",
		"--evidence-boundary", "local/static only",
		"--package-switch-reason", "same-package-continuation",
	}); err != nil {
		t.Fatal(err)
	}
	updated, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	stack := updated.Tasks[0].ConstraintStack
	if stack == nil {
		t.Fatal("expected constraint stack")
	}
	if stack.PackageID != "checkout-package" || stack.PackageSwitchReason != "same-package-continuation" || stack.EvidenceBoundary != "local/static only" {
		t.Fatalf("unexpected constraint stack: %#v", stack)
	}
	if got := strings.Join(stack.AllowedPaths, "\n"); !strings.Contains(got, "apps/customer-web/**") {
		t.Fatalf("expected allowed path in constraint stack, got %#v", stack.AllowedPaths)
	}
}

func TestMisalignmentRecordReportAndObserveTrustRisk(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdMisalignmentRecord([]string{
		"--repo", project,
		"--category", "inaccurate-self-report",
		"--source", "user-pushback",
		"--task-id", "UNKNOWN",
		"--severity", "high",
		"--note", "Worker claimed tests passed without command output.",
	}); err != nil {
		t.Fatal(err)
	}
	report, err := buildMisalignmentInsightsReport(project, ledger)
	if err != nil {
		t.Fatal(err)
	}
	if report.Total != 1 || report.Status != "blocked" {
		t.Fatalf("expected one high-risk blocked report, got %#v", report)
	}
	if report.TrustRisk.Status != "blocked" || len(report.TrustRisk.Items) == 0 {
		t.Fatalf("expected blocked trust risk, got %#v", report.TrustRisk)
	}
	summary, err := observe(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TrustRisk.Status != "blocked" {
		t.Fatalf("expected observe trust risk blocked, got %#v", summary.TrustRisk)
	}
}

func TestPolicyAuditFlagsClaimSelfReportWithoutVerification(t *testing.T) {
	findings := auditOrchestrationPolicyText("docs/reviews/self-report.md", "If the worker says complete, trust it and merge directly without checking diff, gates, or command output.")
	counts := countPolicyAuditFindings(findings)
	if counts[policyRuleClaimVerification] != 1 {
		t.Fatalf("expected OPA010 hit, got findings=%#v counts=%#v", findings, counts)
	}
}

func TestPackMergeReadinessAllowsStateDirOnlyChanges(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/state-dir-only", worker, "HEAD")
	if err := os.WriteFile(filepath.Join(worker, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "feature.txt")
	git(t, worker, "commit", "-q", "-m", "feature ready")
	if err := os.MkdirAll(filepath.Join(worker, ".codex-orchestrator"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, ".codex-orchestrator", "status.md"), []byte("local status\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PACK-STATE-DIR",
		"--worktree", worker,
		"--branch", "codex/state-dir-only",
		"--base-commit", base,
		"--allowed", "feature.txt",
		"--forbidden", "secrets/**",
		"--gate", "git diff --check",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "review", "--task-id", "PACK-STATE-DIR", "--status", "completed-unreviewed", "--note", "ready"}); err != nil {
		t.Fatal(err)
	}

	report, err := buildMergeReadinessPack(project, ledger, "PACK-STATE-DIR")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" {
		t.Fatalf("expected state-dir-only changes not to fail merge-readiness, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, ".codex-orchestrator/ changes are local orchestration state only") {
		t.Fatalf("expected state-dir-only local evidence, got:\n%s", local)
	}
	if got := strings.Join(report.ResidualRisks, "\n"); !strings.Contains(got, "local/static orchestration state changes") {
		t.Fatalf("expected residual risk for local state dir, got %#v", report.ResidualRisks)
	}
}

func TestPackMergeReadinessFailsForbiddenPathCheck(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/pack-forbidden", worker, "HEAD")
	if err := os.MkdirAll(filepath.Join(worker, "secrets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "PACK-FORBIDDEN",
		"--worktree", worker,
		"--branch", "codex/pack-forbidden",
		"--base-commit", base,
		"--allowed", "docs/**",
		"--forbidden", "secrets/**",
		"--gate", "go test ./...",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "secrets", "token.txt"), []byte("placeholder\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "secrets/token.txt")
	git(t, worker, "commit", "-q", "-m", "touch forbidden path")

	report, err := buildMergeReadinessPack(project, ledger, "PACK-FORBIDDEN")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "failed" || !report.NeedsHuman {
		t.Fatalf("expected failed pack needing human review, got %#v", report)
	}
	if report.PathCheck.Status != "failed" {
		t.Fatalf("expected failed path check, got %#v", report.PathCheck)
	}
	if got := strings.Join(report.PathCheck.OutsideAllowed, "\n"); !strings.Contains(got, "secrets/token.txt") {
		t.Fatalf("expected outside-allowed path, got %#v", report.PathCheck)
	}
	if got := strings.Join(report.PathCheck.ForbiddenHits, "\n"); !strings.Contains(got, "secrets/token.txt") {
		t.Fatalf("expected forbidden hit, got %#v", report.PathCheck)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 {
		t.Fatalf("expected no direct/proxy evidence, got %#v", report.Evidence)
	}
}

func TestPathMatcherSupportsDoubleStarAcrossDirectories(t *testing.T) {
	path := "services/cloud-backend/src/main/kotlin/com/examplepos/waitlist/WaitlistReservationLifecycle.kt"
	allowed := []string{"services/cloud-backend/src/main/**/waitlist*/**"}
	forbidden := []string{"services/payment/**"}
	check := evaluateMergeReadinessPathCheck(Task{WriteSet: map[string][]string{
		"allowed":   allowed,
		"forbidden": forbidden,
	}}, []string{path})
	if check.Status != "passed" || len(check.OutsideAllowed) != 0 {
		t.Fatalf("expected double-star allowed path check to pass, got %#v", check)
	}
	if got := strings.Join(check.AllowedMatches, "\n"); !strings.Contains(got, path+" <= "+allowed[0]) {
		t.Fatalf("expected matched allowed rule in report, got %#v", check.AllowedMatches)
	}
	if !repoPathMatchesPattern(path, allowed[0]) {
		t.Fatalf("expected repoPathMatchesPattern to support ** across directories")
	}
}

func TestPackReviewWritesPortablePackageReviewPack(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	outputDir := filepath.Join(root, "review-pack")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/package-review", worker, "HEAD")
	if err := os.MkdirAll(filepath.Join(worker, "docs", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "TASK-PACKAGE",
		"--title", "Package review task",
		"--package-id", "PKG-REVIEW",
		"--worktree", worker,
		"--branch", "codex/package-review",
		"--base-commit", base,
		"--allowed", "feature.txt",
		"--allowed", "docs/**",
		"--forbidden", "secrets/**",
		"--gate", "go test ./...",
	}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "feature.txt"), []byte("feature\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worker, "docs", "reviews", "package-review.md"), []byte("Self-review: local/static evidence only; docs drift checked.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, worker, "add", "feature.txt", "docs/reviews/package-review.md")
	git(t, worker, "commit", "-q", "-m", "package review fixture")
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "review", "--task-id", "TASK-PACKAGE", "--status", "completed-unreviewed", "--note", "ready"}); err != nil {
		t.Fatal(err)
	}

	if err := cmdPackReview([]string{"--repo", project, "--ledger", ledger, "--package-id", "PKG-REVIEW", "--output", outputDir}); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"review-pack.json", "reviewer-prompt.md", "changed-files.txt", "gates.md", "evidence.md", "residual-risks.md", "diff.patch", "TASK-PACKAGE.patch"} {
		if _, err := os.Stat(filepath.Join(outputDir, path)); err != nil {
			t.Fatalf("expected review pack file %s: %v", path, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(outputDir, "review-pack.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report ReviewPack
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.PackageID != "PKG-REVIEW" || report.Status != "passed" {
		t.Fatalf("unexpected review pack: %#v", report)
	}
	if len(report.Tasks) != 1 || report.Tasks[0].ID != "TASK-PACKAGE" {
		t.Fatalf("expected task summary in review pack, got %#v", report.Tasks)
	}
	if !containsAuthorizationStatus(report.AuthorizationMatrix, "merge", "not-authorized-by-pack") {
		t.Fatalf("expected merge authorization boundary, got %#v", report.AuthorizationMatrix)
	}
	prompt, err := os.ReadFile(filepath.Join(outputDir, "reviewer-prompt.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(prompt), "independent read-only reviewer") || !strings.Contains(string(prompt), "evidence labels") {
		t.Fatalf("unexpected reviewer prompt:\n%s", string(prompt))
	}

	acceptancePath := filepath.Join(root, "package-acceptance.json")
	if err := cmdPackAcceptance([]string{"--repo", project, "--ledger", ledger, "--package-id", "PKG-REVIEW", "--write-report", acceptancePath}); err != nil {
		t.Fatal(err)
	}
	acceptanceData, err := os.ReadFile(acceptancePath)
	if err != nil {
		t.Fatal(err)
	}
	var acceptance PackageAcceptanceReport
	if err := json.Unmarshal(acceptanceData, &acceptance); err != nil {
		t.Fatal(err)
	}
	if acceptance.PackageID != "PKG-REVIEW" || acceptance.Status != "passed" || acceptance.Decision != "needs-review" {
		t.Fatalf("unexpected package acceptance report: %#v", acceptance)
	}
	if len(acceptance.Tasks) != 1 || acceptance.Tasks[0].ID != "TASK-PACKAGE" {
		t.Fatalf("expected selected package task in acceptance report, got %#v", acceptance.Tasks)
	}
	if !containsAuthorizationStatus(acceptance.AuthorizationMatrix, "merge", "requires-separate-orchestrator-decision") {
		t.Fatalf("expected package acceptance merge boundary, got %#v", acceptance.AuthorizationMatrix)
	}
	if err := cmdAppendEvent([]string{"--ledger", ledger, "--type", "cleanup", "--task-id", "TASK-PACKAGE", "--status", "cleaned", "--note", "worktree removed after accepted merge"}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "remove", "-f", worker)
	postCleanupAcceptance, err := buildPackageAcceptanceReport(project, ledger, "PKG-REVIEW", []string{"TASK-PACKAGE"})
	if err != nil {
		t.Fatal(err)
	}
	if postCleanupAcceptance.Status != "passed" || postCleanupAcceptance.Decision != "review-ready" || postCleanupAcceptance.NeedsHuman {
		t.Fatalf("expected post-cleanup package acceptance not to fail on removed worktree, got %#v", postCleanupAcceptance)
	}
	if got := strings.Join(postCleanupAcceptance.EvidenceReviewed, "\n"); !strings.Contains(got, "post-cleanup boundary") {
		t.Fatalf("expected post-cleanup evidence, got %#v", postCleanupAcceptance.EvidenceReviewed)
	}
	if got := strings.Join(postCleanupAcceptance.ResidualRisks, "\n"); !strings.Contains(got, "Post-cleanup mode cannot rerun worker worktree diff") {
		t.Fatalf("expected post-cleanup residual risk, got %#v", postCleanupAcceptance.ResidualRisks)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "OTHER-PACKAGE-TASK",
		"--package-id", "OTHER-PACKAGE",
		"--pending-worktree-id", "other-pwt",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdPackReview([]string{"--repo", project, "--ledger", ledger, "--package-id", "PKG-REVIEW", "--task-id", "OTHER-PACKAGE-TASK", "--write-report", filepath.Join(root, "bad-review-pack.json")}); err == nil {
		t.Fatal("expected wrong-package explicit task to fail")
	}
	failedReviewFile := filepath.Join(root, "pi-failed-review.md")
	if err := os.WriteFile(failedReviewFile, []byte("Verdict: reject\nFinding: missing proof.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdReviewImport([]string{"--ledger", ledger, "--package-id", "PKG-REVIEW", "--reviewer", "pi", "--file", failedReviewFile, "--task-id", "TASK-PACKAGE", "--status", "failed"}); err != nil {
		t.Fatal(err)
	}
	failedAcceptance, err := buildPackageAcceptanceReport(project, ledger, "PKG-REVIEW", []string{"TASK-PACKAGE"})
	if err != nil {
		t.Fatal(err)
	}
	if !failedAcceptance.NeedsHuman || failedAcceptance.Decision != "needs-review" || !strings.Contains(strings.Join(failedAcceptance.ResidualRisks, "\n"), "pi reported failed") {
		t.Fatalf("expected failed external review to force package review attention, got %#v", failedAcceptance)
	}
}

func TestReviewRunDryRunAndImportRecordExternalReviewer(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	packDir := filepath.Join(root, "review-pack")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(packDir, 0o755); err != nil {
		t.Fatal(err)
	}
	pack := ReviewPack{
		SchemaVersion: 1,
		PackageID:     "PKG-EXT",
		Tasks:         []MergeReadinessTaskSummary{{ID: "TASK-EXT"}},
		Evidence:      normalizedEvidence(nil),
	}
	if err := writeJSON(filepath.Join(packDir, "review-pack.json"), pack); err != nil {
		t.Fatal(err)
	}
	if err := writeText(filepath.Join(packDir, "reviewer-prompt.md"), "Review this package.\n"); err != nil {
		t.Fatal(err)
	}
	reportPath := filepath.Join(root, "external-review.json")
	if err := cmdReviewRun([]string{"--repo", project, "--ledger", ledger, "--package-id", "PKG-EXT", "--reviewer", "pi", "--pack", packDir, "--dry-run", "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var dryRun ExternalReviewReport
	if err := json.Unmarshal(data, &dryRun); err != nil {
		t.Fatal(err)
	}
	if dryRun.Status != "passed" || !strings.Contains(strings.Join(dryRun.RunnerCommand, " "), "pi") {
		t.Fatalf("expected dry-run pi command, got %#v", dryRun)
	}
	reviewFile := filepath.Join(root, "pi-review.md")
	if err := os.WriteFile(reviewFile, []byte("Verdict: concerns\nFinding: check evidence labels.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := cmdReviewImport([]string{"--ledger", ledger, "--package-id", "PKG-EXT", "--reviewer", "pi", "--file", reviewFile, "--task-id", "TASK-EXT", "--status", "failed"}); err != nil {
		t.Fatal(err)
	}
	updated, err := loadLedger(ledger)
	if err != nil {
		t.Fatal(err)
	}
	if len(updated.RoutineRuns) != 1 {
		t.Fatalf("expected one external review routine run, got %#v", updated.RoutineRuns)
	}
	run := updated.RoutineRuns[0]
	if run.RoutineID != "external-reviewer" || run.PackageID != "PKG-EXT" || run.Reviewer != "pi" || run.Status != "failed" {
		t.Fatalf("unexpected external review run: %#v", run)
	}
	if got := strings.Join(run.Evidence["proxy"], "\n"); !strings.Contains(got, "Imported external reviewer output") {
		t.Fatalf("expected proxy advisory evidence, got %#v", run.Evidence)
	}
}

func TestReviewPolicyDefaultAndConfiguredRecommendations(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	report := runReviewPolicy(project, "", "PKG-POLICY", "medium", 4)
	if report.Status != "passed" || !report.ReviewRequired || report.ReviewDecision != "one-reviewer" {
		t.Fatalf("expected default medium package to require one reviewer, got %#v", report)
	}
	if len(report.RecommendedReviewers) != 1 || report.RecommendedReviewers[0].Name != "pi" {
		t.Fatalf("expected default primary pi recommendation, got %#v", report.RecommendedReviewers)
	}
	if report.EvidenceLabel != "local/static" || !strings.Contains(report.Boundary, "does not run reviewers") {
		t.Fatalf("expected local/static policy boundary, got label=%q boundary=%q", report.EvidenceLabel, report.Boundary)
	}

	configPath := filepath.Join(project, ".codex-orchestrator", "review-policy.json")
	policy := `{
  "reviewPolicyVersion": 1,
  "defaultMode": "package-boundary",
  "primaryReviewer": "go-reviewer",
  "secondaryReviewer": "missing-reviewer",
  "manualReviewers": ["deepseek", "human"],
  "trigger": {"minTasksInPackage": 3, "maxTasksBeforeReview": 5},
  "decision": {"lowRisk": "optional", "mediumRisk": "one-reviewer", "highRisk": "two-reviewers", "externalReviewEvidence": "proxy/advisory"},
  "reviewers": {
    "go-reviewer": {"enabled": true, "command": "go", "timeoutMinutes": 3},
    "missing-reviewer": {"enabled": true, "command": "definitely-missing-codex-orchestrator-reviewer", "timeoutMinutes": 3}
  }
}`
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, []byte(policy), 0o644); err != nil {
		t.Fatal(err)
	}
	configured := runReviewPolicy(project, "", "PKG-HIGH", "high", 5)
	if configured.Status != "passed" || !configured.ReviewRequired || configured.ReviewDecision != "two-reviewers" {
		t.Fatalf("expected high risk package to require two reviewers, got %#v", configured)
	}
	if len(configured.RecommendedReviewers) != 2 {
		t.Fatalf("expected two reviewer decisions, got %#v", configured.RecommendedReviewers)
	}
	if configured.RecommendedReviewers[0].Status != "available" || configured.RecommendedReviewers[1].Status != "missing" {
		t.Fatalf("expected one available and one missing reviewer, got %#v", configured.RecommendedReviewers)
	}
	if !configured.NeedsHuman || !containsString(configured.MissingReviewers, "missing-reviewer") {
		t.Fatalf("expected missing reviewer to need human/import path, got missing=%#v needsHuman=%v", configured.MissingReviewers, configured.NeedsHuman)
	}
}

func TestPackConsultationSummarizesBlockedLedgerTask(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	reportPath := filepath.Join(root, "reports", "consultation.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/blocked", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "CONSULT-BLOCKED",
		"--title", "Blocked owner decision",
		"--thread-id", "thread_blocked",
		"--worktree", worker,
		"--branch", "codex/blocked",
		"--base-commit", base,
		"--allowed", "cmd/codex-orchestrator/**",
		"--forbidden", ".github/**",
		"--gate", "go test ./...",
		"--evidence", "local",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{
		"--ledger", ledger,
		"--type", "blocked",
		"--task-id", "CONSULT-BLOCKED",
		"--status", "blocked",
		"--note", "Blocked on product decision: choose strict cleanup wording before implementation continues.",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordRoutineRun([]string{
		"--ledger", ledger,
		"--routine", "docs-drift-checker",
		"--task-id", "CONSULT-BLOCKED",
		"--status", "blocked",
		"--evidence-local", "reviewed README and roadmap wording",
		"--evidence-blocked", "owner decision still required",
		"--action", "checked local docs",
		"--next", "ask owner to pick wording",
		"--needs-human",
		"--blocked-reason", "Product decision required before cleanup wording can be finalized.",
	}); err != nil {
		t.Fatal(err)
	}

	if err := cmdPackConsultation([]string{"--ledger", ledger, "--task-id", "CONSULT-BLOCKED", "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report ConsultationRequestPack
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("expected consultation JSON, got %q: %v", string(data), err)
	}
	if report.Command != "pack consultation" || report.Status != "blocked" || !report.NeedsHuman {
		t.Fatalf("expected blocked consultation pack needing human, got %#v", report)
	}
	if report.Task.ID != "CONSULT-BLOCKED" || report.Task.Title != "Blocked owner decision" || report.Task.ThreadID != "thread_blocked" {
		t.Fatalf("expected task metadata, got %#v", report.Task)
	}
	if !strings.Contains(report.BlockedReason, "Product decision required") {
		t.Fatalf("expected routine blocked reason, got %q", report.BlockedReason)
	}
	if len(report.AttemptedPaths) < 3 {
		t.Fatalf("expected history and routine attempts, got %#v", report.AttemptedPaths)
	}
	if got := strings.Join(report.RecordedGates, "\n"); !strings.Contains(got, "go test ./...") {
		t.Fatalf("expected recorded gate, got %#v", report.RecordedGates)
	}
	if got := strings.Join(report.EvidenceLabels, "\n"); !strings.Contains(got, "local") || !strings.Contains(got, "blocked") {
		t.Fatalf("expected local/blocked evidence labels, got %#v", report.EvidenceLabels)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 {
		t.Fatalf("expected no direct/proxy evidence, got %#v", report.Evidence)
	}
	if got := report.RequiredHumanInput[0].Kind; got != "product-decision" && got != "routine-review" {
		t.Fatalf("expected product or routine human request, got %#v", report.RequiredHumanInput)
	}
	if len(report.DecisionOptions) < 3 {
		t.Fatalf("expected decision options, got %#v", report.DecisionOptions)
	}
	if report.OwnerDecisionBrief.Title != "Blocked owner decision" || !strings.Contains(report.OwnerDecisionBrief.WhyNeededNow, "Product decision required") {
		t.Fatalf("expected owner decision brief, got %#v", report.OwnerDecisionBrief)
	}
	if len(report.AuthorizationMatrix) == 0 || !containsAuthorizationStatus(report.AuthorizationMatrix, "merge", "not-authorized-by-consultation") {
		t.Fatalf("expected consultation authorization matrix, got %#v", report.AuthorizationMatrix)
	}
	if report.BranchWorktreeDisposition.Recommendation != "keep" {
		t.Fatalf("expected keep disposition for blocked task, got %#v", report.BranchWorktreeDisposition)
	}
	if !strings.Contains(report.NextSafeAction, "CONSULT-BLOCKED") || !strings.Contains(report.Boundary, "does not dispatch, merge, push, cleanup, edit ledger, edit git state") {
		t.Fatalf("expected conservative next action and boundary, next=%q boundary=%q", report.NextSafeAction, report.Boundary)
	}
}

func TestPackConsultationDetectsHumanDeviceAction(t *testing.T) {
	root := t.TempDir()
	project := createRepo(t, filepath.Join(root, "repo"))
	base := gitOutputForTest(t, project, "rev-parse", "HEAD")
	worker := filepath.Join(root, "worker")
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	git(t, project, "worktree", "add", "-q", "-b", "codex/device", worker, "HEAD")
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "CONSULT-DEVICE",
		"--title", "PAX device action required",
		"--worktree", worker,
		"--branch", "codex/device",
		"--base-commit", base,
		"--gate", "go test ./...",
	}); err != nil {
		t.Fatal(err)
	}
	if err := cmdAppendEvent([]string{
		"--ledger", ledger,
		"--type", "blocked",
		"--task-id", "CONSULT-DEVICE",
		"--status", "blocked",
		"--note", "Needs human action on physical PAX device before the worker can continue.",
	}); err != nil {
		t.Fatal(err)
	}

	output := captureStdout(t, func() error {
		return cmdPackConsultation([]string{"--ledger", ledger, "--task-id", "CONSULT-DEVICE", "--json"})
	})
	var report ConsultationRequestPack
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("expected consultation JSON, got %q: %v", output, err)
	}
	foundDeviceInput := false
	for _, input := range report.RequiredHumanInput {
		if input.Kind == "human-physical-action" && strings.Contains(input.Request, "human/device action") {
			foundDeviceInput = true
		}
	}
	if !foundDeviceInput {
		t.Fatalf("expected human/device request, got %#v", report.RequiredHumanInput)
	}
	if report.LiveProofGate.Status != "blocked-outside-pack" || !report.LiveProofGate.Required || !report.LiveProofGate.WaiverRequired {
		t.Fatalf("expected blocked live proof gate for device action, got %#v", report.LiveProofGate)
	}
	if got := strings.Join(report.OwnerDecisionBrief.MissingEvidence, "\n"); !strings.Contains(got, "human-physical-action") || !strings.Contains(got, "Direct live/runtime/device/provider proof") {
		t.Fatalf("expected owner brief missing device evidence, got %#v", report.OwnerDecisionBrief)
	}
	if !strings.Contains(report.Blocker, "PAX device") {
		t.Fatalf("expected PAX blocker, got %q", report.Blocker)
	}
	if report.BranchWorktreeDisposition.Recommendation != "keep" || report.BranchWorktreeDisposition.Branch != "codex/device" {
		t.Fatalf("expected keep branch/worktree disposition, got %#v", report.BranchWorktreeDisposition)
	}
	if report.Status != "blocked" || report.Task.Status != "blocked" {
		t.Fatalf("expected ledger blocker to make the consultation pack blocked, got status=%q task=%#v observed=%q", report.Status, report.Task, report.ObservedStatus)
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
		"README.zh-CN.md mentions all runnable routines",
		"SKILL.md mentions all runnable routines",
		"docs/routines/README.md mentions all runnable routines",
		"docs/v2-usage.md mentions all runnable routines",
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

func TestRunDocsDriftCheckerRoutineFailsOnPostMergeDocsDriftGuardWarning(t *testing.T) {
	root := t.TempDir()
	project := createDocsDriftFixture(t, root, []string{"pr-reviewer", "docs-drift-checker"})
	review := filepath.Join(project, "docs", "reviews", "accepted-routine-runner.md")
	if err := os.MkdirAll(filepath.Dir(review), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(review, []byte(`# Review

Status: accepted merge.
Changed files:
- cmd/codex-orchestrator/main.go
- routines/new-routine.json

Self-review: local evidence only.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runDocsDriftCheckerRoutine(project)
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "docs/reviews/accepted-routine-runner.md may describe an accepted/merged central-impact task") {
		t.Fatalf("expected post-merge docs drift warning, got:\n%s", local)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected local-only failed evidence, got %#v", report.Evidence)
	}
}

func TestRunDocsDriftCheckerRoutineAllowsPostMergeDocsDriftDecision(t *testing.T) {
	root := t.TempDir()
	project := createDocsDriftFixture(t, root, []string{"pr-reviewer", "docs-drift-checker"})
	review := filepath.Join(project, "docs", "reviews", "accepted-routine-runner.md")
	if err := os.MkdirAll(filepath.Dir(review), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(review, []byte(`# Review

Status: accepted merge.
Changed files:
- cmd/codex-orchestrator/main.go
- routines/new-routine.json

Docs drift: README.md and docs/routines/README.md were updated, and the proof remains local/static.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runDocsDriftCheckerRoutine(project)
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "docs/reviews post-merge docs drift guard found no accepted/merged central-impact task notes missing a docs update decision") {
		t.Fatalf("expected post-merge guard pass evidence, got:\n%s", local)
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

func TestRunRoadmapNextTaskSuggesterRoutineWritesPassedReport(t *testing.T) {
	root := t.TempDir()
	project, ledger := createRoadmapNextTaskFixture(t, root, true)
	reportPath := filepath.Join(root, "reports", "roadmap-next-task-suggester.json")

	if err := cmdRunRoutine([]string{"roadmap-next-task-suggester", "--repo", project, "--ledger", ledger, "--write-report", reportPath}); err != nil {
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
	if report.RoutineID != "roadmap-next-task-suggester" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Roadmap candidate tasks: stale task rescuer, PR reviewer, CI fixer, docs drift checker, rebase helper, release verifier, evidence label auditor deeper policy/eval variants, roadmap next-task suggester, per-routine runtime budget / review budget 与 heartbeat 更深集成",
		"Runnable routines from cmd/codex-orchestrator/main.go: ci-fixer, docs-drift-checker, evidence-label-auditor, pr-reviewer, release-verifier, roadmap-next-task-suggester, stale-task-rescuer",
		"rebase helper: already represented by ledger task REBASE-HELPER",
		"local suggestion: evidence label auditor deeper policy/eval variants.",
		"local suggestion: per-routine runtime budget / review budget 与 heartbeat 更深集成.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
	if !strings.Contains(report.NextSuggestedAction, "evidence label auditor deeper policy/eval variants") {
		t.Fatalf("expected primary next action to prefer remaining v3 read-only work, got %q", report.NextSuggestedAction)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected local-only passed evidence, got %#v", report.Evidence)
	}
}

func TestRunRoadmapNextTaskSuggesterRoutineQueueDrainedWhenOnlyUnsafeItemsRemain(t *testing.T) {
	root := t.TempDir()
	project, _ := createRoadmapNextTaskFixture(t, root, false)
	roadmap := `# roadmap

## v3：Routine library

候选 routine：

- rebase helper；
- release verifier。
`
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(roadmap), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapNextTaskSuggesterRoutine(project, "")
	if report.Status != "passed" {
		t.Fatalf("expected passed queue-drained report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "No remaining safe read-only roadmap tasks were found") {
		t.Fatalf("expected queue-drained local evidence, got:\n%s", local)
	}
	if !strings.Contains(report.NextSuggestedAction, "queue appears drained") {
		t.Fatalf("expected queue-drained next action, got %q", report.NextSuggestedAction)
	}
}

func TestRunRoadmapNextTaskSuggesterRoutineParsesNextStagePriorities(t *testing.T) {
	root := t.TempDir()
	project, _ := createRoadmapNextTaskFixture(t, root, false)
	roadmap := `# roadmap

## 下一阶段优先级

1. Runtime status report。
   - 输出 active workers、pending setup 和 completed-unreviewed。

2. First-class setup/worktree state model。
   - 把 pendingWorktreeId、真实 worktree、branch 和 clean commit 作为工具级状态。

3. Automated review checklist。
   - 检查 allowed/forbidden paths、review doc、artifact 和 evidence labels。

4. Evidence-label linter。
   - 防止 local/proxy/weak 被写成 direct/pre/prod/device proof。

5. Post-merge docs drift guard。
   - accepted merge 后提示 central docs 是否需要统领更新。

6. Case study and bootstrap docs。
   - 把真实项目经验写成脱敏案例。

暂不进入的方向：

- 重 daemon；
- 自动 session scheduler；
- Homebrew package manager route。
`
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(roadmap), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapNextTaskSuggesterRoutine(project, "")
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Roadmap candidate tasks: Runtime status report, First-class setup/worktree state model, Automated review checklist, Evidence-label linter, Post-merge docs drift guard, Case study and bootstrap docs",
		"local suggestion: Runtime status report.",
		"local suggestion: First-class setup/worktree state model.",
		"local suggestion: Automated review checklist.",
		"local suggestion: Evidence-label linter.",
		"local suggestion: Post-merge docs drift guard.",
		"local suggestion: Case study and bootstrap docs.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
	if strings.Contains(local, "daemon") || strings.Contains(local, "Homebrew") {
		t.Fatalf("expected excluded future directions not to be suggested, got:\n%s", local)
	}
	if !strings.Contains(report.NextSuggestedAction, "Runtime status report") {
		t.Fatalf("expected primary next action to prefer first next-stage priority, got %q", report.NextSuggestedAction)
	}
}

func TestRunRoadmapNextTaskSuggesterRoutineSkipsCompletedCandidateWording(t *testing.T) {
	root := t.TempDir()
	project, _ := createRoadmapNextTaskFixture(t, root, false)
	roadmap := `# roadmap

## v2.5：Verification routine foundation

剩余：

- budget policy report runner：如果继续推进，下一步只能实现只读 run-routine budget-policy-report。

## v3：Routine library

候选 routine：

- orchestration policy auditor follow-on eval fixtures：已补 transcript-style local review-note fixtures 和 human-review transcript fixtures；
- policy auditor done coverage: already completed.
`
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(roadmap), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapNextTaskSuggesterRoutine(project, "")
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "local suggestion: budget policy report runner") {
		t.Fatalf("expected budget policy report runner to remain suggestable, got:\n%s", local)
	}
	for _, blockedSuggestion := range []string{
		"local suggestion: orchestration policy auditor follow-on eval fixtures",
		"local suggestion: policy auditor done coverage",
	} {
		if strings.Contains(local, blockedSuggestion) {
			t.Fatalf("expected completed roadmap candidate not to be suggested (%q), got:\n%s", blockedSuggestion, local)
		}
	}
	if !strings.Contains(local, "skipped because the roadmap candidate text already marks it completed/done/covered") {
		t.Fatalf("expected completed candidate skip evidence, got:\n%s", local)
	}
	if !strings.Contains(report.NextSuggestedAction, "budget policy report runner") {
		t.Fatalf("expected primary next action to preserve budget policy report runner, got %q", report.NextSuggestedAction)
	}
}

func TestRunRoadmapNextTaskSuggesterRoutineBlockedWhenRoadmapMissing(t *testing.T) {
	root := t.TempDir()
	report := runRoadmapNextTaskSuggesterRoutine(root, "")
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "Could not read docs/roadmap.md") {
		t.Fatalf("expected roadmap blocked evidence, got %#v", report.Evidence)
	}
}

func TestRoadmapScoreUsesProjectAwareDefaultSources(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "docs", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(`# Roadmap

## Remaining

- Runtime proof for PAX device smoke before claiming direct proof.
- Docs only readiness page cleanup for old queue.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "PROGRESS.md"), []byte(`# Progress

- Blocked-removal: unblock webhook projection drift in backend before next dispatch.
- Owner-gated: ask owner to confirm production deploy window.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "reviews", "accepted.md"), []byte(`# Review

## Next Actions

- Vertical completion: finish Terminal payment split across backend and web surfaces.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "", "")
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 {
		t.Fatalf("expected local/static only evidence, got %#v", report.Evidence)
	}
	if got := report.Summary.ByClass["blocked-removal"]; got != 1 {
		t.Fatalf("expected one blocked-removal candidate, got %d in %#v", got, report.Summary.ByClass)
	}
	if got := report.Summary.ByClass["runtime-proof"]; got != 1 {
		t.Fatalf("expected one runtime-proof candidate, got %d in %#v", got, report.Summary.ByClass)
	}
	if got := report.Summary.ByClass["owner-gated"]; got != 1 {
		t.Fatalf("expected one owner-gated candidate, got %d in %#v", got, report.Summary.ByClass)
	}
	if got := report.Summary.ByClass["shallow-risk"]; got != 1 {
		t.Fatalf("expected one shallow-risk candidate, got %d in %#v", got, report.Summary.ByClass)
	}
	if report.Candidates[0].Classification != "blocked-removal" {
		t.Fatalf("expected blocked-removal to rank first, got %#v", report.Candidates[0])
	}
	if !containsString(report.Candidates[0].WriteSetHints, "backend/service surfaces") {
		t.Fatalf("expected backend write-set hint, got %#v", report.Candidates[0].WriteSetHints)
	}
	blocked := strings.Join(report.Evidence["blocked"], "\n")
	if !strings.Contains(blocked, "Real project judgement") {
		t.Fatalf("expected blocked human-judgement boundary, got %q", blocked)
	}
}

func TestRoadmapScoreReadsConfigSources(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "PLAN.md"), []byte(`# Plan

- Runtime proof: browser smoke for admin web before direct evidence.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"sources":["PLAN.md"]}`
	if err := os.WriteFile(filepath.Join(project, "roadmap-score.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "roadmap-score.json", "")
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if len(report.Sources) != 1 || report.Sources[0].Path != "PLAN.md" || report.Sources[0].Candidates != 1 {
		t.Fatalf("expected config source only, got %#v", report.Sources)
	}
	if report.Candidates[0].Classification != "runtime-proof" {
		t.Fatalf("expected runtime-proof candidate, got %#v", report.Candidates[0])
	}
	if !containsString(report.Candidates[0].WriteSetHints, "product app surfaces") {
		t.Fatalf("expected product app write-set hint, got %#v", report.Candidates[0].WriteSetHints)
	}
}

func TestRoadmapScoreBlockedWhenNoCandidates(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(`# Roadmap

Everything is completed.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "", "")
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "No actionable local/static candidate lines") {
		t.Fatalf("expected no-candidates evidence, got %q", got)
	}
}

func TestRoadmapScoreHandlesColonSectionLabels(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(`# Roadmap

暂不进入的方向：

- 声称 helper 能证明 live Codex App runtime、production、payment、hardware 或设备行为。

下一阶段优先级：

- Package ledger status：待做，围绕一个功能包显示 active/review/blocked/cleaned 状态。
`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "", "")
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if len(report.Candidates) != 1 {
		t.Fatalf("expected only actionable priority candidate, got %#v", report.Candidates)
	}
	if !strings.Contains(report.Candidates[0].Title, "Package ledger status") {
		t.Fatalf("expected package ledger candidate, got %#v", report.Candidates[0])
	}
}

func TestRoadmapScorePrioritizesFeaturePackageLane(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(`# Roadmap

## 下一阶段功能包推进顺序

- Package ledger / package status：待做，围绕一个功能包显示 active/review/blocked/cleaned 状态。
- Runtime proof for admin browser smoke before direct evidence.
`), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "", "")
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if !strings.Contains(report.Candidates[0].Title, "Package ledger") {
		t.Fatalf("expected package lane to rank first, got %#v", report.Candidates)
	}
	if report.Candidates[0].Score <= report.Candidates[1].Score {
		t.Fatalf("expected package lane score to outrank runtime proof, got %#v", report.Candidates)
	}
}

func TestRoadmapScoreIgnoresReviewResidualRiskFollowup(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "docs", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(`# Roadmap

## 下一阶段优先级

3. Consultation Request Pack：待做。
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "reviews", "old-budget.md"), []byte(`# Budget Review

## Residual Risks

- Budget-policy static eval remains a follow-up for detecting future wording.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"sources":["docs/roadmap.md","docs/reviews/old-budget.md"]}`
	if err := os.WriteFile(filepath.Join(project, "roadmap-score.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(project, defaultLedger)
	ledger := Ledger{
		Version:     1,
		ProjectRoot: project,
		Tasks: []Task{{
			ID:     "BUDGET-POLICY-STATIC-EVAL",
			Title:  "Budget-policy static eval",
			Status: "cleaned",
			History: []map[string]string{{
				"status": "cleaned",
				"note":   "Budget-policy static eval remains a follow-up for detecting future wording",
			}},
		}},
	}
	if err := writeJSON(ledgerPath, ledger); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "roadmap-score.json", ledgerPath)
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if !strings.Contains(report.NextSuggestedAction, "Consultation Request Pack") {
		t.Fatalf("expected current roadmap pending task to rank first, got %q with candidates %#v", report.NextSuggestedAction, report.Candidates)
	}
	for _, candidate := range report.Candidates {
		if strings.Contains(candidate.Title, "Budget-policy static eval") {
			t.Fatalf("expected residual-risk review prose to be ignored, got candidate %#v", candidate)
		}
	}
	if !strings.Contains(strings.Join(report.Evidence["local"], "\n"), "completed/merged/cleaned matches are demoted") {
		t.Fatalf("expected ledger demotion evidence, got %#v", report.Evidence)
	}
}

func TestRoadmapScoreDemotesExplicitReviewNextAction(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "docs", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(`# Roadmap

## 下一阶段优先级

- Runtime status page package：待做。
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "reviews", "old-budget.md"), []byte(`# Budget Review

## Next Actions

- Budget-policy static eval remains a follow-up for detecting future wording.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"sources":["docs/roadmap.md","docs/reviews/old-budget.md"]}`
	if err := os.WriteFile(filepath.Join(project, "roadmap-score.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(project, defaultLedger)
	ledger := Ledger{
		Version:     1,
		ProjectRoot: project,
		Tasks: []Task{{
			ID:     "BUDGET-POLICY-STATIC-EVAL",
			Title:  "Budget-policy static eval",
			Status: "cleaned",
		}},
	}
	if err := writeJSON(ledgerPath, ledger); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "roadmap-score.json", ledgerPath)
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if !strings.Contains(report.NextSuggestedAction, "Runtime status page package") {
		t.Fatalf("expected current roadmap package to rank first, got %q with candidates %#v", report.NextSuggestedAction, report.Candidates)
	}
	var demoted RoadmapScoreCandidate
	for _, candidate := range report.Candidates {
		if strings.Contains(candidate.Title, "Budget-policy static eval") {
			demoted = candidate
			break
		}
	}
	if demoted.Title == "" || demoted.LedgerMatch == "" || demoted.Score >= report.Candidates[0].Score {
		t.Fatalf("expected explicit but completed review next action to be demoted, top=%#v demoted=%#v", report.Candidates[0], demoted)
	}
}

func TestRoadmapScoreDemotesResolvedBlockerByLedgerOverlap(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	if err := os.MkdirAll(filepath.Join(project, "docs", "reviews"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(`# Roadmap

## 下一阶段优先级

- KDS route snapshot package：待做。
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", "reviews", "old-cloud-sync.md"), []byte(`# Old Cloud Sync Review

## Next Actions

- Cloud sync outbox projection deployed readback blocker remains.
`), 0o644); err != nil {
		t.Fatal(err)
	}
	config := `{"sources":["docs/roadmap.md","docs/reviews/old-cloud-sync.md"]}`
	if err := os.WriteFile(filepath.Join(project, "roadmap-score.json"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	ledgerPath := filepath.Join(project, defaultLedger)
	ledger := Ledger{
		Version:     1,
		ProjectRoot: project,
		Tasks: []Task{{
			ID:     "TF-PRE-PROJECTION-CLOSEOUT",
			Title:  "Cloud projection readback closeout",
			Status: "cleaned",
			History: []map[string]string{{
				"type":   "review",
				"status": "cleaned",
				"note":   "Outbox deployed correlation was resolved with local/static evidence.",
			}},
		}},
	}
	if err := writeJSON(ledgerPath, ledger); err != nil {
		t.Fatal(err)
	}

	report := runRoadmapScore(project, "roadmap-score.json", ledgerPath)
	if report.Status != "passed" {
		t.Fatalf("expected passed report, got %#v", report)
	}
	if !strings.Contains(report.NextSuggestedAction, "KDS route snapshot package") {
		t.Fatalf("expected current roadmap package to rank first, got %q with candidates %#v", report.NextSuggestedAction, report.Candidates)
	}
	var stale RoadmapScoreCandidate
	for _, candidate := range report.Candidates {
		if strings.Contains(candidate.Title, "Cloud sync outbox projection") {
			stale = candidate
			break
		}
	}
	if stale.Title == "" || stale.LedgerMatch == "" || stale.Score >= report.Candidates[0].Score {
		t.Fatalf("expected stale blocker to be demoted by terminal ledger overlap, top=%#v stale=%#v", report.Candidates[0], stale)
	}
	if got := strings.Join(stale.RiskHints, "\n"); !strings.Contains(got, "possibly resolved stale blocker") {
		t.Fatalf("expected resolved stale blocker risk hint, got %#v", stale.RiskHints)
	}
}

func TestRunBudgetPolicyReportRoutineWritesPassedReport(t *testing.T) {
	root := t.TempDir()
	project, ledger, heartbeat := createBudgetPolicyFixture(t, root)
	reportPath := filepath.Join(root, "reports", "budget-policy-report.json")

	if err := cmdRunRoutine([]string{"budget-policy-report", "--repo", project, "--ledger", ledger, "--heartbeat-report", heartbeat, "--write-report", reportPath}); err != nil {
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
	if report.RoutineID != "budget-policy-report" || report.Status != "passed" || !report.NeedsHuman {
		t.Fatalf("unexpected report: %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"docs/roadmap.md preserves budget-policy-report review-only boundary wording.",
		"docs/routines/README.md preserves budget-policy-report review-only boundary wording.",
		"Routine budget metadata coverage: total=2 withMaxRuntime=1 withReviewBudget=1 withBoth=1 withAnyBudget=1 withoutAnyBudget=1.",
		"Ledger budget metadata summary",
		"Heartbeat budgetPressure evidenceLabel: local/static",
		"Heartbeat budgetPressure warnings copied as local/static evidence: Task TASK-1 runtime budget near limit: 8m elapsed of 10m.",
		"No scheduler, priority engine, automatic killing, dispatch enforcement, merge, push, delete, cleanup, or worker-control action was performed.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
	blocked := strings.Join(report.Evidence["blocked"], "\n")
	for _, want := range []string{
		"Task TASK-1 is review-ready but has no recorded review-ready timestamp; human review elapsed time is unknown.",
		"Live Codex App session runtime, worker wall-clock state, and human review elapsed time were not available from direct runtime APIs; unknown live timing remains blocked/unknown.",
	} {
		if !strings.Contains(blocked, want) {
			t.Fatalf("expected blocked evidence %q in:\n%s", want, blocked)
		}
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 {
		t.Fatalf("expected no direct/proxy evidence, got %#v", report.Evidence)
	}
	for _, action := range report.ActionsTaken {
		lower := strings.ToLower(action)
		for _, forbidden := range []string{"dispatched", "prioritized", "paused", "killed", "merged", "pushed", "deleted", "cleaned"} {
			if strings.Contains(lower, forbidden) {
				t.Fatalf("expected no forbidden control action %q in actionsTaken: %#v", forbidden, report.ActionsTaken)
			}
		}
	}
}

func TestRunBudgetPolicyReportRoutineRunsWithoutOptionalInputs(t *testing.T) {
	root := t.TempDir()
	project, _, _ := createBudgetPolicyFixture(t, root)
	if err := os.RemoveAll(filepath.Join(project, defaultStateDir)); err != nil {
		t.Fatal(err)
	}

	report := runBudgetPolicyReportRoutine(project, "", "")
	if report.Status != "passed" {
		t.Fatalf("expected passed report without optional inputs, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Repo-local ledger is absent; task budget metadata inspection skipped.",
		"Repo-local heartbeat report is absent; heartbeat budgetPressure copy skipped.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected optional-input evidence %q in:\n%s", want, local)
		}
	}
}

func TestRunBudgetPolicyReportRoutineBlockedWhenDocsMissing(t *testing.T) {
	root := t.TempDir()
	report := runBudgetPolicyReportRoutine(root, "", "")
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "Could not read docs/roadmap.md") {
		t.Fatalf("expected roadmap blocked evidence, got %#v", report.Evidence)
	}
}

func TestRunEvidenceLabelAuditorRoutineWritesPassedReport(t *testing.T) {
	root := t.TempDir()
	project := createEvidenceAuditFixture(t, root)
	reportPath := filepath.Join(root, "reports", "evidence-label-auditor.json")

	if err := cmdRunRoutine([]string{"evidence-label-auditor", "--repo", project, "--write-report", reportPath}); err != nil {
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
	if report.RoutineID != "evidence-label-auditor" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Routine specs with direct evidence explicitly reserved: docs-drift-checker",
		"Scanned ",
		"repo-local evidence-label input file(s)",
		"Rule hits: none.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected only local evidence, got %#v", report.Evidence)
	}
}

func TestRunEvidenceLabelAuditorRoutineFailsOnPhraseMisuse(t *testing.T) {
	root := t.TempDir()
	project := createEvidenceAuditFixture(t, root)
	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte("Local unit tests provide direct runtime proof for production.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runEvidenceLabelAuditorRoutine(project)
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "[ELA004] README.md:1: local/static suspicion") {
		t.Fatalf("expected README suspicion, got:\n%s", local)
	}
	if !strings.Contains(local, "Rule hits: ELA004=1.") {
		t.Fatalf("expected ELA004 summary, got:\n%s", local)
	}
	if len(report.Evidence["direct"]) != 0 || len(report.Evidence["proxy"]) != 0 || len(report.Evidence["blocked"]) != 0 {
		t.Fatalf("expected local-only failed evidence, got %#v", report.Evidence)
	}
}

func TestRunEvidenceLabelAuditorRoutineScansReviewDocsForEvidencePromotion(t *testing.T) {
	root := t.TempDir()
	project := createEvidenceAuditFixture(t, root)
	reviewDir := filepath.Join(project, "docs", "reviews")
	if err := os.MkdirAll(reviewDir, 0o755); err != nil {
		t.Fatal(err)
	}
	review := strings.Join([]string{
		"# Worker Handoff",
		"",
		"Evidence summary: local static screenshots and mocked payment tests count as pre/prod/device/payment proof.",
		"Safe line: local static checks are blocked for payment proof unless explicit direct evidence is attached.",
	}, "\n")
	if err := os.WriteFile(filepath.Join(reviewDir, "handoff.md"), []byte(review+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runEvidenceLabelAuditorRoutine(project)
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "[ELA010] docs/reviews/handoff.md:3: local/static suspicion") {
		t.Fatalf("expected review-doc ELA010 finding, got:\n%s", local)
	}
	if strings.Contains(local, "docs/reviews/handoff.md:4") {
		t.Fatalf("did not expect explicit-direct-evidence safe line finding, got:\n%s", local)
	}
}

func TestRunEvidenceLabelAuditorRoutineFailsOnReportBucketsAndStaticDirect(t *testing.T) {
	root := t.TempDir()
	project := createEvidenceAuditFixture(t, root)
	report := RoutineRunReport{
		RoutineID: "docs-drift-checker",
		Status:    "passed",
		Evidence: map[string][]string{
			"direct": []string{"rendered docs in browser"},
		},
		ActionsTaken:        []string{"claimed direct docs proof"},
		NeedsHuman:          false,
		NextSuggestedAction: "none",
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "examples", "routine-reports", "bad.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	audit := runEvidenceLabelAuditorRoutine(project)
	if audit.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", audit)
	}
	local := strings.Join(audit.Evidence["local"], "\n")
	for _, want := range []string{
		`[ELA008] examples/routine-reports/bad.json: local/static suspicion: RoutineRunReport for docs-drift-checker is missing evidence bucket "proxy"`,
		"[ELA009] examples/routine-reports/bad.json: local/static suspicion: RoutineRunReport for static-only routine docs-drift-checker contains direct evidence",
		"Rule hits: ELA008=3, ELA009=1.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected finding %q in:\n%s", want, local)
		}
	}
}

func TestAuditEvidenceTextAllowsGlossaryProhibitionAndBlockedDefinitions(t *testing.T) {
	text := strings.Join([]string{
		"Evidence labels are direct evidence, proxy evidence, local evidence, and blocked evidence buckets.",
		"Do not claim direct runtime proof from local checks.",
		"Blocked evidence label: the claim could not be proven safely.",
	}, "\n")
	findings := auditEvidenceText("README.md", text)
	if len(findings) != 0 {
		t.Fatalf("expected no glossary/prohibition findings, got %#v", renderEvidenceAuditFindings(findings))
	}
}

func TestAuditEvidenceTextStillFlagsBlockedOverclaim(t *testing.T) {
	findings := auditEvidenceText("README.md", "Blocked evidence provides direct runtime proof.\n")
	got := strings.Join(renderEvidenceAuditFindings(findings), "\n")
	if !strings.Contains(got, "[ELA004] README.md:1: local/static suspicion") {
		t.Fatalf("expected blocked overclaim finding, got:\n%s", got)
	}
}

func TestAuditRoutineSpecEvidenceDescriptionsAddsRuleIDs(t *testing.T) {
	spec := RoutineSpec{
		ID: "fixture-proof",
		OutputSchema: RoutineOutputSpec{
			Evidence: map[string]string{
				"direct":  "Local fixture proof.",
				"local":   "Local evidence provides direct runtime proof.",
				"blocked": "Blocked evidence is verified.",
			},
		},
	}
	findings := auditRoutineSpecEvidenceDescriptions("routines/fixture-proof.json", spec)
	got := strings.Join(renderEvidenceAuditFindings(findings), "\n")
	for _, want := range []string{
		"[ELA001] routines/fixture-proof.json: local/static suspicion: direct evidence description for fixture-proof contains local/static wording",
		"[ELA002] routines/fixture-proof.json: local/static suspicion: local evidence description for fixture-proof contains strong proof wording",
		"[ELA003] routines/fixture-proof.json: local/static suspicion: blocked evidence description for fixture-proof may describe proof as blocked",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected finding %q in:\n%s", want, got)
		}
	}
}

func TestRunEvidenceLabelAuditorRoutineBlockedWhenSpecsMissing(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("safe docs\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	report := runEvidenceLabelAuditorRoutine(root)
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "Could not inspect routines directory") {
		t.Fatalf("expected routines directory blocked evidence, got %#v", report.Evidence)
	}
}

func TestRunOrchestrationPolicyAuditorRoutinePassesSafeFixture(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	reportPath := filepath.Join(root, "reports", "orchestration-policy-auditor.json")

	if err := cmdRunRoutine([]string{"orchestration-policy-auditor", "--repo", project, "--write-report", reportPath}); err != nil {
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
	if report.RoutineID != "orchestration-policy-auditor" || report.Status != "passed" {
		t.Fatalf("unexpected report: %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Scanned ",
		"repo-local orchestration policy input file(s)",
		"Rule hits: none.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
}

func TestRunOrchestrationPolicyAuditorRoutineFlagsPolicyMisuse(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	bad := strings.Join([]string{
		"Dry run mode can dispatch workers immediately.",
		"",
		"If worktree setup failed, fallback to the main checkout and implement the task there.",
		"",
		"When a child task is completed, delete heartbeat and stop the loop.",
		"",
		"Worker prompt: use a worktree branch for this task.",
		"",
		"Local proxy smoke counts as direct proof.",
		"",
		"The budget-policy-report helper automatically paused workers after heartbeat budget warnings and enforced dispatch eligibility.",
	}, "\n")
	if err := os.WriteFile(filepath.Join(project, "README.md"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	report := runOrchestrationPolicyAuditorRoutine(project)
	if report.Status != "failed" {
		t.Fatalf("expected failed report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"[OPA001] README.md:1: dry-run wording appears to allow dispatch/session creation without explicit confirmation",
		"[OPA002] README.md:3: fallback wording appears to allow implementation in the orchestrator/main checkout after setup failure",
		"[OPA003] README.md:5: heartbeat/child-task completion wording may stop the larger queue without a ledger/roadmap/repo-truth check",
		"[OPA004] README.md:7: worker/delegation prompt lacks one of the core boundaries",
		"[OPA005] README.md:9: evidence wording appears to allow local/proxy/weak evidence to be promoted to direct proof",
		"[OPA008] README.md:11: budget-policy wording appears to promote local/static budget evidence or imply helper budget enforcement/scheduling behavior",
		"Rule hits: OPA001=1, OPA002=1, OPA003=1, OPA004=1, OPA005=1, OPA008=1.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected finding %q in:\n%s", want, local)
		}
	}
}

func TestRunOrchestrationPolicyAuditorRoutineBlockedWhenNoInputs(t *testing.T) {
	root := t.TempDir()
	report := runOrchestrationPolicyAuditorRoutine(root)
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if got := strings.Join(report.Evidence["blocked"], "\n"); !strings.Contains(got, "Could not collect orchestration policy audit paths") {
		t.Fatalf("expected input collection blocked evidence, got %#v", report.Evidence)
	}
}

func TestRulesProposeFromTextProducesReviewOnlyProposal(t *testing.T) {
	text := strings.Join([]string{
		"Dry run mode can dispatch workers immediately.",
		"",
		"Local proxy smoke counts as direct proof.",
	}, "\n")

	report := runRulesPropose("", text, "")
	if report.Status != "passed" {
		t.Fatalf("expected passed rules proposal report, got %#v", report)
	}
	if !report.NeedsHumanReview || report.EvidenceLabel != "local" {
		t.Fatalf("expected local human-review report, got %#v", report)
	}
	if len(report.Proposals) != 2 {
		t.Fatalf("expected two rule proposals, got %#v", report.Proposals)
	}
	got := report.Proposals[0].Title + "\n" + report.Proposals[1].Title
	for _, want := range []string{
		"Require explicit approval before dispatch after dry run",
		"Prevent local or proxy evidence from becoming direct proof",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected proposal %q in:\n%s", want, got)
		}
	}
	for _, proposal := range report.Proposals {
		if !proposal.NeedsHumanReview || proposal.EvidenceLabel != "local" || proposal.Source != "inline text" {
			t.Fatalf("expected review-only local proposal, got %#v", proposal)
		}
	}
	if gotEvidence := strings.Join(report.Evidence, "\n"); !strings.Contains(gotEvidence, "no rule, skill, README, policy, AGENTS, or CLAUDE file was edited") {
		t.Fatalf("expected no-mutation evidence, got:\n%s", gotEvidence)
	}
}

func TestRulesProposeBlockedWithoutInput(t *testing.T) {
	report := runRulesPropose("", "", "")
	if report.Status != "blocked" || report.BlockedReason == "" {
		t.Fatalf("expected blocked report, got %#v", report)
	}
	if len(report.Proposals) != 0 {
		t.Fatalf("expected no proposals without input, got %#v", report.Proposals)
	}
	if !strings.Contains(report.BlockedReason, "requires one of") {
		t.Fatalf("expected clear blocked reason, got %q", report.BlockedReason)
	}
}

func TestCmdRulesProposeWritesReportOnly(t *testing.T) {
	root := t.TempDir()
	reviewPath := filepath.Join(root, "review.md")
	reportPath := filepath.Join(root, "rules-proposal.json")
	if err := os.WriteFile(reviewPath, []byte("If worktree setup failed, fallback to the main checkout and implement the task there.\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := cmdRules([]string{"propose", "--from-review", reviewPath, "--write-report", reportPath}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report RuleProposalReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Status != "passed" || len(report.Proposals) != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
	proposal := report.Proposals[0]
	if proposal.Title != "Block main-checkout fallback after worker setup failure" {
		t.Fatalf("unexpected proposal: %#v", proposal)
	}
	if !proposal.NeedsHumanReview || proposal.EvidenceLabel != "local" || proposal.Source != reviewPath {
		t.Fatalf("expected review-only local proposal from review file, got %#v", proposal)
	}
}

func TestPolicyCheckPassesWithEvalFixtures(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	fixtureDir := filepath.Join(project, "eval", "orchestration-policy-auditor")
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "safe-orchestrator-prompt",
		Files: map[string]string{
			"README.md": "Dry run must wait for explicit user confirmation before dispatching workers.",
		},
		ExpectedRuleHits: map[string]int{},
	})
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "dry-run-dispatch-without-approval",
		Files: map[string]string{
			"README.md": "Dry run mode can dispatch workers immediately.",
		},
		ExpectedRuleHits: map[string]int{"OPA001": 1},
	})

	report := runPolicyCheck(project, filepath.Join("eval", "orchestration-policy-auditor"))
	if report.RoutineID != "policy-check" || report.Status != "passed" {
		t.Fatalf("expected passed policy-check report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"orchestration-policy-auditor status: passed",
		"Ran 2 orchestration policy eval fixture(s)",
		"Policy eval fixtures: passed.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
}

func TestPolicyCheckFailsOnEvalFixtureMismatch(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	fixtureDir := filepath.Join(project, "eval", "orchestration-policy-auditor")
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "mismatched-fixture",
		Files: map[string]string{
			"README.md": "Dry run mode can dispatch workers immediately.",
		},
		ExpectedRuleHits: map[string]int{},
	})

	report := runPolicyCheck(project, filepath.Join("eval", "orchestration-policy-auditor"))
	if report.Status != "failed" {
		t.Fatalf("expected failed policy-check report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "mismatched-fixture: expected none, got OPA001=1.") {
		t.Fatalf("expected fixture mismatch evidence, got:\n%s", local)
	}
}

func TestCmdPolicyCheckWritesReport(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	fixtureDir := filepath.Join(project, "eval", "orchestration-policy-auditor")
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "safe-orchestrator-prompt",
		Files: map[string]string{
			"README.md": "Worker prompt: use an isolated worktree session, list allowed paths and forbidden paths, do not use subagents or Paseo, self-review, do not merge, and do not push.",
		},
		ExpectedRuleHits: map[string]int{},
	})
	reportPath := filepath.Join(root, "reports", "policy-check.json")
	if err := cmdPolicy([]string{"check", "--repo", project, "--write-report", reportPath}); err != nil {
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
	if report.RoutineID != "policy-check" || report.Status != "passed" {
		t.Fatalf("unexpected policy report: %#v", report)
	}
}

func TestEvalRunPassesWithEvalFixtures(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	fixtureDir := filepath.Join(project, "eval", "orchestration-policy-auditor")
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "safe-orchestrator-prompt",
		Files: map[string]string{
			"README.md": "Dry run must output a plan and wait for explicit user confirmation before dispatching workers.",
		},
		ExpectedRuleHits: map[string]int{},
	})
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "worker-boundary-missing",
		Files: map[string]string{
			"README.md": "Worker prompt: use a worktree branch for this task.",
		},
		ExpectedRuleHits: map[string]int{"OPA004": 1},
	})

	report := runEvalSuite(project, "orchestration-policy-auditor", "")
	if report.RoutineID != "eval-run" || report.Status != "passed" {
		t.Fatalf("expected passed eval-run report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	for _, want := range []string{
		"Eval suite: orchestration-policy-auditor",
		"Ran 2 orchestration policy eval fixture(s)",
		"Policy eval fixtures: passed.",
	} {
		if !strings.Contains(local, want) {
			t.Fatalf("expected local evidence %q in:\n%s", want, local)
		}
	}
}

func TestEvalRunFailsOnFixtureMismatch(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	fixtureDir := filepath.Join(project, "eval", "orchestration-policy-auditor")
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "mismatched-fixture",
		Files: map[string]string{
			"README.md": "Local proxy smoke counts as direct proof.",
		},
		ExpectedRuleHits: map[string]int{},
	})

	report := runEvalSuite(project, "orchestration-policy-auditor", "")
	if report.Status != "failed" {
		t.Fatalf("expected failed eval-run report, got %#v", report)
	}
	local := strings.Join(report.Evidence["local"], "\n")
	if !strings.Contains(local, "mismatched-fixture: expected none, got OPA005=1.") {
		t.Fatalf("expected fixture mismatch evidence, got:\n%s", local)
	}
}

func TestCmdEvalRunWritesReport(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	fixtureDir := filepath.Join(project, "eval", "orchestration-policy-auditor")
	writePolicyEvalFixture(t, fixtureDir, policyEvalFixture{
		SchemaVersion: 1,
		ID:            "safe-orchestrator-prompt",
		Files: map[string]string{
			"README.md": "Do not promote local or proxy evidence to direct proof.",
		},
		ExpectedRuleHits: map[string]int{},
	})
	reportPath := filepath.Join(root, "reports", "eval-run.json")
	if err := cmdEval([]string{"run", "--repo", project, "--write-report", reportPath}); err != nil {
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
	if report.RoutineID != "eval-run" || report.Status != "passed" {
		t.Fatalf("unexpected eval report: %#v", report)
	}
}

func TestEvalRunBlocksUnsupportedSuite(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	report := runEvalSuite(project, "unknown-suite", "")
	if report.Status != "blocked" || report.BlockedReason != "unsupported eval suite" {
		t.Fatalf("expected unsupported suite block, got %#v", report)
	}
}

func TestEvalAddFailureWritesVerifiedFixture(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	result, err := addFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"new-dry-run-failure",
		"dry run failure",
		"README.md",
		"Dry run mode can dispatch workers immediately.",
		"",
		[]string{"OPA001=1"},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ID != "new-dry-run-failure" || result.ExpectedRuleHits["OPA001"] != 1 || result.ActualRuleHits["OPA001"] != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
	data, err := os.ReadFile(result.Path)
	if err != nil {
		t.Fatal(err)
	}
	var fixture policyEvalFixture
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.ID != "new-dry-run-failure" || fixture.Files["README.md"] == "" || fixture.ExpectedRuleHits["OPA001"] != 1 {
		t.Fatalf("unexpected fixture: %#v", fixture)
	}
}

func TestCmdEvalAddFailureWritesJSONResult(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	if err := cmdEval([]string{
		"add-failure",
		"--repo", project,
		"--id", "new-worker-boundary-failure",
		"--text", "Worker prompt: use a worktree branch for this task.",
		"--expect", "OPA004=1",
		"--json",
	}); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(project, "eval", "orchestration-policy-auditor", "new-worker-boundary-failure.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected fixture file: %v", err)
	}
}

func TestEvalDraftFailureDoesNotWriteFixture(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	reviewPath := filepath.Join(project, "review.md")
	if err := os.WriteFile(reviewPath, []byte("The orchestrator rewrote the heartbeat prompt on every wakeup with current task details."), 0o644); err != nil {
		t.Fatal(err)
	}
	report, err := draftFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"heartbeat-prompt-churn-draft",
		"heartbeat prompt churn draft",
		"docs/reviews/heartbeat.md",
		reviewPath,
		"",
		"",
		[]string{"OPA006=1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "ready-for-approval" || !report.MatchesExpected {
		t.Fatalf("unexpected draft report: %#v", report)
	}
	if report.ActualRuleHits["OPA006"] != 1 || !strings.Contains(report.AddFailureCommand, "eval") || !strings.Contains(report.AddFailureCommand, "add-failure") {
		t.Fatalf("unexpected draft command/report: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(project, "eval", "orchestration-policy-auditor", "heartbeat-prompt-churn-draft.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("draft-failure must not write fixture, stat=%v", err)
	}
}

func TestEvalDraftFailureReportsExpectationMismatch(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	report, err := draftFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"mismatch-draft",
		"",
		"README.md",
		"",
		"Local proxy smoke counts as direct proof.",
		"",
		[]string{"OPA001=1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "needs-expectation-review" || report.MatchesExpected {
		t.Fatalf("expected mismatch draft report, got %#v", report)
	}
	if report.ActualRuleHits["OPA005"] != 1 || report.AddFailureCommand != "" {
		t.Fatalf("unexpected mismatch details: %#v", report)
	}
}

func TestCmdEvalDraftFailureWritesJSONReport(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	reportPath := filepath.Join(root, "draft.json")
	if err := cmdEval([]string{
		"draft-failure",
		"--repo", project,
		"--id", "worker-boundary-draft",
		"--text", "Worker prompt: use a worktree branch for this task.",
		"--expect", "OPA004=1",
		"--write-report", reportPath,
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatal(err)
	}
	var report EvalFailureDraftReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Command != "eval draft-failure" || report.Status != "ready-for-approval" || !report.MatchesExpected {
		t.Fatalf("unexpected report: %#v", report)
	}
	if _, err := os.Stat(filepath.Join(project, "eval", "orchestration-policy-auditor", "worker-boundary-draft.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("draft-failure command must not write fixture, stat=%v", err)
	}
}

func TestEvalAddFailureRejectsExpectationMismatch(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	_, err := addFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"bad-expectation",
		"",
		"README.md",
		"Dry run mode can dispatch workers immediately.",
		"",
		[]string{"OPA002=1"},
		false,
	)
	if err == nil || !strings.Contains(err.Error(), "expected OPA002=1, but text produced OPA001=1") {
		t.Fatalf("expected mismatch error, got %v", err)
	}
	path := filepath.Join(project, "eval", "orchestration-policy-auditor", "bad-expectation.json")
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("mismatched fixture should not be written, stat=%v", statErr)
	}
}

func TestEvalAddFailureRequiresForceForOverwrite(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	_, err := addFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"duplicate-fixture",
		"",
		"README.md",
		"Local proxy smoke counts as direct proof.",
		"",
		[]string{"OPA005=1"},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	_, err = addFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"duplicate-fixture",
		"",
		"README.md",
		"Local proxy smoke counts as direct proof.",
		"",
		[]string{"OPA005=1"},
		false,
	)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected overwrite error, got %v", err)
	}
	result, err := addFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"duplicate-fixture",
		"",
		"README.md",
		"Local proxy smoke counts as direct proof.",
		"",
		[]string{"OPA005=1"},
		true,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !result.Overwritten {
		t.Fatalf("expected overwritten result: %#v", result)
	}
}

func TestEvalAddFailureAcceptsBudgetBoundaryRule(t *testing.T) {
	root := t.TempDir()
	project := createOrchestrationPolicyFixture(t, root)
	result, err := addFailureEvalFixture(
		project,
		"orchestration-policy-auditor",
		"",
		"budget-helper-control-overclaim",
		"",
		"docs/reviews/local-budget-review.md",
		"The budget-policy-report helper automatically paused workers and enforced dispatch eligibility after heartbeat budget warnings.",
		"",
		[]string{"OPA008=1"},
		false,
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExpectedRuleHits["OPA008"] != 1 || result.ActualRuleHits["OPA008"] != 1 {
		t.Fatalf("unexpected budget boundary fixture result: %#v", result)
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

func captureStdout(t *testing.T, fn func() error) string {
	t.Helper()
	old := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	runErr := fn()
	os.Stdout = old
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(reader)
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	if runErr != nil {
		t.Fatal(runErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(data)
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
		filepath.Join("docs", "routines", "README.md"): strings.Join(readmeRoutineMentions, " "),
		filepath.Join("docs", "v2-usage.md"):           allMentions,
		filepath.Join("docs", "roadmap.md"):            allMentions,
	}
	for path, text := range docs {
		if err := os.WriteFile(filepath.Join(project, path), []byte(text+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return project
}

func createRoadmapNextTaskFixture(t *testing.T, root string, withLedger bool) (string, string) {
	t.Helper()
	project := createRepo(t, filepath.Join(root, "repo"))
	for _, dir := range []string{
		filepath.Join(project, "cmd", "codex-orchestrator"),
		filepath.Join(project, "docs"),
		filepath.Join(project, "routines"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	source := `package main

func cmdRunRoutine(args []string) error {
	switch args[0] {
	case "stale-task-rescuer":
		return nil
	case "pr-reviewer":
		return nil
	case "ci-fixer":
		return nil
	case "release-verifier":
		return nil
	case "docs-drift-checker":
		return nil
	case "evidence-label-auditor":
		return nil
	case "roadmap-next-task-suggester":
		return nil
	default:
		return nil
	}
}
`
	if err := os.WriteFile(filepath.Join(project, "cmd", "codex-orchestrator", "main.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"stale-task-rescuer",
		"pr-reviewer",
		"ci-fixer",
		"release-verifier",
		"docs-drift-checker",
		"evidence-label-auditor",
		"roadmap-next-task-suggester",
	} {
		data, err := json.Marshal(RoutineSpec{ID: id})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, "routines", id+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	roadmap := `# roadmap

## v2.5：Verification routine foundation

剩余：

- per-routine runtime budget / review budget 与 heartbeat 更深集成。

## v3：Routine library

候选 routine：

- stale task rescuer；
- PR reviewer；
- CI fixer；
- docs drift checker；
- rebase helper；
- release verifier；
- evidence label auditor deeper policy/eval variants；
- roadmap next-task suggester。
`
	if err := os.WriteFile(filepath.Join(project, "docs", "roadmap.md"), []byte(roadmap), 0o644); err != nil {
		t.Fatal(err)
	}
	ledger := filepath.Join(project, ".codex-orchestrator", "ledger.json")
	if !withLedger {
		return project, ""
	}
	if err := cmdInit([]string{"--ledger", ledger, "--project-root", project}); err != nil {
		t.Fatal(err)
	}
	if err := cmdRecordTask([]string{
		"--ledger", ledger,
		"--id", "REBASE-HELPER",
		"--title", "rebase helper",
		"--worktree", filepath.Join(root, "missing-rebase"),
		"--branch", "codex/rebase-helper",
		"--base-commit", gitOutputForTest(t, project, "rev-parse", "HEAD"),
	}); err != nil {
		t.Fatal(err)
	}
	return project, ledger
}

func createBudgetPolicyFixture(t *testing.T, root string) (string, string, string) {
	t.Helper()
	project := filepath.Join(root, "repo")
	for _, dir := range []string{
		filepath.Join(project, "docs", "routines"),
		filepath.Join(project, "routines"),
		filepath.Join(project, defaultStateDir),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	docs := map[string]string{
		filepath.Join("docs", "roadmap.md"):            "budget-policy-report remains review-only and must not dispatch enforcement, kill workers, merge, push, or cleanup worktrees.\n",
		filepath.Join("docs", "routines", "README.md"): "run-routine budget-policy-report is review-only. It reports local/static budget metadata and must not kill workers or make dispatch eligibility decisions.\n",
		filepath.Join("docs", "v2-usage.md"):           "budget-policy-report\n",
	}
	for path, text := range docs {
		if err := os.WriteFile(filepath.Join(project, path), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	specs := []RoutineSpec{
		{
			ID:                  "budgeted",
			MaxRuntimeMinutes:   10,
			ReviewBudgetMinutes: 5,
		},
		{
			ID: "missing-budget",
		},
	}
	for _, spec := range specs {
		data, err := json.Marshal(spec)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(project, "routines", spec.ID+".json"), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ledger := Ledger{
		Version:     1,
		ProjectRoot: project,
		Tasks: []Task{{
			ID:     "TASK-1",
			Status: "completed-unreviewed",
			Budget: &BudgetMetadata{
				MaxRuntimeMinutes:   10,
				ReviewBudgetMinutes: 5,
			},
			History: []map[string]string{{
				"type": "record-task",
			}},
		}},
	}
	ledgerPath := filepath.Join(project, defaultLedger)
	if err := writeJSON(ledgerPath, ledger); err != nil {
		t.Fatal(err)
	}
	heartbeat := ObserveSummary{
		BudgetSummary: BudgetSummary{
			TasksWithBudget:           1,
			TasksMissingBudget:        0,
			RoutineSpecsWithBudget:    1,
			RoutineSpecsMissingBudget: 1,
		},
		BudgetPressure: BudgetPressureSummary{
			EvidenceLabel:              "local/static",
			Warnings:                   []string{"Task TASK-1 runtime budget near limit: 8m elapsed of 10m."},
			TasksWithUnknownReviewTime: 1,
		},
	}
	heartbeatPath := filepath.Join(project, defaultStateDir, "heartbeat-report.json")
	if err := writeJSON(heartbeatPath, heartbeat); err != nil {
		t.Fatal(err)
	}
	return project, ledgerPath, heartbeatPath
}

func createEvidenceAuditFixture(t *testing.T, root string) string {
	t.Helper()
	project := filepath.Join(root, "repo")
	for _, dir := range []string{
		filepath.Join(project, "docs", "routines"),
		filepath.Join(project, "routines"),
		filepath.Join(project, "examples", "routine-reports"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	docs := map[string]string{
		"README.md":       "Local checks stay local. Do not claim direct runtime proof from static checks.",
		"README.zh-CN.md": "本地检查只作为本地证据，不要升级为直接证明。",
		"SKILL.md":        "Keep direct, proxy, local, and blocked evidence labels separate.",
		filepath.Join("docs", "routines", "README.md"): "Routine reports include direct, proxy, local, and blocked evidence buckets.",
		filepath.Join("docs", "roadmap.md"):            "The evidence-label-auditor is documented as a local/static routine.",
	}
	for path, text := range docs {
		if err := os.WriteFile(filepath.Join(project, path), []byte(text+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeEvidenceAuditRoutineSpec(t, project, "docs-drift-checker", "Reserved for future doc-render or user-facing publication proof; the MVP does not emit direct proof.")
	writeEvidenceAuditRoutineSpec(t, project, "api-proof", "Approved target API was called and returned the expected behavior.")
	report := RoutineRunReport{
		RoutineID: "docs-drift-checker",
		Status:    "passed",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   []string{"checked docs"},
			"blocked": {},
		},
		ActionsTaken:        []string{"checked docs"},
		NeedsHuman:          false,
		NextSuggestedAction: "record local report",
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "examples", "routine-reports", "docs-drift-checker.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	return project
}

func createOrchestrationPolicyFixture(t *testing.T, root string) string {
	t.Helper()
	project := filepath.Join(root, "repo")
	for _, dir := range []string{
		filepath.Join(project, "docs", "routines"),
		filepath.Join(project, "docs", "reviews"),
		filepath.Join(project, "routines"),
		filepath.Join(project, "examples", "routine-reports"),
		filepath.Join(project, ".codex-orchestrator"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	docs := map[string]string{
		"README.md": strings.Join([]string{
			"Dry run must output a plan and wait for explicit user confirmation before dispatching workers.",
			"If setup failed, report a blocker; do not implement in the orchestrator checkout.",
			"When a child task is completed, inspect ledger, roadmap, repo truth, and queue before deleting heartbeat or stopping.",
			"Worker prompt: use an isolated worktree session, list allowed paths and forbidden paths, do not use subagents or Paseo, self-review, do not merge, and do not push.",
			"Do not promote local or proxy evidence to direct proof.",
		}, "\n\n"),
		"README.zh-CN.md": "dry run 后等待确认；worker 使用独立 worktree；不要把 local/proxy 写成 direct。",
		"SKILL.md":        "Worker prompt must require isolated worktree, no subagents/Paseo, self-review, and no merge/push/cleanup.",
		filepath.Join("docs", "routines", "README.md"): "The orchestration-policy-auditor scans local docs for dry-run barrier, fallback guard, continuation guard, worker boundary, and evidence boundary issues.",
		filepath.Join("docs", "roadmap.md"):            "V4 policy/eval adds orchestration-policy-auditor as a local static policy checker.",
		filepath.Join("docs", "reviews", "safe.md"):    "Completed worker cleanup only after ledger and queue checks.",
	}
	for path, text := range docs {
		if err := os.WriteFile(filepath.Join(project, path), []byte(text+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeEvidenceAuditRoutineSpec(t, project, "orchestration-policy-auditor", "Reserved for future human-reviewed orchestration transcript proof; the MVP does not emit direct proof.")
	report := RoutineRunReport{
		RoutineID: "orchestration-policy-auditor",
		Status:    "passed",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   []string{"checked orchestration docs"},
			"blocked": {},
		},
		ActionsTaken:        []string{"checked orchestration docs"},
		NeedsHuman:          false,
		NextSuggestedAction: "record local report",
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "examples", "routine-reports", "orchestration-policy-auditor.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, ".codex-orchestrator", "events.jsonl"), []byte(`{"type":"heartbeat","status":"review-needed"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return project
}

func writePolicyEvalFixture(t *testing.T, dir string, fixture policyEvalFixture) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, fixture.ID+".json")
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func containsAuthorizationStatus(matrix []AuthorizationCheck, action string, status string) bool {
	for _, entry := range matrix {
		if entry.Action == action && entry.Status == status {
			return true
		}
	}
	return false
}

func writeEvidenceAuditRoutineSpec(t *testing.T, project string, id string, directEvidence string) {
	t.Helper()
	spec := RoutineSpec{
		SchemaVersion:    1,
		ID:               id,
		Title:            id,
		Purpose:          "fixture routine",
		Trigger:          "test",
		Inputs:           []string{"fixture"},
		AllowedActions:   []string{"inspect read-only"},
		ForbiddenActions: []string{"stage, commit, merge, push, or mutate ledger"},
		Gates:            []string{"fixture gate"},
		EvidenceLabels:   []string{"direct", "proxy", "local", "blocked"},
		OutputSchema: RoutineOutputSpec{
			RequiredFields: []string{"status", "evidence", "actionsTaken", "needsHuman", "blockedReason", "nextSuggestedAction"},
			StatusValues:   []string{"passed", "failed", "blocked"},
			Evidence: map[string]string{
				"direct":  directEvidence,
				"proxy":   "Accepted indirect evidence.",
				"local":   "Local source or fixture inspection.",
				"blocked": "Claim could not be proven safely.",
			},
		},
		Escalation: []string{"stop if fixture is unavailable"},
	}
	data, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "routines", id+".json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}
