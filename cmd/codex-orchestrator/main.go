package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultStateDir = ".codex-orchestrator"
	defaultLedger   = defaultStateDir + "/ledger.json"
	defaultEvents   = defaultStateDir + "/events.jsonl"
)

type Ledger struct {
	Version        int    `json:"version"`
	ProjectRoot    string `json:"projectRoot"`
	DefaultBranch  string `json:"defaultBranch"`
	Remote         string `json:"remote"`
	PushPolicy     string `json:"pushPolicy"`
	MaxConcurrency int    `json:"maxConcurrency"`
	CreatedAt      string `json:"createdAt"`
	UpdatedAt      string `json:"updatedAt"`
	Tasks          []Task `json:"tasks"`
}

type Task struct {
	ID              string              `json:"id"`
	Title           string              `json:"title,omitempty"`
	ThreadID        string              `json:"threadId,omitempty"`
	Worktree        string              `json:"worktree"`
	Branch          string              `json:"branch"`
	BaseCommit      string              `json:"baseCommit,omitempty"`
	Status          string              `json:"status"`
	WriteSet        map[string][]string `json:"writeSet,omitempty"`
	Gates           []string            `json:"gates,omitempty"`
	Evidence        map[string]any      `json:"evidence,omitempty"`
	LastObservation map[string]string   `json:"lastObservation,omitempty"`
	History         []map[string]string `json:"history,omitempty"`
}

type stringList []string

