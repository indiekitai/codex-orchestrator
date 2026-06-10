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

type ObserveSummary struct {
	Ledger        string        `json:"ledger"`
	Version       int           `json:"version"`
	ProjectRoot   string        `json:"projectRoot"`
	DefaultBranch string        `json:"defaultBranch"`
	ObservedAt    string        `json:"observedAt"`
	Observations  []Observation `json:"observations"`
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
  codex-orchestrator observe [--ledger PATH] [--json] [--write-report PATH]
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
	if err := fs.Parse(args); err != nil {
		return err
	}
	summary, err := observe(*ledgerPath)
	if err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, summary); err != nil {
			return err
		}
	}
	if *jsonOut {
		return printJSON(summary)
	}
	printObservations(summary)
	return nil
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
	ledger, err := loadLedger(ledgerPath)
	if err != nil {
		return ObserveSummary{}, err
	}
	observations := make([]Observation, 0, len(ledger.Tasks))
	for _, task := range ledger.Tasks {
		observations = append(observations, inspectTask(task))
	}
	return ObserveSummary{
		Ledger:        ledgerPath,
		Version:       ledger.Version,
		ProjectRoot:   ledger.ProjectRoot,
		DefaultBranch: ledger.DefaultBranch,
		ObservedAt:    nowISO(),
		Observations:  observations,
	}, nil
}

func inspectTask(task Task) Observation {
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
		return Observation{
			ID:     task.ID,
			Status: "pending-setup",
			Action: "verify setup or mark stale if expired",
			Note:   fmt.Sprintf("Worktree does not exist: %s", worktree),
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
		return Observation{
			ID:        task.ID,
			Status:    statusValue,
			Action:    "inspect manually",
			Note:      "Could not compare baseCommit; ledger may be a template or base is missing.",
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