func (s *stringList) String() string {
	return strings.Join(*s, ",")
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type Observation struct {
	ID        string `json:"id,omitempty"`
	Status    string `json:"status"`
	Action    string `json:"action"`
	Note      string `json:"note"`
	GitStatus string `json:"gitStatus,omitempty"`
}

type IntegrationState struct {
	Path      string `json:"path"`
	Dirty     bool   `json:"dirty"`
	GitStatus string `json:"gitStatus,omitempty"`
	Error     string `json:"error,omitempty"`
}

type ReviewPressure struct {
	MaxConcurrency   int `json:"maxConcurrency"`
	Active           int `json:"active"`
	PendingSetup     int `json:"pendingSetup"`
	ReviewNeeded     int `json:"reviewNeeded"`
	Stale            int `json:"stale"`
	Blocked          int `json:"blocked"`
	CleanupNeeded    int `json:"cleanupNeeded"`
	AvailableSlots   int `json:"availableSlots"`
	ReviewQueueLimit int `json:"reviewQueueLimit"`
}

type ObserveSummary struct {
	Ledger             string           `json:"ledger"`
	Version            int              `json:"version"`
	ProjectRoot        string           `json:"projectRoot"`
	DefaultBranch      string           `json:"defaultBranch"`
	ObservedAt         string           `json:"observedAt"`
	OverallStatus      string           `json:"overallStatus"`
	RecommendedActions []string         `json:"recommendedActions"`
	Counts             map[string]int   `json:"counts"`
	ReviewPressure     ReviewPressure   `json:"reviewPressure"`
	Integration        IntegrationState `json:"integration"`
	Observations       []Observation    `json:"observations"`
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	switch args[0] {
	case "init":
		return cmdInit(args[1:])
	case "record-task":
		return cmdRecordTask(args[1:])
	case "append-event":
		return cmdAppendEvent(args[1:])
	case "observe":
		return cmdObserve(args[1:])
	case "heartbeat":
		return cmdHeartbeat(args[1:])
	case "status":
		return cmdStatus(args[1:])
	case "help", "-h", "--help":
		usage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func usage() {
	fmt.Println(`codex-orchestrator v2 helper

Usage:
  codex-orchestrator init [--ledger PATH] [--project-root PATH]
  codex-orchestrator record-task --id ID --worktree PATH --branch BRANCH [--allowed PATH] [--forbidden PATH] [--gate CMD]
  codex-orchestrator append-event --type TYPE [--task-id ID] [--status STATUS] [--note TEXT]
  codex-orchestrator observe [--ledger PATH] [--json] [--write-report PATH] [--write-summary PATH]
  codex-orchestrator heartbeat [--ledger PATH] [--interval 5m] [--count 0] [--write-report PATH]
  codex-orchestrator status [--ledger PATH] [--json]

This helper is conservative: it does not create Codex sessions, merge, push,
delete branches, or clean worktrees.`)
}

func cmdInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	eventsPath := fs.String("events", "", "events path")
	projectRoot := fs.String("project-root", ".", "project root")
	defaultBranchValue := fs.String("default-branch", "", "default branch")
	remote := fs.String("remote", "origin", "remote")
	pushPolicy := fs.String("push-policy", "manual", "push policy")
	maxConcurrency := fs.Int("max-concurrency", 2, "max concurrency")
	force := fs.Bool("force", false, "overwrite existing ledger")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := os.Stat(expandPath(*ledgerPath)); err == nil && !*force {
		return fmt.Errorf("ledger already exists: %s (use --force to overwrite)", *ledgerPath)
	}
	root, err := filepath.Abs(*projectRoot)
	if err != nil {
		return err
	}
	branch := *defaultBranchValue
	if branch == "" {
		branch = defaultBranch(root)
	}
	now := nowISO()
	ledger := Ledger{
		Version:        1,
		ProjectRoot:    root,
		DefaultBranch:  branch,
		Remote:         *remote,
		PushPolicy:     *pushPolicy,
		MaxConcurrency: *maxConcurrency,
		CreatedAt:      now,
		UpdatedAt:      now,
		Tasks:          []Task{},
	}
	if err := writeJSON(*ledgerPath, ledger); err != nil {
		return err
	}
	resolvedEvents := *eventsPath
	if resolvedEvents == "" {
		resolvedEvents = eventsPathForLedger(*ledgerPath)
	}
	if err := appendEvent(resolvedEvents, map[string]any{
		"at":     now,
		"type":   "init",
		"status": "created",
		"ledger": *ledgerPath,
	}); err != nil {
		return err
	}
	fmt.Printf("Initialized ledger: %s\n", *ledgerPath)
	fmt.Printf("Initialized events: %s\n", resolvedEvents)
	return nil
}

func cmdRecordTask(args []string) error {
	fs := flag.NewFlagSet("record-task", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	eventsPath := fs.String("events", "", "events path")
	id := fs.String("id", "", "task id")
	title := fs.String("title", "", "task title")
	threadID := fs.String("thread-id", "", "Codex thread id")
	worktree := fs.String("worktree", "", "task worktree path")
	branch := fs.String("branch", "", "task branch")
	baseCommit := fs.String("base-commit", "", "base commit")
	status := fs.String("status", "active", "task status")
	evidence := fs.String("evidence", "local", "expected evidence type")
	evidenceNote := fs.String("evidence-note", "", "evidence note")
	note := fs.String("note", "", "history note")
	var allowed stringList
	var forbidden stringList
	var gates stringList
	fs.Var(&allowed, "allowed", "allowed write path, repeatable")
	fs.Var(&forbidden, "forbidden", "forbidden path, repeatable")
	fs.Var(&gates, "gate", "verification gate, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *id == "" {
		return errors.New("record-task requires --id")
	}
	if *worktree == "" {
		return errors.New("record-task requires --worktree")
	}
	if *branch == "" {
		return errors.New("record-task requires --branch")
	}
	ledger, err := loadLedger(*ledgerPath)
	if err != nil {
		return err
	}
	if findTaskIndex(ledger.Tasks, *id) >= 0 {
		return fmt.Errorf("task already exists: %s", *id)
	}
	base := *baseCommit
	if base == "" {
		base = headCommit(ledger.ProjectRoot)
	}
	now := nowISO()
	taskTitle := *title
	if taskTitle == "" {
		taskTitle = *id
	}
	historyNote := *note
	if historyNote == "" {
		historyNote = "Task recorded."
	}
	task := Task{
		ID:         *id,
		Title:      taskTitle,
		ThreadID:   *threadID,
		Worktree:   *worktree,
		Branch:     *branch,
		BaseCommit: base,
		Status:     *status,
		WriteSet: map[string][]string{
			"allowed":   []string(allowed),
			"forbidden": []string(forbidden),
		},
		Gates: []string(gates),
		Evidence: map[string]any{
			"expected": *evidence,
			"labels":   []string{"direct", "proxy", "blocked"},
			"notes":    *evidenceNote,
		},
		LastObservation: map[string]string{
			"at":     now,
			"result": *status,
			"note":   "Task recorded.",
		},
		History: []map[string]string{{
			"at":     now,
			"type":   "record-task",
			"status": *status,
			"note":   historyNote,
		}},
	}
	ledger.Tasks = append(ledger.Tasks, task)
	if err := saveLedger(*ledgerPath, &ledger); err != nil {
		return err
	}
	resolvedEvents := *eventsPath
	if resolvedEvents == "" {
		resolvedEvents = eventsPathForLedger(*ledgerPath)
	}
	if err := appendEvent(resolvedEvents, map[string]any{
		"at":     nowISO(),
		"type":   "record-task",
		"taskId": *id,
		"status": *status,
	}); err != nil {
		return err
	}
	fmt.Printf("Recorded task: %s\n", *id)
	return nil
}

func cmdAppendEvent(args []string) error {
	fs := flag.NewFlagSet("append-event", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	eventsPath := fs.String("events", "", "events path")
	taskID := fs.String("task-id", "", "task id")
	eventType := fs.String("type", "", "event type")
	status := fs.String("status", "", "status")
	note := fs.String("note", "", "event note")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *eventType == "" {
		return errors.New("append-event requires --type")
	}
	ledger, err := loadLedger(*ledgerPath)
	if err != nil {
		return err
	}
	taskIndex := -1
	if *taskID != "" {
		taskIndex = findTaskIndex(ledger.Tasks, *taskID)
		if taskIndex < 0 {
			return fmt.Errorf("task not found: %s", *taskID)
		}
	}
	now := nowISO()
	event := map[string]any{
		"at":     now,
		"type":   *eventType,
		"status": emptyToNil(*status),
		"taskId": emptyToNil(*taskID),
		"note":   *note,
	}
	resolvedEvents := *eventsPath
	if resolvedEvents == "" {
		resolvedEvents = eventsPathForLedger(*ledgerPath)
	}
	if err := appendEvent(resolvedEvents, event); err != nil {
		return err
	}
	if *taskID != "" {
		task := &ledger.Tasks[taskIndex]
		if *status != "" {
			task.Status = *status
		}
		task.LastObservation = map[string]string{
			"at":     now,
			"result": task.Status,
			"note":   *note,
		}
		task.History = append(task.History, compactEvent(event))
		if err := saveLedger(*ledgerPath, &ledger); err != nil {
			return err
		}
	}
	fmt.Printf("Appended event: %s\n", *eventType)
	return nil
}

func cmdObserve(args []string) error {
	fs := flag.NewFlagSet("observe", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	jsonOut := fs.Bool("json", false, "print JSON")
	writeReport := fs.String("write-report", "", "write JSON report")
	writeSummary := fs.String("write-summary", "", "write Markdown summary")
	staleAfter := fs.Duration("stale-after", 15*time.Minute, "stale threshold")
	if err := fs.Parse(args); err != nil {
		return err
	}
	summary, err := observeWithOptions(*ledgerPath, *staleAfter)
	if err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, summary); err != nil {
			return err
		}
	}
	if *writeSummary != "" {
		if err := writeText(*writeSummary, renderSummary(summary)); err != nil {
			return err
		}
	}
	if *jsonOut {
		return printJSON(summary)
	}
	printObservations(summary)
	return nil
}

func cmdHeartbeat(args []string) error {
	fs := flag.NewFlagSet("heartbeat", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	eventsPath := fs.String("events", "", "events path")
	jsonOut := fs.Bool("json", false, "print JSON")
	writeReport := fs.String("write-report", "", "write latest JSON report")
	writeSummary := fs.String("write-summary", "", "write latest Markdown summary")
	interval := fs.Duration("interval", 5*time.Minute, "heartbeat interval")
	count := fs.Int("count", 1, "number of checks; 0 runs forever")
	staleAfter := fs.Duration("stale-after", 15*time.Minute, "stale threshold")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *count < 0 {
		return errors.New("heartbeat --count cannot be negative")
	}
	if *interval < 0 {
		return errors.New("heartbeat --interval cannot be negative")
	}
	resolvedEvents := *eventsPath
	if resolvedEvents == "" {
		resolvedEvents = eventsPathForLedger(*ledgerPath)
	}
	iteration := 0
	for {
		iteration++
		summary, err := observeWithOptions(*ledgerPath, *staleAfter)
		if err != nil {
			return err
		}
		if *writeReport != "" {
			if err := writeJSON(*writeReport, summary); err != nil {
				return err
			}
		}
		if *writeSummary != "" {
			if err := writeText(*writeSummary, renderSummary(summary)); err != nil {
				return err
			}
		}
		if err := appendEvent(resolvedEvents, map[string]any{
			"at":     summary.ObservedAt,
			"type":   "heartbeat",
			"status": summary.OverallStatus,
			"note":   strings.Join(summary.RecommendedActions, " | "),
		}); err != nil {
			return err
		}
		if *jsonOut {
			if err := printJSON(summary); err != nil {
				return err
			}
		} else {
			printObservations(summary)
		}
		if *count > 0 && iteration >= *count {
			return nil
		}
		if *interval == 0 {
			return nil
		}
		time.Sleep(*interval)
	}
}

func cmdStatus(args []string) error {
	fs := flag.NewFlagSet("status", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ledger, err := loadLedger(*ledgerPath)
	if err != nil {
		return err
	}
	counts := map[string]int{}
	for _, task := range ledger.Tasks {
		status := task.Status
		if status == "" {
			status = "unknown"
		}
		counts[status]++
	}
	result := map[string]any{
		"ledger":        *ledgerPath,
		"projectRoot":   ledger.ProjectRoot,
		"defaultBranch": ledger.DefaultBranch,
		"taskCount":     len(ledger.Tasks),
		"counts":        counts,
	}
	if *jsonOut {
		return printJSON(result)
	}
	fmt.Printf("Ledger: %s\n", *ledgerPath)
	fmt.Printf("Project: %s default=%s\n", ledger.ProjectRoot, ledger.DefaultBranch)
	fmt.Printf("Tasks: %d\n", len(ledger.Tasks))
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("- %s: %d\n", key, counts[key])
	}
	return nil
}

func observe(ledgerPath string) (ObserveSummary, error) {
	return observeWithOptions(ledgerPath, 15*time.Minute)
}

func observeWithOptions(ledgerPath string, staleAfter time.Duration) (ObserveSummary, error) {
	ledger, err := loadLedger(ledgerPath)
	if err != nil {
		return ObserveSummary{}, err
	}
	observations := make([]Observation, 0, len(ledger.Tasks))
	for _, task := range ledger.Tasks {
		observations = append(observations, inspectTask(task, staleAfter))
	}
	integration := inspectIntegration(ledger.ProjectRoot)
	counts := countObservationStatuses(observations)
	pressure := calculateReviewPressure(counts, ledger.MaxConcurrency)
	overall, actions := summarizeObservations(integration, counts, pressure)
	return ObserveSummary{
		Ledger:             ledgerPath,
		Version:            ledger.Version,
		ProjectRoot:        ledger.ProjectRoot,
		DefaultBranch:      ledger.DefaultBranch,
		ObservedAt:         nowISO(),
		OverallStatus:      overall,
		RecommendedActions: actions,
		Counts:             counts,
		ReviewPressure:     pressure,
		Integration:        integration,
		Observations:       observations,
	}, nil
}

func inspectTask(task Task, staleAfter time.Duration) Observation {
	if isTerminalStatus(task.Status) {
		if task.Worktree != "" {
			worktree := expandPath(task.Worktree)
			if _, err := os.Stat(worktree); err == nil && task.Status != "rejected" {
				return Observation{
					ID:     task.ID,
					Status: "cleanup-needed",
					Action: "remove accepted task worktree and delete local task branch if safe",
					Note:   fmt.Sprintf("Task is %s but worktree still exists: %s", task.Status, worktree),
				}
			}
		}
		return Observation{
			ID:     task.ID,
			Status: task.Status,
			Action: "quiet",
			Note:   fmt.Sprintf("Task is recorded as %s.", task.Status),
		}
	}
	if task.Worktree == "" {
		return Observation{
			ID:     task.ID,
			Status: "blocked",
			Action: "record missing worktree path",
			Note:   "Task has no worktree path in ledger.",
		}
	}
	worktree := expandPath(task.Worktree)
	if _, err := os.Stat(worktree); err != nil {
		statusValue := "pending-setup"
		action := "verify setup or mark stale if expired"
		note := fmt.Sprintf("Worktree does not exist: %s", worktree)
		if isTaskStale(task, staleAfter) {
			statusValue = "stale-needs-inspection"
			action = "inspect pending setup and decide whether to re-dispatch or abandon"
			note = fmt.Sprintf("Worktree does not exist and the last observation is older than %s: %s", staleAfter, worktree)
		}
		return Observation{
			ID:     task.ID,
			Status: statusValue,
			Action: action,
			Note:   note,
		}
	}
	status, err := gitOutput(worktree, "status", "--short", "--branch")
	if err != nil {
		return Observation{
			ID:     task.ID,
			Status: "blocked",
			Action: "inspect worktree git state",
			Note:   err.Error(),
		}
	}
	branch := currentBranch(status)
	if task.Branch != "" && branch != "" && branch != task.Branch {
		return Observation{
			ID:        task.ID,
			Status:    "blocked",
			Action:    "fix branch mismatch before review",
			Note:      fmt.Sprintf("Expected %s, found %s.", task.Branch, branch),
			GitStatus: status,
		}
	}
	if hasDirtyChanges(status) {
		return Observation{
			ID:        task.ID,
			Status:    "stale-needs-inspection",
			Action:    "inspect uncommitted scoped diff or nudge same worker",
			Note:      "Worktree has uncommitted changes.",
			GitStatus: status,
		}
	}
	commitsAfterBase, known := hasCommitsAfterBase(worktree, task.BaseCommit)
	if known && commitsAfterBase {
		return Observation{
			ID:        task.ID,
			Status:    "completed-unreviewed",
			Action:    "orchestrator review required before merge",
			Note:      "Clean worktree has commits after baseCommit.",
			GitStatus: status,
		}
	}
	if !known {
		statusValue := task.Status
		if statusValue == "" {
			statusValue = "active"
		}
		if statusValue == "active" && isTaskStale(task, staleAfter) {
			return Observation{
				ID:        task.ID,
				Status:    "stale-needs-inspection",
				Action:    "inspect manually",
				Note:      fmt.Sprintf("Task has no comparable baseCommit and the last observation is older than %s.", staleAfter),
				GitStatus: status,
			}
		}
		return Observation{
			ID:        task.ID,
			Status:    statusValue,
			Action:    "inspect manually",
			Note:      "Could not compare baseCommit; ledger may be a template or base is missing.",
			GitStatus: status,
		}
	}
	if task.Status == "active" && isTaskStale(task, staleAfter) {
		return Observation{
			ID:        task.ID,
			Status:    "stale-needs-inspection",
			Action:    "inspect recent thread messages or nudge same worker",
			Note:      fmt.Sprintf("Clean worktree has no commits after baseCommit, and last observation is older than %s.", staleAfter),
			GitStatus: status,
		}
	}
	return Observation{
		ID:        task.ID,
		Status:    "active",
		Action:    "quiet",
		Note:      "Clean worktree has no commits after baseCommit.",
		GitStatus: status,
	}
}

func loadLedger(path string) (Ledger, error) {
	var ledger Ledger
	data, err := os.ReadFile(expandPath(path))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ledger, fmt.Errorf("ledger not found: %s (run init first)", path)
		}
		return ledger, err
	}
	if err := json.Unmarshal(data, &ledger); err != nil {
		return ledger, err
	}
	return ledger, nil
}

func saveLedger(path string, ledger *Ledger) error {
	ledger.UpdatedAt = nowISO()
	return writeJSON(path, ledger)
}

func findTaskIndex(tasks []Task, id string) int {
	for index, task := range tasks {
		if task.ID == id {
			return index
		}
	}
	return -1
}

func inspectIntegration(projectRoot string) IntegrationState {
	root := expandPath(projectRoot)
	if root == "" {
		root = "."
	}
	state := IntegrationState{Path: root}
	status, err := gitOutput(root, "status", "--short", "--branch")
	if err != nil {
		state.Error = err.Error()
		return state
	}
	state.GitStatus = status
	state.Dirty = hasDirtyChangesIgnoringStateDir(status)
	return state
}

func isTaskStale(task Task, threshold time.Duration) bool {
	if threshold <= 0 {
		return false
	}
	at := task.LastObservation["at"]
	if at == "" && len(task.History) > 0 {
		at = task.History[len(task.History)-1]["at"]
	}
	if at == "" {
		return false
	}
	observed, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return false
	}
	return time.Since(observed) > threshold
}

func countObservationStatuses(observations []Observation) map[string]int {
	counts := map[string]int{}
	for _, observation := range observations {
		status := observation.Status
		if status == "" {
			status = "unknown"
		}
		counts[status]++
	}
	return counts
}

func calculateReviewPressure(counts map[string]int, maxConcurrency int) ReviewPressure {
	if maxConcurrency <= 0 {
		maxConcurrency = 2
	}
	active := counts["active"]
	pending := counts["pending-setup"]
	usedSlots := active + pending + counts["reviewing"]
	available := maxConcurrency - usedSlots
	if available < 0 {
		available = 0
	}
	return ReviewPressure{
		MaxConcurrency:   maxConcurrency,
		Active:           active,
		PendingSetup:     pending,
		ReviewNeeded:     counts["completed-unreviewed"],
		Stale:            counts["stale-needs-inspection"],
		Blocked:          counts["blocked"],
		CleanupNeeded:    counts["cleanup-needed"],
		AvailableSlots:   available,
		ReviewQueueLimit: 1,
	}
}

func summarizeObservations(integration IntegrationState, counts map[string]int, pressure ReviewPressure) (string, []string) {
	if integration.Error != "" {
		return "blocked", []string{"Inspect integration checkout git state before dispatching or merging."}
	}
	if integration.Dirty {
		return "blocked", []string{"Integration checkout is dirty; classify local changes before dispatching or merging."}
	}
	switch {
	case counts["blocked"] > 0:
		return "blocked", []string{"Resolve blocked task setup/state before dispatching new work."}
	case counts["completed-unreviewed"] > 0:
		if pressure.ReviewNeeded > pressure.ReviewQueueLimit {
			return "review-needed", []string{"Review queue is saturated; review completed task commits before dispatching more work."}
		}
		return "review-needed", []string{"Review completed task commits, run gates, then merge or reject."}
	case counts["cleanup-needed"] > 0:
		return "cleanup-needed", []string{"Cleanup accepted task worktrees/branches before dispatching more work."}
	case counts["stale-needs-inspection"] > 0:
		return "stale", []string{"Inspect stale task worktrees and either nudge, take over, abandon, or mark blocked."}
	}
	if pressure.AvailableSlots > 0 {
		return "dispatch-possible", []string{"Capacity is available; dispatch the next safe roadmap task if one exists."}
	}
	return "quiet", []string{"Active tasks are within concurrency limit; continue monitoring."}
}

func isTerminalStatus(status string) bool {
	switch status {
	case "merged", "rejected", "abandoned":
		return true
	default:
		return false
	}
}

func writeJSON(path string, value any) error {
	target := expandPath(path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(target, data, 0o644)
}

func writeText(path string, value string) error {
	target := expandPath(path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if !strings.HasSuffix(value, "\n") {
		value += "\n"
	}
	return os.WriteFile(target, []byte(value), 0o644)
}

func appendEvent(path string, event map[string]any) error {
	target := expandPath(path)
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	f, err := os.OpenFile(target, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func renderSummary(summary ObserveSummary) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# codex-orchestrator heartbeat\n\n")
	fmt.Fprintf(&b, "- observedAt: `%s`\n", summary.ObservedAt)
	fmt.Fprintf(&b, "- overallStatus: `%s`\n", summary.OverallStatus)
	fmt.Fprintf(&b, "- ledger: `%s`\n", summary.Ledger)
	fmt.Fprintf(&b, "- projectRoot: `%s`\n", summary.ProjectRoot)
	fmt.Fprintf(&b, "- defaultBranch: `%s`\n", summary.DefaultBranch)
	fmt.Fprintf(&b, "- integrationDirty: `%t`\n", summary.Integration.Dirty)
	fmt.Fprintf(&b, "- active: `%d`\n", summary.ReviewPressure.Active)
	fmt.Fprintf(&b, "- reviewNeeded: `%d`\n", summary.ReviewPressure.ReviewNeeded)
	fmt.Fprintf(&b, "- stale: `%d`\n", summary.ReviewPressure.Stale)
	fmt.Fprintf(&b, "- blocked: `%d`\n", summary.ReviewPressure.Blocked)
	fmt.Fprintf(&b, "- cleanupNeeded: `%d`\n", summary.ReviewPressure.CleanupNeeded)
	fmt.Fprintf(&b, "- availableSlots: `%d`\n", summary.ReviewPressure.AvailableSlots)
	if summary.Integration.Error != "" {
		fmt.Fprintf(&b, "- integrationError: `%s`\n", summary.Integration.Error)
	}
	if len(summary.RecommendedActions) > 0 {
		fmt.Fprintf(&b, "\n## Recommended Actions\n\n")
		for _, action := range summary.RecommendedActions {
			fmt.Fprintf(&b, "- %s\n", action)
		}
	}
	fmt.Fprintf(&b, "\n## Tasks\n\n")
	if len(summary.Observations) == 0 {
		fmt.Fprintf(&b, "- No tasks recorded.\n")
		return b.String()
	}
	for _, item := range summary.Observations {
		fmt.Fprintf(&b, "- `%s`: `%s` - %s\n", item.ID, item.Status, item.Action)
		if item.Note != "" {
			fmt.Fprintf(&b, "  - note: %s\n", item.Note)
		}
	}
	return b.String()
}

func compactEvent(event map[string]any) map[string]string {
	result := map[string]string{}
	for _, key := range []string{"at", "type", "status", "taskId", "note"} {
		value, ok := event[key]
		if !ok || value == nil {
			continue
		}
		text := fmt.Sprint(value)
		if text != "" {
			result[key] = text
		}
	}
	return result
}

func emptyToNil(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func printJSON(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func printObservations(summary ObserveSummary) {
	fmt.Printf("Ledger: %s\n", summary.Ledger)
	fmt.Printf("Project: %s default=%s\n", summary.ProjectRoot, summary.DefaultBranch)
	fmt.Printf("Overall: %s\n", summary.OverallStatus)
	if summary.Integration.Error != "" {
		fmt.Printf("Integration: blocked (%s)\n", summary.Integration.Error)
	} else {
		fmt.Printf("Integration dirty: %t\n", summary.Integration.Dirty)
	}
	for _, action := range summary.RecommendedActions {
		fmt.Printf("Action: %s\n", action)
	}
	for _, item := range summary.Observations {
		fmt.Println()
		fmt.Printf("- %s: %s\n", item.ID, item.Status)
		fmt.Printf("  action: %s\n", item.Action)
		fmt.Printf("  note: %s\n", item.Note)
		if item.GitStatus != "" {
			fmt.Println("  git:")
			for _, line := range strings.Split(item.GitStatus, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
	}
}

func defaultBranch(repo string) string {
	if branch, err := gitOutput(repo, "branch", "--show-current"); err == nil && strings.TrimSpace(branch) != "" {
		return strings.TrimSpace(branch)
	}
	if ref, err := gitOutput(repo, "symbolic-ref", "--short", "refs/remotes/origin/HEAD"); err == nil && strings.Contains(ref, "/") {
		parts := strings.SplitN(strings.TrimSpace(ref), "/", 2)
		return parts[1]
	}
	return "main"
}

func headCommit(repo string) string {
	if repo == "" {
		repo = "."
	}
	if commit, err := gitOutput(expandPath(repo), "rev-parse", "HEAD"); err == nil {
		return strings.TrimSpace(commit)
	}
	return ""
}

func eventsPathForLedger(ledgerPath string) string {
	return filepath.Join(filepath.Dir(expandPath(ledgerPath)), "events.jsonl")
}

func gitOutput(cwd string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func currentBranch(statusOutput string) string {
	lines := strings.Split(statusOutput, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "## ") {
		return ""
	}
	branch := strings.TrimSpace(strings.TrimPrefix(lines[0], "## "))
	branch = strings.SplitN(branch, "...", 2)[0]
	if branch == "HEAD (no branch)" {
		return ""
	}
	return branch
}

func hasDirtyChanges(statusOutput string) bool {
	for _, line := range strings.Split(statusOutput, "\n") {
		if line != "" && !strings.HasPrefix(line, "## ") {
			return true
		}
	}
	return false
}

func hasDirtyChangesIgnoringStateDir(statusOutput string) bool {
	for _, line := range strings.Split(statusOutput, "\n") {
		if line == "" || strings.HasPrefix(line, "## ") {
			continue
		}
		if strings.HasPrefix(line, "?? "+defaultStateDir+"/") || strings.HasPrefix(line, "?? "+defaultStateDir) {
			continue
		}
		if strings.Contains(line, " "+defaultStateDir+"/") {
			continue
		}
		return true
	}
	return false
}

func hasCommitsAfterBase(worktree string, baseCommit string) (bool, bool) {
	if baseCommit == "" || allZeros(baseCommit) {
		return false, false
	}
	out, err := gitOutput(worktree, "rev-list", "--count", baseCommit+"..HEAD")
	if err != nil {
		return false, false
	}
	return strings.TrimSpace(out) != "0", true
}

func allZeros(value string) bool {
	for _, ch := range value {
		if ch != '0' {
			return false
		}
	}
	return value != ""
}

func expandPath(path string) string {
	if path == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
	}
	if strings.HasPrefix(path, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}
