package main

import (
	"context"
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
	"unicode"
)

const (
	defaultStateDir = ".codex-orchestrator"
	defaultLedger   = defaultStateDir + "/ledger.json"
	defaultEvents   = defaultStateDir + "/events.jsonl"
)

type Ledger struct {
	Version        int          `json:"version"`
	ProjectRoot    string       `json:"projectRoot"`
	DefaultBranch  string       `json:"defaultBranch"`
	Remote         string       `json:"remote"`
	PushPolicy     string       `json:"pushPolicy"`
	MaxConcurrency int          `json:"maxConcurrency"`
	CreatedAt      string       `json:"createdAt"`
	UpdatedAt      string       `json:"updatedAt"`
	Tasks          []Task       `json:"tasks"`
	RoutineRuns    []RoutineRun `json:"routineRuns,omitempty"`
}

type Task struct {
	ID              string              `json:"id"`
	Title           string              `json:"title,omitempty"`
	ThreadID        string              `json:"threadId,omitempty"`
	Worktree        string              `json:"worktree"`
	Branch          string              `json:"branch"`
	BaseCommit      string              `json:"baseCommit,omitempty"`
	Status          string              `json:"status"`
	Budget          *BudgetMetadata     `json:"budget,omitempty"`
	WriteSet        map[string][]string `json:"writeSet,omitempty"`
	Gates           []string            `json:"gates,omitempty"`
	Evidence        map[string]any      `json:"evidence,omitempty"`
	LastObservation map[string]string   `json:"lastObservation,omitempty"`
	History         []map[string]string `json:"history,omitempty"`
}

type BudgetMetadata struct {
	MaxRuntimeMinutes   int    `json:"maxRuntimeMinutes,omitempty"`
	ReviewBudgetMinutes int    `json:"reviewBudgetMinutes,omitempty"`
	Note                string `json:"note,omitempty"`
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
	ID        string          `json:"id,omitempty"`
	Status    string          `json:"status"`
	Action    string          `json:"action"`
	Note      string          `json:"note"`
	GitStatus string          `json:"gitStatus,omitempty"`
	Budget    *BudgetMetadata `json:"budget,omitempty"`
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

type BudgetSummary struct {
	TasksWithBudget          int `json:"tasksWithBudget"`
	TasksMissingBudget       int `json:"tasksMissingBudget"`
	TotalMaxRuntimeMinutes   int `json:"totalMaxRuntimeMinutes,omitempty"`
	TotalReviewBudgetMinutes int `json:"totalReviewBudgetMinutes,omitempty"`
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
	BudgetSummary      BudgetSummary    `json:"budgetSummary"`
	Integration        IntegrationState `json:"integration"`
	Observations       []Observation    `json:"observations"`
	RecentRoutineRuns  []RoutineRun     `json:"recentRoutineRuns,omitempty"`
}

type RoutineSpec struct {
	SchemaVersion       int               `json:"schemaVersion"`
	ID                  string            `json:"id"`
	Title               string            `json:"title"`
	Purpose             string            `json:"purpose"`
	Trigger             string            `json:"trigger"`
	Inputs              []string          `json:"inputs"`
	AllowedActions      []string          `json:"allowedActions"`
	ForbiddenActions    []string          `json:"forbiddenActions"`
	Gates               []string          `json:"gates"`
	EvidenceLabels      []string          `json:"evidenceLabels"`
	OutputSchema        RoutineOutputSpec `json:"outputSchema"`
	Escalation          []string          `json:"escalation"`
	MaxRuntimeMinutes   int               `json:"maxRuntimeMinutes,omitempty"`
	ReviewBudgetMinutes int               `json:"reviewBudgetMinutes,omitempty"`
}

type RoutineOutputSpec struct {
	RequiredFields []string          `json:"requiredFields"`
	StatusValues   []string          `json:"statusValues"`
	Evidence       map[string]string `json:"evidence,omitempty"`
}

type RoutineValidationReport struct {
	Directory string                    `json:"directory"`
	CheckedAt string                    `json:"checkedAt"`
	Valid     bool                      `json:"valid"`
	Specs     []RoutineValidationResult `json:"specs"`
	Errors    []string                  `json:"errors,omitempty"`
}

type RoutineValidationResult struct {
	Path   string   `json:"path"`
	ID     string   `json:"id,omitempty"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

type RoutineRun struct {
	At                  string              `json:"at"`
	RoutineID           string              `json:"routineId"`
	TaskID              string              `json:"taskId,omitempty"`
	Status              string              `json:"status"`
	Evidence            map[string][]string `json:"evidence"`
	ActionsTaken        []string            `json:"actionsTaken"`
	NeedsHuman          bool                `json:"needsHuman"`
	BlockedReason       string              `json:"blockedReason,omitempty"`
	NextSuggestedAction string              `json:"nextSuggestedAction"`
}

type RoutineRunReport struct {
	RoutineID           string              `json:"routineId"`
	TaskID              string              `json:"taskId,omitempty"`
	Status              string              `json:"status"`
	Evidence            map[string][]string `json:"evidence"`
	ActionsTaken        []string            `json:"actionsTaken"`
	NeedsHuman          bool                `json:"needsHuman"`
	BlockedReason       string              `json:"blockedReason,omitempty"`
	NextSuggestedAction string              `json:"nextSuggestedAction"`
}

type githubReleaseView struct {
	TagName         string               `json:"tagName"`
	IsPrerelease    bool                 `json:"isPrerelease"`
	IsDraft         bool                 `json:"isDraft"`
	URL             string               `json:"url"`
	TargetCommitish string               `json:"targetCommitish"`
	Assets          []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name string `json:"name"`
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
	case "validate-routines":
		return cmdValidateRoutines(args[1:])
	case "run-routine":
		return cmdRunRoutine(args[1:])
	case "record-routine-run":
		return cmdRecordRoutineRun(args[1:])
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
  codex-orchestrator record-task --id ID --worktree PATH --branch BRANCH [--allowed PATH] [--forbidden PATH] [--gate CMD] [--max-runtime-minutes N] [--review-budget-minutes N]
  codex-orchestrator append-event --type TYPE [--task-id ID] [--status STATUS] [--note TEXT]
  codex-orchestrator observe [--ledger PATH] [--json] [--write-report PATH] [--write-summary PATH]
  codex-orchestrator heartbeat [--ledger PATH] [--interval 5m] [--count 0] [--write-report PATH]
  codex-orchestrator status [--ledger PATH] [--json]
  codex-orchestrator validate-routines [--dir routines] [--json]
  codex-orchestrator run-routine pr-reviewer --task-id TASK [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine stale-task-rescuer --task-id TASK [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine ci-fixer --task-id TASK [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine release-verifier --tag TAG [--repo PATH] [--expected-asset NAME] [--write-report PATH] [--json]
  codex-orchestrator run-routine docs-drift-checker [--repo PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine evidence-label-auditor [--repo PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine roadmap-next-task-suggester [--repo PATH] [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator record-routine-run --routine ID --status passed|failed|blocked [--task-id TASK]
  codex-orchestrator record-routine-run --report-json PATH

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
	maxRuntimeMinutes := fs.Int("max-runtime-minutes", 0, "optional task runtime budget in minutes")
	reviewBudgetMinutes := fs.Int("review-budget-minutes", 0, "optional task review budget in minutes")
	budgetNote := fs.String("budget-note", "", "optional budget note")
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
	if *maxRuntimeMinutes < 0 {
		return errors.New("record-task --max-runtime-minutes cannot be negative")
	}
	if *reviewBudgetMinutes < 0 {
		return errors.New("record-task --review-budget-minutes cannot be negative")
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
		Budget:     taskBudgetFromFlags(*maxRuntimeMinutes, *reviewBudgetMinutes, *budgetNote),
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
	event := map[string]any{
		"at":     nowISO(),
		"type":   "record-task",
		"taskId": *id,
		"status": *status,
	}
	if task.Budget != nil {
		event["budget"] = task.Budget
	}
	if err := appendEvent(resolvedEvents, event); err != nil {
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
		"ledger":            *ledgerPath,
		"projectRoot":       ledger.ProjectRoot,
		"defaultBranch":     ledger.DefaultBranch,
		"taskCount":         len(ledger.Tasks),
		"routineRunCount":   len(ledger.RoutineRuns),
		"counts":            counts,
		"recentRoutineRuns": recentRoutineRuns(ledger.RoutineRuns, 5),
	}
	if *jsonOut {
		return printJSON(result)
	}
	fmt.Printf("Ledger: %s\n", *ledgerPath)
	fmt.Printf("Project: %s default=%s\n", ledger.ProjectRoot, ledger.DefaultBranch)
	fmt.Printf("Tasks: %d\n", len(ledger.Tasks))
	fmt.Printf("Routine runs: %d\n", len(ledger.RoutineRuns))
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("- %s: %d\n", key, counts[key])
	}
	if recent := recentRoutineRuns(ledger.RoutineRuns, 5); len(recent) > 0 {
		fmt.Println("Recent routine runs:")
		for _, run := range recent {
			fmt.Printf("- %s %s", run.RoutineID, run.Status)
			if run.TaskID != "" {
				fmt.Printf(" task=%s", run.TaskID)
			}
			if run.NextSuggestedAction != "" {
				fmt.Printf(" next=%q", run.NextSuggestedAction)
			}
			fmt.Println()
		}
	}
	return nil
}

func cmdValidateRoutines(args []string) error {
	fs := flag.NewFlagSet("validate-routines", flag.ExitOnError)
	dir := fs.String("dir", "routines", "routine specs directory")
	jsonOut := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := validateRoutines(*dir)
	if *jsonOut {
		if err := printJSON(report); err != nil {
			return err
		}
	} else {
		printRoutineValidationReport(report)
	}
	if !report.Valid {
		return errors.New("routine validation failed")
	}
	return nil
}

func cmdRunRoutine(args []string) error {
	if len(args) == 0 {
		return errors.New("run-routine requires a routine id")
	}
	switch args[0] {
	case "pr-reviewer":
		return cmdRunPRReviewerRoutine(args[1:])
	case "stale-task-rescuer":
		return cmdRunStaleTaskRescuerRoutine(args[1:])
	case "ci-fixer":
		return cmdRunCIFixerRoutine(args[1:])
	case "release-verifier":
		return cmdRunReleaseVerifierRoutine(args[1:])
	case "docs-drift-checker":
		return cmdRunDocsDriftCheckerRoutine(args[1:])
	case "evidence-label-auditor":
		return cmdRunEvidenceLabelAuditorRoutine(args[1:])
	case "roadmap-next-task-suggester":
		return cmdRunRoadmapNextTaskSuggesterRoutine(args[1:])
	default:
		return fmt.Errorf("unsupported routine %q", args[0])
	}
}

func cmdRunPRReviewerRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine pr-reviewer", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	taskID := fs.String("task-id", "", "task id to inspect")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" {
		return errors.New("run-routine pr-reviewer requires --task-id")
	}
	report, err := runPRReviewerRoutine(*ledgerPath, *taskID)
	if err != nil {
		return err
	}
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut || *writeReport == "" {
		return printJSON(report)
	}
	fmt.Printf("Wrote routine report: %s\n", *writeReport)
	return nil
}

func runPRReviewerRoutine(ledgerPath string, taskID string) (RoutineRunReport, error) {
	report := RoutineRunReport{
		RoutineID: "pr-reviewer",
		TaskID:    taskID,
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Loaded ledger task record",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked routine precondition, then rerun pr-reviewer.",
	}
	ledger, err := loadLedger(ledgerPath)
	if err != nil {
		return report, err
	}
	taskIndex := findTaskIndex(ledger.Tasks, taskID)
	if taskIndex < 0 {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Task not found in ledger: "+taskID)
		report.BlockedReason = "task not found in ledger"
		return report, nil
	}
	task := ledger.Tasks[taskIndex]
	report.Evidence["local"] = append(report.Evidence["local"], "Task exists in ledger: "+task.ID)
	if task.Worktree == "" {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Ledger task has no worktree path.")
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree path is missing"
		return report, nil
	}
	worktree := expandPath(task.Worktree)
	if info, err := os.Stat(worktree); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Worktree does not exist: %s", worktree))
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree is missing"
		return report, nil
	} else if !info.IsDir() {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Worktree path is not a directory: %s", worktree))
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree path is not a directory"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Worktree exists: "+worktree)
	report.ActionsTaken = append(report.ActionsTaken, "Inspected task worktree git state")

	statusOut, err := gitOutput(worktree, "status", "--short", "--branch")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git status --short --branch failed: "+err.Error())
		report.BlockedReason = "could not inspect task worktree git status"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git status --short --branch:\n"+statusOut)

	branch := currentBranch(statusOut)
	if task.Branch != "" {
		if branch == "" {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], "Expected branch "+task.Branch+", but current branch could not be determined.")
			report.BlockedReason = "could not determine current branch"
			return report, nil
		}
		if branch != task.Branch {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Expected branch %s, found %s.", task.Branch, branch))
			report.BlockedReason = "task worktree branch does not match ledger branch"
			return report, nil
		}
		report.Evidence["local"] = append(report.Evidence["local"], "Branch matches ledger branch: "+branch)
	}

	if hasDirtyChanges(statusOut) {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "Worktree has uncommitted changes; routine is read-only and did not stage or modify them.")
		report.NextSuggestedAction = "Return to the same task worker for commit, cleanup, or explicit handoff of the dirty diff."
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Worktree is clean.")

	base := strings.TrimSpace(task.BaseCommit)
	if base == "" || allZeros(base) {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Task has no comparable baseCommit.")
		report.BlockedReason = "task baseCommit is missing"
		return report, nil
	}

	countOut, err := gitOutput(worktree, "rev-list", "--count", base+"..HEAD")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git rev-list --count "+base+"..HEAD failed: "+err.Error())
		report.BlockedReason = "could not compare task branch with baseCommit"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git rev-list --count "+base+"..HEAD: "+countOut)
	if strings.TrimSpace(countOut) == "0" {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "No commits are present after baseCommit.")
		report.NextSuggestedAction = "Do not merge; ask the task worker for a committed implementation or mark the task abandoned."
		return report, nil
	}

	nameStatusOut, err := gitOutput(worktree, "diff", "--name-status", base+"..HEAD")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git diff --name-status "+base+"..HEAD failed: "+err.Error())
		report.BlockedReason = "could not inspect task branch diff"
		return report, nil
	}
	if nameStatusOut == "" {
		nameStatusOut = "(no changed files)"
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git diff --name-status "+base+"..HEAD:\n"+nameStatusOut)
	report.ActionsTaken = append(report.ActionsTaken, "Checked committed file list against baseCommit")

	diffCheckOut, err := gitOutput(worktree, "diff", "--check", base+"..HEAD")
	if err != nil {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "git diff --check "+base+"..HEAD failed:\n"+err.Error())
		report.NextSuggestedAction = "Return to the same task worker to fix whitespace/conflict-marker issues, then rerun pr-reviewer."
		return report, nil
	}
	if diffCheckOut == "" {
		diffCheckOut = "passed with no output"
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git diff --check "+base+"..HEAD: "+diffCheckOut)
	report.ActionsTaken = append(report.ActionsTaken, "Ran read-only committed diff checks")

	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Review the local/static report and, if sufficient for this task, record it with record-routine-run --report-json before any separate merge decision."
	return report, nil
}

func cmdRunStaleTaskRescuerRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine stale-task-rescuer", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	taskID := fs.String("task-id", "", "task id to inspect")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" {
		return errors.New("run-routine stale-task-rescuer requires --task-id")
	}
	report, err := runStaleTaskRescuerRoutine(*ledgerPath, *taskID)
	if err != nil {
		return err
	}
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut || *writeReport == "" {
		return printJSON(report)
	}
	fmt.Printf("Wrote routine report: %s\n", *writeReport)
	return nil
}

func runStaleTaskRescuerRoutine(ledgerPath string, taskID string) (RoutineRunReport, error) {
	report := RoutineRunReport{
		RoutineID: "stale-task-rescuer",
		TaskID:    taskID,
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Loaded ledger task record",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked stale-task-rescuer precondition, then rerun the routine.",
	}
	ledger, err := loadLedger(ledgerPath)
	if err != nil {
		return report, err
	}
	taskIndex := findTaskIndex(ledger.Tasks, taskID)
	if taskIndex < 0 {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Task not found in ledger: "+taskID)
		report.BlockedReason = "task not found in ledger"
		return report, nil
	}
	task := ledger.Tasks[taskIndex]
	report.Evidence["local"] = append(report.Evidence["local"], "Task exists in ledger: "+task.ID)
	if task.Status != "" {
		report.Evidence["local"] = append(report.Evidence["local"], "Ledger task status: "+task.Status)
	} else {
		report.Evidence["local"] = append(report.Evidence["local"], "Ledger task status is empty.")
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Last observation: "+formatOptionalStringMap(task.LastObservation))
	report.Evidence["local"] = append(report.Evidence["local"], "Task history: "+formatTaskHistory(task.History, 3))
	report.ActionsTaken = append(report.ActionsTaken, "Inspected ledger status, last observation, and task history")

	if task.Worktree == "" {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Ledger task has no worktree path.")
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree path is missing"
		return report, nil
	}
	worktree := expandPath(task.Worktree)
	if info, err := os.Stat(worktree); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Worktree does not exist: %s", worktree))
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree is missing"
		return report, nil
	} else if !info.IsDir() {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Worktree path is not a directory: %s", worktree))
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree path is not a directory"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Worktree exists: "+worktree)
	report.ActionsTaken = append(report.ActionsTaken, "Inspected task worktree git state read-only")

	statusOut, err := gitOutput(worktree, "status", "--short", "--branch")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git status --short --branch failed: "+err.Error())
		report.BlockedReason = "could not inspect task worktree git status"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git status --short --branch:\n"+statusOut)

	branch := currentBranch(statusOut)
	if task.Branch != "" {
		if branch == "" {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], "Expected branch "+task.Branch+", but current branch could not be determined.")
			report.BlockedReason = "could not determine current branch"
			return report, nil
		}
		if branch != task.Branch {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Expected branch %s, found %s.", task.Branch, branch))
			report.BlockedReason = "task worktree branch does not match ledger branch"
			return report, nil
		}
		report.Evidence["local"] = append(report.Evidence["local"], "Branch matches ledger branch: "+branch)
	}

	logOut, err := gitOutput(worktree, "log", "--oneline", "-3")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git log --oneline -3 failed: "+err.Error())
		report.BlockedReason = "could not inspect task branch history"
		return report, nil
	}
	if logOut == "" {
		logOut = "(no commits)"
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git log --oneline -3:\n"+logOut)
	report.ActionsTaken = append(report.ActionsTaken, "Inspected recent task branch commits")

	base := strings.TrimSpace(task.BaseCommit)
	if base == "" || allZeros(base) {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Task has no comparable baseCommit.")
		report.BlockedReason = "task baseCommit is missing"
		return report, nil
	}

	if hasDirtyChanges(statusOut) {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "Worktree has uncommitted changes; routine is read-only and did not stage, commit, or modify them.")
		addOptionalGitEvidence(&report, worktree, "diff", "--name-status")
		addOptionalGitEvidence(&report, worktree, "diff", "--cached", "--name-status")
		addOptionalGitEvidence(&report, worktree, "ls-files", "--others", "--exclude-standard")
		report.ActionsTaken = append(report.ActionsTaken, "Captured local uncommitted change evidence without modifying the worktree")
		report.NextSuggestedAction = "Return to the same worker or perform a same-task takeover to preserve and finish the local diff; do not dispatch unrelated work over this task."
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Worktree is clean.")

	countOut, err := gitOutput(worktree, "rev-list", "--count", base+"..HEAD")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git rev-list --count "+base+"..HEAD failed: "+err.Error())
		report.BlockedReason = "could not compare task branch with baseCommit"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git rev-list --count "+base+"..HEAD: "+countOut)
	if strings.TrimSpace(countOut) == "0" {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "Clean worktree has no commits after baseCommit.")
		report.NextSuggestedAction = "Return to the same worker or same-task takeover to determine whether the task is still active, empty, or should be explicitly abandoned; do not dispatch unrelated work as a substitute."
		return report, nil
	}

	nameStatusOut, err := gitOutput(worktree, "diff", "--name-status", base+"..HEAD")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git diff --name-status "+base+"..HEAD failed: "+err.Error())
		report.BlockedReason = "could not inspect task branch diff"
		return report, nil
	}
	if nameStatusOut == "" {
		nameStatusOut = "(no changed files)"
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git diff --name-status "+base+"..HEAD:\n"+nameStatusOut)
	report.ActionsTaken = append(report.ActionsTaken, "Checked committed file list against baseCommit")

	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Run orchestrator review of the committed diff before any merge, cleanup, or ledger status change."
	return report, nil
}

func cmdRunCIFixerRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine ci-fixer", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	taskID := fs.String("task-id", "", "task id to inspect")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *taskID == "" {
		return errors.New("run-routine ci-fixer requires --task-id")
	}
	report, err := runCIFixerRoutine(*ledgerPath, *taskID)
	if err != nil {
		return err
	}
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut || *writeReport == "" {
		return printJSON(report)
	}
	fmt.Printf("Wrote routine report: %s\n", *writeReport)
	return nil
}

func runCIFixerRoutine(ledgerPath string, taskID string) (RoutineRunReport, error) {
	report := RoutineRunReport{
		RoutineID: "ci-fixer",
		TaskID:    taskID,
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Loaded ledger task record",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked ci-fixer precondition, then rerun the routine.",
	}
	ledger, err := loadLedger(ledgerPath)
	if err != nil {
		return report, err
	}
	taskIndex := findTaskIndex(ledger.Tasks, taskID)
	if taskIndex < 0 {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Task not found in ledger: "+taskID)
		report.BlockedReason = "task not found in ledger"
		return report, nil
	}
	task := ledger.Tasks[taskIndex]
	report.Evidence["local"] = append(report.Evidence["local"], "Task exists in ledger: "+task.ID)
	gates := nonEmptyStrings(task.Gates)
	if len(gates) > 0 {
		report.Evidence["local"] = append(report.Evidence["local"], "Recorded task gates: "+strings.Join(gates, " && "))
	} else {
		report.Status = "blocked"
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Task has no recorded gates for ci-fixer to run.")
		report.BlockedReason = "task gates are missing"
		report.NextSuggestedAction = "Record explicit local gate commands on the task before running ci-fixer."
		return report, nil
	}

	if task.Worktree == "" {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Ledger task has no worktree path.")
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree path is missing"
		return report, nil
	}
	worktree := expandPath(task.Worktree)
	if info, err := os.Stat(worktree); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Worktree does not exist: %s", worktree))
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree is missing"
		return report, nil
	} else if !info.IsDir() {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Worktree path is not a directory: %s", worktree))
		report.ActionsTaken = append(report.ActionsTaken, "Checked task worktree path")
		report.BlockedReason = "task worktree path is not a directory"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Worktree exists: "+worktree)
	report.ActionsTaken = append(report.ActionsTaken, "Inspected task worktree git state read-only")

	statusOut, err := gitOutput(worktree, "status", "--short", "--branch")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git status --short --branch failed: "+err.Error())
		report.BlockedReason = "could not inspect task worktree git status"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git status --short --branch:\n"+statusOut)

	branch := currentBranch(statusOut)
	if task.Branch != "" {
		if branch == "" {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], "Expected branch "+task.Branch+", but current branch could not be determined.")
			report.BlockedReason = "could not determine current branch"
			return report, nil
		}
		if branch != task.Branch {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Expected branch %s, found %s.", task.Branch, branch))
			report.BlockedReason = "task worktree branch does not match ledger branch"
			return report, nil
		}
		report.Evidence["local"] = append(report.Evidence["local"], "Branch matches ledger branch: "+branch)
	}

	if hasDirtyChanges(statusOut) {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "Worktree has uncommitted changes; ci-fixer is read-only and did not stage, commit, or modify them.")
		addOptionalGitEvidence(&report, worktree, "diff", "--name-status")
		addOptionalGitEvidence(&report, worktree, "diff", "--cached", "--name-status")
		addOptionalGitEvidence(&report, worktree, "ls-files", "--others", "--exclude-standard")
		report.ActionsTaken = append(report.ActionsTaken, "Captured local uncommitted change evidence without modifying the worktree")
		report.NextSuggestedAction = "Return to the same worker or perform a same-task takeover to commit or clean the task worktree before CI fix review."
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Worktree is clean.")

	base := strings.TrimSpace(task.BaseCommit)
	if base == "" || allZeros(base) {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Task has no comparable baseCommit.")
		report.BlockedReason = "task baseCommit is missing"
		return report, nil
	}

	countOut, err := gitOutput(worktree, "rev-list", "--count", base+"..HEAD")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git rev-list --count "+base+"..HEAD failed: "+err.Error())
		report.BlockedReason = "could not compare task branch with baseCommit"
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git rev-list --count "+base+"..HEAD: "+countOut)

	nameStatusOut, err := gitOutput(worktree, "diff", "--name-status", base+"..HEAD")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git diff --name-status "+base+"..HEAD failed: "+err.Error())
		report.BlockedReason = "could not inspect task branch diff"
		return report, nil
	}
	if nameStatusOut == "" {
		nameStatusOut = "(no changed files)"
	}
	report.Evidence["local"] = append(report.Evidence["local"], "git diff --name-status "+base+"..HEAD:\n"+nameStatusOut)
	report.ActionsTaken = append(report.ActionsTaken, "Checked committed file list against baseCommit")

	allGatesPassed := true
	for _, gate := range gates {
		output, err := shellOutput(worktree, 2*time.Minute, gate)
		label := "gate " + gate
		if strings.TrimSpace(output) == "" {
			output = "(no output)"
		}
		if err != nil {
			allGatesPassed = false
			report.Status = "failed"
			report.Evidence["local"] = append(report.Evidence["local"], label+" failed:\n"+output+"\nerror: "+err.Error())
			report.NextSuggestedAction = "Return to the same task worker to fix the recorded local gate failure, then rerun ci-fixer."
			continue
		}
		report.Evidence["local"] = append(report.Evidence["local"], label+" passed:\n"+output)
	}
	report.ActionsTaken = append(report.ActionsTaken, "Ran recorded task gates in the task worktree with a local timeout")
	if !allGatesPassed {
		return report, nil
	}

	afterStatus, err := gitOutput(worktree, "status", "--short", "--branch")
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "post-gate git status --short --branch failed: "+err.Error())
		report.BlockedReason = "could not inspect task worktree git status after gates"
		report.Status = "blocked"
		report.NextSuggestedAction = "Inspect the task worktree state before treating ci-fixer as complete."
		return report, nil
	}
	report.Evidence["local"] = append(report.Evidence["local"], "post-gate git status --short --branch:\n"+afterStatus)
	if hasDirtyChanges(afterStatus) {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "Recorded gates left the worktree dirty; ci-fixer did not stage, commit, or clean the generated changes.")
		report.NextSuggestedAction = "Return to the same task worker to commit or clean gate-generated changes before CI fix review."
		return report, nil
	}

	if strings.TrimSpace(countOut) == "0" {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "Gates passed, but no commits are present after baseCommit.")
		report.NextSuggestedAction = "Do not merge; ask the task worker for a committed implementation or mark the task abandoned."
		return report, nil
	}

	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Run orchestrator review/merge flow for the clean task branch; ci-fixer made no automatic code changes."
	return report, nil
}

func cmdRunReleaseVerifierRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine release-verifier", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository path to inspect")
	tag := fs.String("tag", "", "release tag to verify")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	var expectedAssets stringList
	fs.Var(&expectedAssets, "expected-asset", "expected release asset name, repeatable; defaults to this repo's Go CLI release assets")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *tag == "" {
		return errors.New("run-routine release-verifier requires --tag")
	}
	report := runReleaseVerifierRoutine(*repo, *tag, []string(expectedAssets))
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut || *writeReport == "" {
		return printJSON(report)
	}
	fmt.Printf("Wrote routine report: %s\n", *writeReport)
	return nil
}

func runReleaseVerifierRoutine(repo string, tag string, expectedAssets []string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "release-verifier",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Inspected local git tag state read-only",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked release-verifier precondition, then rerun the routine.",
	}
	repo = expandPath(repo)
	if repo == "" {
		repo = "."
	}
	if info, err := os.Stat(repo); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path does not exist: %s", repo))
		report.BlockedReason = "repository path is missing"
		return report
	} else if !info.IsDir() {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path is not a directory: %s", repo))
		report.BlockedReason = "repository path is not a directory"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Repository exists: "+repo)

	commit, err := gitOutput(repo, "rev-parse", "--verify", "refs/tags/"+tag+"^{}")
	if err != nil {
		if isGitRepositoryError(err.Error()) {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], "git rev-parse --verify refs/tags/"+tag+"^{} failed: "+err.Error())
			report.BlockedReason = "could not inspect local git repository"
			return report
		}
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], "git rev-parse --verify refs/tags/"+tag+"^{} failed: "+err.Error())
		report.NextSuggestedAction = "Create or fetch the expected tag, then rerun release-verifier before trusting the release."
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], fmt.Sprintf("Local git tag %s resolves to commit %s.", tag, strings.TrimSpace(commit)))

	tagType, err := gitOutput(repo, "cat-file", "-t", "refs/tags/"+tag)
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "git cat-file -t refs/tags/"+tag+" failed: "+err.Error())
		report.BlockedReason = "could not inspect local tag object type"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Local git tag object type: "+strings.TrimSpace(tagType))

	expected := nonEmptyStrings(expectedAssets)
	if len(expected) == 0 {
		expected = defaultReleaseVerifierAssets()
	}
	sort.Strings(expected)
	report.Evidence["local"] = append(report.Evidence["local"], "Expected release assets: "+strings.Join(expected, ", "))
	report.ActionsTaken = append(report.ActionsTaken, "Compared GitHub release metadata through gh when available")

	if _, err := exec.LookPath("gh"); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "gh executable not found on PATH; GitHub release and asset state were not inspected.")
		report.BlockedReason = "gh is unavailable"
		report.NextSuggestedAction = "Install/authenticate gh or provide another read-only GitHub release evidence source, then rerun release-verifier."
		return report
	}

	out, err := commandOutput(repo, "gh", "release", "view", tag, "--json", "tagName,isPrerelease,isDraft,url,targetCommitish,assets")
	if err != nil {
		if isNotFoundError(err.Error()) {
			report.Status = "failed"
			report.Evidence["proxy"] = append(report.Evidence["proxy"], "gh release view "+tag+" failed: "+err.Error())
			report.NextSuggestedAction = "Publish the GitHub release for the local tag, then rerun release-verifier."
			return report
		}
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "gh release view "+tag+" failed: "+err.Error())
		report.BlockedReason = "could not inspect GitHub release metadata"
		report.NextSuggestedAction = "Resolve gh authentication/network/repository access, then rerun release-verifier."
		return report
	}
	var release githubReleaseView
	if err := json.Unmarshal([]byte(out), &release); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not parse gh release view JSON: "+err.Error())
		report.BlockedReason = "could not parse GitHub release metadata"
		return report
	}
	report.Evidence["proxy"] = append(report.Evidence["proxy"], fmt.Sprintf("GitHub release exists for %s: url=%s targetCommitish=%s draft=%t prerelease=%t", release.TagName, release.URL, release.TargetCommitish, release.IsDraft, release.IsPrerelease))

	failed := false
	if release.TagName != "" && release.TagName != tag {
		failed = true
		report.Evidence["proxy"] = append(report.Evidence["proxy"], fmt.Sprintf("GitHub release tag mismatch: expected %s, got %s.", tag, release.TagName))
	}
	if release.IsDraft {
		failed = true
		report.Evidence["proxy"] = append(report.Evidence["proxy"], "GitHub release is still a draft.")
	}
	expectedPrerelease := isPrereleaseTag(tag)
	if release.IsPrerelease != expectedPrerelease {
		failed = true
		report.Evidence["proxy"] = append(report.Evidence["proxy"], fmt.Sprintf("GitHub prerelease mismatch for %s: expected %t, got %t.", tag, expectedPrerelease, release.IsPrerelease))
	}

	actualAssets := releaseAssetNames(release.Assets)
	report.Evidence["proxy"] = append(report.Evidence["proxy"], "GitHub release assets: "+formatStringList(actualAssets))
	missing := missingStrings(expected, actualAssets)
	if len(missing) > 0 {
		failed = true
		report.Evidence["proxy"] = append(report.Evidence["proxy"], "Missing expected release assets: "+strings.Join(missing, ", "))
	}

	if failed {
		report.Status = "failed"
		report.NextSuggestedAction = "Fix the GitHub release metadata/assets without mutating state from this routine, then rerun release-verifier."
		return report
	}
	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Record this routine report if the local/proxy release evidence is sufficient; perform any separate runtime/download smoke before claiming production proof."
	return report
}

func cmdRunDocsDriftCheckerRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine docs-drift-checker", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository path to inspect")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runDocsDriftCheckerRoutine(*repo)
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut || *writeReport == "" {
		return printJSON(report)
	}
	fmt.Printf("Wrote routine report: %s\n", *writeReport)
	return nil
}

func runDocsDriftCheckerRoutine(repo string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "docs-drift-checker",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Inspected runnable routine source and docs read-only",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked docs-drift-checker precondition, then rerun the routine.",
	}
	repo = expandPath(repo)
	if repo == "" {
		repo = "."
	}
	if info, err := os.Stat(repo); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path does not exist: %s", repo))
		report.BlockedReason = "repository path is missing"
		return report
	} else if !info.IsDir() {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path is not a directory: %s", repo))
		report.BlockedReason = "repository path is not a directory"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Repository exists: "+repo)

	sourcePath := filepath.Join(repo, "cmd", "codex-orchestrator", "main.go")
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not read runnable routine source "+sourcePath+": "+err.Error())
		report.BlockedReason = "could not inspect runnable routine source"
		return report
	}
	runnableIDs, err := extractRunnableRoutineIDs(string(sourceData))
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not extract run-routine command surface: "+err.Error())
		report.BlockedReason = "could not inspect run-routine command surface"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Runnable routines from cmd/codex-orchestrator/main.go: "+strings.Join(runnableIDs, ", "))

	specIDs, err := collectRoutineSpecIDs(filepath.Join(repo, "routines"))
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not inspect routines directory: "+err.Error())
		report.BlockedReason = "could not inspect routine specs"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Routine specs in routines/: "+strings.Join(specIDs, ", "))

	failures := []string{}
	for _, id := range runnableIDs {
		if !containsString(specIDs, id) {
			failures = append(failures, fmt.Sprintf("Runnable routine %s is missing routines/%s.json.", id, id))
		}
	}

	requiredDocs := []string{
		"README.md",
		"README.zh-CN.md",
		"SKILL.md",
		filepath.Join("docs", "routines", "README.md"),
	}
	for _, doc := range requiredDocs {
		text, readErr := os.ReadFile(filepath.Join(repo, doc))
		if readErr != nil {
			failures = append(failures, fmt.Sprintf("Required docs file %s could not be read: %v.", doc, readErr))
			continue
		}
		missing := missingStrings(runnableIDs, documentedRoutineIDs(string(text), runnableIDs))
		if len(missing) > 0 {
			failures = append(failures, fmt.Sprintf("%s is missing runnable routine reference(s): %s.", doc, strings.Join(missing, ", ")))
		} else {
			report.Evidence["local"] = append(report.Evidence["local"], doc+" mentions all runnable routines.")
		}
	}

	roadmapPath := filepath.Join(repo, "docs", "roadmap.md")
	if roadmapText, readErr := os.ReadFile(roadmapPath); readErr == nil {
		missing := missingStrings(runnableIDs, documentedRoutineIDs(string(roadmapText), runnableIDs))
		if len(missing) > 0 {
			failures = append(failures, fmt.Sprintf("docs/roadmap.md is missing runnable routine reference(s): %s.", strings.Join(missing, ", ")))
		} else {
			report.Evidence["local"] = append(report.Evidence["local"], "docs/roadmap.md mentions all runnable routines.")
		}
		for _, phrase := range staleRoadmapPhrases() {
			if strings.Contains(string(roadmapText), phrase) {
				failures = append(failures, fmt.Sprintf("docs/roadmap.md contains stale status text %q even though run-routine has runnable MVPs.", phrase))
			}
		}
	} else if errors.Is(readErr, os.ErrNotExist) {
		report.Evidence["local"] = append(report.Evidence["local"], "docs/roadmap.md is absent; optional roadmap drift check skipped.")
	} else {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not read optional docs/roadmap.md: "+readErr.Error())
		report.BlockedReason = "could not inspect optional roadmap"
		return report
	}

	report.ActionsTaken = append(report.ActionsTaken, "Compared runnable routines against JSON specs and key docs")
	if len(failures) > 0 {
		sort.Strings(failures)
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], failures...)
		report.NextSuggestedAction = "Update routine specs and key docs so the runnable command surface is documented, then rerun docs-drift-checker."
		return report
	}

	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Record this local/static report if the docs drift check is sufficient; no direct runtime proof was produced."
	return report
}

func cmdRunEvidenceLabelAuditorRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine evidence-label-auditor", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository path to inspect")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runEvidenceLabelAuditorRoutine(*repo)
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut || *writeReport == "" {
		return printJSON(report)
	}
	fmt.Printf("Wrote routine report: %s\n", *writeReport)
	return nil
}

func runEvidenceLabelAuditorRoutine(repo string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "evidence-label-auditor",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Inspected repo-local docs, routine specs, routine reports, and ledger-shaped JSON read-only",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked evidence-label-auditor precondition, then rerun the routine.",
	}
	repo = expandPath(repo)
	if repo == "" {
		repo = "."
	}
	if info, err := os.Stat(repo); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path does not exist: %s", repo))
		report.BlockedReason = "repository path is missing"
		return report
	} else if !info.IsDir() {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path is not a directory: %s", repo))
		report.BlockedReason = "repository path is not a directory"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Repository exists: "+repo)

	staticDirectReserved, specFindings, err := inspectEvidenceRoutineSpecs(repo)
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not inspect routines directory: "+err.Error())
		report.BlockedReason = "could not inspect routine specs"
		return report
	}
	reservedIDs := mapKeys(staticDirectReserved)
	if len(reservedIDs) == 0 {
		report.Evidence["local"] = append(report.Evidence["local"], "No routine specs explicitly reserve direct evidence.")
	} else {
		report.Evidence["local"] = append(report.Evidence["local"], "Routine specs with direct evidence explicitly reserved: "+strings.Join(reservedIDs, ", "))
	}

	paths, err := evidenceAuditPaths(repo)
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not collect evidence audit paths: "+err.Error())
		report.BlockedReason = "could not collect evidence audit inputs"
		return report
	}
	findings := append([]evidenceAuditFinding{}, specFindings...)
	for _, path := range paths {
		data, readErr := os.ReadFile(filepath.Join(repo, path))
		if readErr != nil {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Could not read %s: %v", path, readErr))
			report.BlockedReason = "could not read evidence audit input"
			return report
		}
		if strings.HasSuffix(path, ".json") {
			findings = append(findings, auditEvidenceJSON(path, data, staticDirectReserved)...)
		} else {
			findings = append(findings, auditEvidenceText(path, string(data))...)
		}
	}
	report.ActionsTaken = append(report.ActionsTaken, "Applied deterministic local/static evidence-label policy/eval rules ELA001-ELA009")
	report.Evidence["local"] = append(report.Evidence["local"], fmt.Sprintf("Scanned %d repo-local evidence-label input file(s).", len(paths)))
	report.Evidence["local"] = append(report.Evidence["local"], summarizeEvidenceAuditFindings(findings))

	if len(findings) > 0 {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], renderEvidenceAuditFindings(findings)...)
		report.NextSuggestedAction = "Review these local/static evidence-label suspicions, fix confirmed wording or report-shape issues, then rerun evidence-label-auditor."
		return report
	}

	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Record this local/static report if the conservative evidence-label audit is sufficient; no direct runtime proof was produced."
	return report
}

func cmdRunRoadmapNextTaskSuggesterRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine roadmap-next-task-suggester", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository path to inspect")
	ledgerPath := fs.String("ledger", "", "optional ledger path; defaults to REPO/.codex-orchestrator/ledger.json when present")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runRoadmapNextTaskSuggesterRoutine(*repo, *ledgerPath)
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut || *writeReport == "" {
		return printJSON(report)
	}
	fmt.Printf("Wrote routine report: %s\n", *writeReport)
	return nil
}

func runRoadmapNextTaskSuggesterRoutine(repo string, ledgerPath string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "roadmap-next-task-suggester",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Inspected roadmap, routine specs, runnable routines, and repo-local ledger state read-only",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked roadmap-next-task-suggester precondition, then rerun the routine.",
	}
	repo = expandPath(repo)
	if repo == "" {
		repo = "."
	}
	if info, err := os.Stat(repo); err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path does not exist: %s", repo))
		report.BlockedReason = "repository path is missing"
		return report
	} else if !info.IsDir() {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Repository path is not a directory: %s", repo))
		report.BlockedReason = "repository path is not a directory"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Repository exists: "+repo)

	roadmapPath := filepath.Join(repo, "docs", "roadmap.md")
	roadmapData, err := os.ReadFile(roadmapPath)
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not read docs/roadmap.md: "+err.Error())
		report.BlockedReason = "could not inspect roadmap"
		return report
	}
	candidates, err := parseRoadmapNextTaskCandidates(string(roadmapData))
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not derive roadmap task candidates from docs/roadmap.md: "+err.Error())
		report.BlockedReason = "roadmap candidate list is missing or ambiguous"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Roadmap candidate tasks: "+formatRoadmapCandidateNames(candidates))
	report.ActionsTaken = append(report.ActionsTaken, "Parsed v3 candidate routines and explicit remaining blocks from docs/roadmap.md")

	sourcePath := filepath.Join(repo, "cmd", "codex-orchestrator", "main.go")
	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not read runnable routine source "+sourcePath+": "+err.Error())
		report.BlockedReason = "could not inspect runnable routine source"
		return report
	}
	runnableIDs, err := extractRunnableRoutineIDs(string(sourceData))
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not extract run-routine command surface: "+err.Error())
		report.BlockedReason = "could not inspect run-routine command surface"
		return report
	}
	specIDs, err := collectRoutineSpecIDs(filepath.Join(repo, "routines"))
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not inspect routines directory: "+err.Error())
		report.BlockedReason = "could not inspect routine specs"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], "Runnable routines from cmd/codex-orchestrator/main.go: "+strings.Join(runnableIDs, ", "))
	report.Evidence["local"] = append(report.Evidence["local"], "Routine specs in routines/: "+strings.Join(specIDs, ", "))
	report.ActionsTaken = append(report.ActionsTaken, "Compared roadmap candidates against local runnable routines and routine specs")

	implemented := knownRoutineNameSet(append(append([]string{}, runnableIDs...), specIDs...))

	var ledger Ledger
	var observationsByTaskID map[string]string
	if resolvedLedger, ok := resolveOptionalLedgerPath(repo, ledgerPath); ok {
		loaded, loadErr := loadLedger(resolvedLedger)
		if loadErr != nil {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not inspect repo-local ledger "+resolvedLedger+": "+loadErr.Error())
			report.BlockedReason = "could not inspect ledger"
			return report
		}
		ledger = loaded
		report.Evidence["local"] = append(report.Evidence["local"], fmt.Sprintf("Ledger loaded from %s with %d task(s).", resolvedLedger, len(ledger.Tasks)))
		observationsByTaskID = observeTaskStatuses(ledger.Tasks)
		report.ActionsTaken = append(report.ActionsTaken, "Applied ledger task-state filters to roadmap candidates")
	} else {
		report.Evidence["local"] = append(report.Evidence["local"], "Repo-local ledger is absent; active/merged task filter skipped.")
	}

	suggestions := []string{}
	skipped := []string{}
	for _, candidate := range candidates {
		if implemented[normalizeRoadmapKey(candidate.Name)] {
			skipped = append(skipped, fmt.Sprintf("%s: already runnable or already has a routine spec.", candidate.Name))
			continue
		}
		if reason, ok := candidateBlockedByLedger(candidate, ledger.Tasks, observationsByTaskID); ok {
			skipped = append(skipped, fmt.Sprintf("%s: %s", candidate.Name, reason))
			continue
		}
		safe, safetyNote := classifyRoadmapCandidateSafety(candidate.Name)
		if !safe {
			skipped = append(skipped, fmt.Sprintf("%s: deferred because it is not a conservative read-only local task (%s).", candidate.Name, safetyNote))
			continue
		}
		reason := fmt.Sprintf("roadmap still lists %q under %s and it remains unimplemented in local runnable routines/specs", candidate.Name, candidate.Source)
		suggestions = append(suggestions, fmt.Sprintf("local suggestion: %s. reason: %s. safety: %s.", candidate.Name, reason, safetyNote))
	}
	if len(skipped) > 0 {
		report.Evidence["local"] = append(report.Evidence["local"], skipped...)
	}

	if len(suggestions) == 0 {
		report.Status = "passed"
		report.BlockedReason = ""
		report.Evidence["local"] = append(report.Evidence["local"], "No remaining safe read-only roadmap tasks were found after implemented-state and ledger filters.")
		report.NextSuggestedAction = "Safe read-only roadmap queue appears drained; either widen scope to a mutating/network task with explicit approval or update docs/roadmap.md with the next bounded local slice."
		return report
	}

	report.Status = "passed"
	report.BlockedReason = ""
	report.Evidence["local"] = append(report.Evidence["local"], suggestions...)
	report.NextSuggestedAction = primaryRoadmapNextAction(suggestions[0])
	return report
}

func cmdRecordRoutineRun(args []string) error {
	fs := flag.NewFlagSet("record-routine-run", flag.ExitOnError)
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	eventsPath := fs.String("events", "", "events path")
	reportJSON := fs.String("report-json", "", "routine report JSON path")
	routineID := fs.String("routine", "", "routine id")
	taskID := fs.String("task-id", "", "optional task id")
	status := fs.String("status", "", "routine status: passed, failed, or blocked")
	needsHuman := fs.Bool("needs-human", false, "whether human input is needed")
	blockedReason := fs.String("blocked-reason", "", "blocked reason")
	nextSuggestedAction := fs.String("next", "", "next suggested action")
	var directEvidence stringList
	var proxyEvidence stringList
	var localEvidence stringList
	var blockedEvidence stringList
	var actionsTaken stringList
	fs.Var(&directEvidence, "evidence-direct", "direct evidence item, repeatable")
	fs.Var(&proxyEvidence, "evidence-proxy", "proxy evidence item, repeatable")
	fs.Var(&localEvidence, "evidence-local", "local evidence item, repeatable")
	fs.Var(&blockedEvidence, "evidence-blocked", "blocked evidence item, repeatable")
	fs.Var(&actionsTaken, "action", "action taken, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := RoutineRunReport{}
	if *reportJSON != "" {
		loaded, err := loadRoutineRunReport(*reportJSON)
		if err != nil {
			return err
		}
		report = loaded
	} else {
		report = RoutineRunReport{
			RoutineID: *routineID,
			TaskID:    *taskID,
			Status:    *status,
			Evidence: map[string][]string{
				"direct":  []string(directEvidence),
				"proxy":   []string(proxyEvidence),
				"local":   []string(localEvidence),
				"blocked": []string(blockedEvidence),
			},
			ActionsTaken:        []string(actionsTaken),
			NeedsHuman:          *needsHuman,
			BlockedReason:       *blockedReason,
			NextSuggestedAction: *nextSuggestedAction,
		}
	}
	if err := validateRoutineRunReport(report); err != nil {
		return err
	}
	ledger, err := loadLedger(*ledgerPath)
	if err != nil {
		return err
	}
	if report.TaskID != "" && findTaskIndex(ledger.Tasks, report.TaskID) < 0 {
		return fmt.Errorf("task not found: %s", report.TaskID)
	}
	now := nowISO()
	run := RoutineRun{
		At:                  now,
		RoutineID:           report.RoutineID,
		TaskID:              report.TaskID,
		Status:              report.Status,
		Evidence:            normalizedEvidence(report.Evidence),
		ActionsTaken:        report.ActionsTaken,
		NeedsHuman:          report.NeedsHuman,
		BlockedReason:       report.BlockedReason,
		NextSuggestedAction: report.NextSuggestedAction,
	}
	ledger.RoutineRuns = append(ledger.RoutineRuns, run)
	if err := saveLedger(*ledgerPath, &ledger); err != nil {
		return err
	}
	resolvedEvents := *eventsPath
	if resolvedEvents == "" {
		resolvedEvents = eventsPathForLedger(*ledgerPath)
	}
	if err := appendEvent(resolvedEvents, map[string]any{
		"at":                  now,
		"type":                "routine-run",
		"routineId":           report.RoutineID,
		"taskId":              emptyToNil(report.TaskID),
		"status":              report.Status,
		"needsHuman":          report.NeedsHuman,
		"blockedReason":       emptyToNil(report.BlockedReason),
		"nextSuggestedAction": report.NextSuggestedAction,
	}); err != nil {
		return err
	}
	fmt.Printf("Recorded routine run: %s %s\n", report.RoutineID, report.Status)
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
	budget := summarizeTaskBudgets(ledger.Tasks)
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
		BudgetSummary:      budget,
		Integration:        integration,
		Observations:       observations,
		RecentRoutineRuns:  recentRoutineRuns(ledger.RoutineRuns, 5),
	}, nil
}

func taskBudgetFromFlags(maxRuntimeMinutes int, reviewBudgetMinutes int, note string) *BudgetMetadata {
	note = strings.TrimSpace(note)
	if maxRuntimeMinutes == 0 && reviewBudgetMinutes == 0 && note == "" {
		return nil
	}
	return &BudgetMetadata{
		MaxRuntimeMinutes:   maxRuntimeMinutes,
		ReviewBudgetMinutes: reviewBudgetMinutes,
		Note:                note,
	}
}

func summarizeTaskBudgets(tasks []Task) BudgetSummary {
	var summary BudgetSummary
	for _, task := range tasks {
		if task.Budget == nil {
			summary.TasksMissingBudget++
			continue
		}
		summary.TasksWithBudget++
		summary.TotalMaxRuntimeMinutes += task.Budget.MaxRuntimeMinutes
		summary.TotalReviewBudgetMinutes += task.Budget.ReviewBudgetMinutes
	}
	return summary
}

func taskObservation(task Task, status string, action string, note string, gitStatus string) Observation {
	return Observation{
		ID:        task.ID,
		Status:    status,
		Action:    action,
		Note:      note,
		GitStatus: gitStatus,
		Budget:    task.Budget,
	}
}

func inspectTask(task Task, staleAfter time.Duration) Observation {
	if isTerminalStatus(task.Status) {
		if task.Worktree != "" {
			worktree := expandPath(task.Worktree)
			if _, err := os.Stat(worktree); err == nil && task.Status != "rejected" {
				return taskObservation(task, "cleanup-needed", "remove accepted task worktree and delete local task branch if safe", fmt.Sprintf("Task is %s but worktree still exists: %s", task.Status, worktree), "")
			}
		}
		return taskObservation(task, task.Status, "quiet", fmt.Sprintf("Task is recorded as %s.", task.Status), "")
	}
	if task.Worktree == "" {
		return taskObservation(task, "blocked", "record missing worktree path", "Task has no worktree path in ledger.", "")
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
		return taskObservation(task, statusValue, action, note, "")
	}
	status, err := gitOutput(worktree, "status", "--short", "--branch")
	if err != nil {
		return taskObservation(task, "blocked", "inspect worktree git state", err.Error(), "")
	}
	branch := currentBranch(status)
	if task.Branch != "" && branch != "" && branch != task.Branch {
		return taskObservation(task, "blocked", "fix branch mismatch before review", fmt.Sprintf("Expected %s, found %s.", task.Branch, branch), status)
	}
	if hasDirtyChanges(status) {
		return taskObservation(task, "stale-needs-inspection", "inspect uncommitted scoped diff or nudge same worker", "Worktree has uncommitted changes.", status)
	}
	commitsAfterBase, known := hasCommitsAfterBase(worktree, task.BaseCommit)
	if known && commitsAfterBase {
		return taskObservation(task, "completed-unreviewed", "orchestrator review required before merge", "Clean worktree has commits after baseCommit.", status)
	}
	if !known {
		statusValue := task.Status
		if statusValue == "" {
			statusValue = "active"
		}
		if statusValue == "active" && isTaskStale(task, staleAfter) {
			return taskObservation(task, "stale-needs-inspection", "inspect manually", fmt.Sprintf("Task has no comparable baseCommit and the last observation is older than %s.", staleAfter), status)
		}
		return taskObservation(task, statusValue, "inspect manually", "Could not compare baseCommit; ledger may be a template or base is missing.", status)
	}
	if task.Status == "active" && isTaskStale(task, staleAfter) {
		return taskObservation(task, "stale-needs-inspection", "inspect recent thread messages or nudge same worker", fmt.Sprintf("Clean worktree has no commits after baseCommit, and last observation is older than %s.", staleAfter), status)
	}
	return taskObservation(task, "active", "quiet", "Clean worktree has no commits after baseCommit.", status)
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

func loadRoutineRunReport(path string) (RoutineRunReport, error) {
	var report RoutineRunReport
	data, err := os.ReadFile(expandPath(path))
	if err != nil {
		return report, err
	}
	if err := json.Unmarshal(data, &report); err != nil {
		return report, err
	}
	return report, nil
}

func validateRoutineRunReport(report RoutineRunReport) error {
	if strings.TrimSpace(report.RoutineID) == "" {
		return errors.New("routine report requires routineId")
	}
	if !containsString([]string{"passed", "failed", "blocked"}, report.Status) {
		return errors.New("routine report status must be passed, failed, or blocked")
	}
	evidence := normalizedEvidence(report.Evidence)
	if len(evidence["direct"])+len(evidence["proxy"])+len(evidence["local"])+len(evidence["blocked"]) == 0 {
		return errors.New("routine report requires at least one evidence item")
	}
	if len(report.ActionsTaken) == 0 {
		return errors.New("routine report requires at least one actionsTaken item")
	}
	for index, action := range report.ActionsTaken {
		if strings.TrimSpace(action) == "" {
			return fmt.Errorf("routine report actionsTaken[%d] must not be empty", index)
		}
	}
	if strings.TrimSpace(report.NextSuggestedAction) == "" {
		return errors.New("routine report requires nextSuggestedAction")
	}
	if report.Status == "blocked" && strings.TrimSpace(report.BlockedReason) == "" {
		return errors.New("blocked routine report requires blockedReason")
	}
	return nil
}

func normalizedEvidence(evidence map[string][]string) map[string][]string {
	normalized := map[string][]string{
		"direct":  {},
		"proxy":   {},
		"local":   {},
		"blocked": {},
	}
	for _, key := range []string{"direct", "proxy", "local", "blocked"} {
		for _, item := range evidence[key] {
			if strings.TrimSpace(item) != "" {
				normalized[key] = append(normalized[key], item)
			}
		}
	}
	return normalized
}

func nonEmptyStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			normalized = append(normalized, value)
		}
	}
	return normalized
}

func addOptionalGitEvidence(report *RoutineRunReport, worktree string, args ...string) {
	out, err := gitOutput(worktree, args...)
	label := "git " + strings.Join(args, " ")
	if err != nil {
		report.Evidence["local"] = append(report.Evidence["local"], label+" failed while collecting optional local evidence: "+err.Error())
		return
	}
	if strings.TrimSpace(out) == "" {
		out = "(no output)"
	}
	report.Evidence["local"] = append(report.Evidence["local"], label+":\n"+out)
}

func formatOptionalStringMap(values map[string]string) string {
	if len(values) == 0 {
		return "(none recorded)"
	}
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		if values[key] == "" {
			continue
		}
		parts = append(parts, key+"="+values[key])
	}
	if len(parts) == 0 {
		return "(none recorded)"
	}
	return strings.Join(parts, ", ")
}

func formatTaskHistory(history []map[string]string, limit int) string {
	if len(history) == 0 {
		return "(none recorded)"
	}
	if limit <= 0 || limit > len(history) {
		limit = len(history)
	}
	start := len(history) - limit
	items := make([]string, 0, limit)
	for index, event := range history[start:] {
		items = append(items, fmt.Sprintf("#%d{%s}", start+index+1, formatOptionalStringMap(event)))
	}
	return strings.Join(items, "; ")
}

func recentRoutineRuns(runs []RoutineRun, limit int) []RoutineRun {
	if limit <= 0 || len(runs) == 0 {
		return nil
	}
	start := len(runs) - limit
	if start < 0 {
		start = 0
	}
	recent := append([]RoutineRun(nil), runs[start:]...)
	for left, right := 0, len(recent)-1; left < right; left, right = left+1, right-1 {
		recent[left], recent[right] = recent[right], recent[left]
	}
	return recent
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
	case "merged", "released", "cleaned", "rejected", "abandoned":
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
	fmt.Fprintf(&b, "- tasksWithBudget: `%d`\n", summary.BudgetSummary.TasksWithBudget)
	if summary.BudgetSummary.TotalMaxRuntimeMinutes > 0 {
		fmt.Fprintf(&b, "- totalMaxRuntimeMinutes: `%d`\n", summary.BudgetSummary.TotalMaxRuntimeMinutes)
	}
	if summary.BudgetSummary.TotalReviewBudgetMinutes > 0 {
		fmt.Fprintf(&b, "- totalReviewBudgetMinutes: `%d`\n", summary.BudgetSummary.TotalReviewBudgetMinutes)
	}
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
	} else {
		for _, item := range summary.Observations {
			fmt.Fprintf(&b, "- `%s`: `%s` - %s\n", item.ID, item.Status, item.Action)
			if item.Note != "" {
				fmt.Fprintf(&b, "  - note: %s\n", item.Note)
			}
			if budget := formatBudget(item.Budget); budget != "" {
				fmt.Fprintf(&b, "  - budget: %s\n", budget)
			}
		}
	}
	if len(summary.RecentRoutineRuns) > 0 {
		fmt.Fprintf(&b, "\n## Recent Routine Runs\n\n")
		for _, run := range summary.RecentRoutineRuns {
			fmt.Fprintf(&b, "- `%s`: `%s`", run.RoutineID, run.Status)
			if run.TaskID != "" {
				fmt.Fprintf(&b, " task=`%s`", run.TaskID)
			}
			if run.NextSuggestedAction != "" {
				fmt.Fprintf(&b, " - next: %s", run.NextSuggestedAction)
			}
			fmt.Fprintf(&b, "\n")
			if run.BlockedReason != "" {
				fmt.Fprintf(&b, "  - blockedReason: %s\n", run.BlockedReason)
			}
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

func formatBudget(budget *BudgetMetadata) string {
	if budget == nil {
		return ""
	}
	parts := []string{}
	if budget.MaxRuntimeMinutes > 0 {
		parts = append(parts, fmt.Sprintf("maxRuntime=%dm", budget.MaxRuntimeMinutes))
	}
	if budget.ReviewBudgetMinutes > 0 {
		parts = append(parts, fmt.Sprintf("review=%dm", budget.ReviewBudgetMinutes))
	}
	if budget.Note != "" {
		parts = append(parts, "note="+budget.Note)
	}
	return strings.Join(parts, " ")
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
	fmt.Printf("Budget: tasksWithBudget=%d totalMaxRuntimeMinutes=%d totalReviewBudgetMinutes=%d\n",
		summary.BudgetSummary.TasksWithBudget,
		summary.BudgetSummary.TotalMaxRuntimeMinutes,
		summary.BudgetSummary.TotalReviewBudgetMinutes,
	)
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
		if budget := formatBudget(item.Budget); budget != "" {
			fmt.Printf("  budget: %s\n", budget)
		}
		if item.GitStatus != "" {
			fmt.Println("  git:")
			for _, line := range strings.Split(item.GitStatus, "\n") {
				fmt.Printf("    %s\n", line)
			}
		}
	}
	if len(summary.RecentRoutineRuns) > 0 {
		fmt.Println()
		fmt.Println("Recent routine runs:")
		for _, run := range summary.RecentRoutineRuns {
			fmt.Printf("- %s: %s", run.RoutineID, run.Status)
			if run.TaskID != "" {
				fmt.Printf(" task=%s", run.TaskID)
			}
			if run.NextSuggestedAction != "" {
				fmt.Printf(" next=%q", run.NextSuggestedAction)
			}
			fmt.Println()
			if run.BlockedReason != "" {
				fmt.Printf("  blockedReason: %s\n", run.BlockedReason)
			}
		}
	}
}

func validateRoutines(dir string) RoutineValidationReport {
	root := expandPath(dir)
	report := RoutineValidationReport{
		Directory: root,
		CheckedAt: nowISO(),
		Valid:     true,
		Specs:     []RoutineValidationResult{},
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		report.Valid = false
		report.Errors = append(report.Errors, err.Error())
		return report
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(root, entry.Name())
		result := validateRoutineFile(path)
		if !result.Valid {
			report.Valid = false
		}
		report.Specs = append(report.Specs, result)
	}
	sort.Slice(report.Specs, func(i, j int) bool {
		return report.Specs[i].Path < report.Specs[j].Path
	})
	if len(report.Specs) == 0 {
		report.Valid = false
		report.Errors = append(report.Errors, "no routine spec JSON files found")
	}
	return report
}

func validateRoutineFile(path string) RoutineValidationResult {
	result := RoutineValidationResult{Path: path, Valid: true}
	data, err := os.ReadFile(path)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	var spec RoutineSpec
	if err := json.Unmarshal(data, &spec); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, err.Error())
		return result
	}
	result.ID = spec.ID
	for _, issue := range validateRoutineSpec(spec) {
		result.Valid = false
		result.Errors = append(result.Errors, issue)
	}
	return result
}

func collectRoutineSpecIDs(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var spec RoutineSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if strings.TrimSpace(spec.ID) != "" {
			ids = append(ids, spec.ID)
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil, errors.New("no routine spec JSON files found")
	}
	return ids, nil
}

func extractRunnableRoutineIDs(source string) ([]string, error) {
	start := strings.Index(source, "func cmdRunRoutine(")
	if start < 0 {
		return nil, errors.New("cmdRunRoutine function not found")
	}
	block := source[start:]
	if next := strings.Index(block[len("func cmdRunRoutine("):], "\nfunc "); next >= 0 {
		block = block[:len("func cmdRunRoutine(")+next]
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, `case "`) {
			continue
		}
		rest := strings.TrimPrefix(trimmed, "case ")
		for {
			rest = strings.TrimSpace(rest)
			if !strings.HasPrefix(rest, `"`) {
				break
			}
			rest = strings.TrimPrefix(rest, `"`)
			end := strings.Index(rest, `"`)
			if end < 0 {
				break
			}
			id := rest[:end]
			if id != "" && !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
			rest = rest[end+1:]
			comma := strings.Index(rest, ",")
			colon := strings.Index(rest, ":")
			if comma < 0 || (colon >= 0 && colon < comma) {
				break
			}
			rest = rest[comma+1:]
		}
	}
	sort.Strings(ids)
	if len(ids) == 0 {
		return nil, errors.New("no run-routine cases found")
	}
	return ids, nil
}

func documentedRoutineIDs(text string, ids []string) []string {
	found := []string{}
	for _, id := range ids {
		if strings.Contains(text, id) {
			found = append(found, id)
		}
	}
	return found
}

type roadmapCandidate struct {
	Name   string
	Source string
	Line   int
	Kind   string
}

func parseRoadmapNextTaskCandidates(text string) ([]roadmapCandidate, error) {
	lines := strings.Split(text, "\n")
	candidates := []roadmapCandidate{}
	seen := map[string]bool{}
	currentSection := ""
	inV3 := false
	collectV3Candidates := false
	collectRemaining := false
	addCandidate := func(name string, line int, kind string) {
		name = cleanRoadmapCandidateText(name)
		key := normalizeRoadmapKey(name)
		if key == "" || seen[key] {
			return
		}
		seen[key] = true
		source := currentSection
		if source == "" {
			source = "roadmap"
		}
		candidates = append(candidates, roadmapCandidate{Name: name, Source: source, Line: line, Kind: kind})
	}
	for index, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(trimmed, "## "):
			currentSection = strings.TrimSpace(strings.TrimPrefix(trimmed, "## "))
			inV3 = strings.Contains(strings.ToLower(currentSection), "v3") && strings.Contains(strings.ToLower(currentSection), "routine")
			collectV3Candidates = false
			collectRemaining = false
			continue
		case strings.HasPrefix(trimmed, "### "):
			collectV3Candidates = false
			collectRemaining = false
			continue
		case inV3 && strings.HasPrefix(trimmed, "候选 routine"):
			collectV3Candidates = true
			collectRemaining = false
			continue
		case trimmed == "剩余：":
			collectRemaining = true
			collectV3Candidates = false
			continue
		}
		if collectV3Candidates {
			if strings.HasPrefix(trimmed, "- ") {
				addCandidate(strings.TrimPrefix(trimmed, "- "), index+1, "v3")
				continue
			}
			if trimmed == "" {
				continue
			}
			collectV3Candidates = false
		}
		if collectRemaining {
			switch {
			case strings.HasPrefix(trimmed, "- "):
				addCandidate(strings.TrimPrefix(trimmed, "- "), index+1, "remaining")
				continue
			case hasNumberedListPrefix(trimmed):
				addCandidate(trimmed[strings.Index(trimmed, ".")+1:], index+1, "remaining")
				continue
			case trimmed == "":
				continue
			default:
				collectRemaining = false
			}
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no v3 candidates or explicit remaining tasks found")
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		left := roadmapCandidatePriority(candidates[i])
		right := roadmapCandidatePriority(candidates[j])
		if left != right {
			return left < right
		}
		return candidates[i].Line < candidates[j].Line
	})
	return candidates, nil
}

func roadmapCandidatePriority(candidate roadmapCandidate) int {
	switch candidate.Kind {
	case "v3":
		return 0
	case "remaining":
		return 1
	default:
		return 2
	}
}

func cleanRoadmapCandidateText(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "- ")
	value = strings.TrimSuffix(value, "；")
	value = strings.TrimSuffix(value, "。")
	value = strings.TrimSuffix(value, ":")
	return strings.TrimSpace(value)
}

func hasNumberedListPrefix(value string) bool {
	if len(value) < 3 {
		return false
	}
	dot := strings.Index(value, ".")
	if dot <= 0 {
		return false
	}
	for _, r := range value[:dot] {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return dot+1 < len(value) && value[dot+1] == ' '
}

func normalizeRoadmapKey(value string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(value) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func formatRoadmapCandidateNames(candidates []roadmapCandidate) string {
	names := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		names = append(names, candidate.Name)
	}
	return strings.Join(names, ", ")
}

func knownRoutineNameSet(values []string) map[string]bool {
	known := map[string]bool{}
	for _, value := range values {
		key := normalizeRoadmapKey(value)
		if key != "" {
			known[key] = true
		}
	}
	return known
}

func resolveOptionalLedgerPath(repo string, explicit string) (string, bool) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, true
	}
	defaultPath := filepath.Join(repo, defaultLedger)
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, true
	}
	return "", false
}

func observeTaskStatuses(tasks []Task) map[string]string {
	statuses := map[string]string{}
	for _, task := range tasks {
		statuses[task.ID] = inspectTask(task, 15*time.Minute).Status
	}
	return statuses
}

func candidateBlockedByLedger(candidate roadmapCandidate, tasks []Task, observationsByTaskID map[string]string) (string, bool) {
	candidateKey := normalizeRoadmapKey(candidate.Name)
	if candidateKey == "" {
		return "", false
	}
	for _, task := range tasks {
		if !taskMatchesCandidate(task, candidateKey) {
			continue
		}
		taskStatus := strings.TrimSpace(task.Status)
		observedStatus := strings.TrimSpace(observationsByTaskID[task.ID])
		if isLedgerReservedStatus(taskStatus) || isObservationReservedStatus(observedStatus) {
			reason := fmt.Sprintf("already represented by ledger task %s (status=%s observed=%s)", task.ID, emptyToUnknown(taskStatus), emptyToUnknown(observedStatus))
			return reason, true
		}
	}
	return "", false
}

func taskMatchesCandidate(task Task, candidateKey string) bool {
	for _, value := range []string{task.ID, task.Title, task.Branch} {
		key := normalizeRoadmapKey(value)
		if key == "" {
			continue
		}
		if key == candidateKey || strings.Contains(key, candidateKey) || strings.Contains(candidateKey, key) {
			return true
		}
	}
	return false
}

func isLedgerReservedStatus(status string) bool {
	switch status {
	case "active", "pending-setup", "completed-unreviewed", "merged":
		return true
	default:
		return false
	}
}

func isObservationReservedStatus(status string) bool {
	switch status {
	case "active", "pending-setup", "completed-unreviewed":
		return true
	default:
		return false
	}
}

func classifyRoadmapCandidateSafety(name string) (bool, string) {
	key := normalizeRoadmapKey(name)
	switch {
	case containsAnyFold(name, []string{"rebase", "merge", "push", "cleanup", "delete", "session", "worker pool", "daemon", "release", "tag", "deploy", "hardware", "payment", "prod", "notification"}):
		return false, "it implies mutating git/release/session/runtime work instead of a local read-only planning or audit slice"
	case strings.Contains(key, "reviewer"),
		strings.Contains(key, "checker"),
		strings.Contains(key, "auditor"),
		strings.Contains(key, "suggester"),
		strings.Contains(key, "budget"),
		strings.Contains(key, "policy"),
		strings.Contains(key, "eval"),
		strings.Contains(key, "ledger"),
		strings.Contains(key, "heartbeat"):
		return true, "it can stay bounded to repo-local docs/spec/ledger analysis without merge, push, release, or session mutation"
	default:
		return false, "it is not clearly a bounded read-only local task"
	}
}

func primaryRoadmapNextAction(suggestion string) string {
	const prefix = "local suggestion: "
	if strings.HasPrefix(suggestion, prefix) {
		suggestion = strings.TrimPrefix(suggestion, prefix)
	}
	if dot := strings.Index(suggestion, "."); dot >= 0 {
		suggestion = suggestion[:dot]
	}
	return suggestion
}

type evidenceAuditFinding struct {
	RuleID  string
	Message string
}

const (
	evidenceRuleSpecDirectWeakLocal  = "ELA001"
	evidenceRuleSpecLocalOverclaim   = "ELA002"
	evidenceRuleSpecBlockedOverclaim = "ELA003"
	evidenceRuleTextOverclaim        = "ELA004"
	evidenceRuleJSONParse            = "ELA005"
	evidenceRuleReportMissingObject  = "ELA006"
	evidenceRuleReportBadObject      = "ELA007"
	evidenceRuleReportMissingBucket  = "ELA008"
	evidenceRuleReportReservedDirect = "ELA009"
)

func newEvidenceAuditFinding(ruleID string, format string, args ...any) evidenceAuditFinding {
	return evidenceAuditFinding{
		RuleID:  ruleID,
		Message: fmt.Sprintf(format, args...),
	}
}

func renderEvidenceAuditFindings(findings []evidenceAuditFinding) []string {
	sorted := append([]evidenceAuditFinding{}, findings...)
	sort.Slice(sorted, func(i int, j int) bool {
		if sorted[i].RuleID == sorted[j].RuleID {
			return sorted[i].Message < sorted[j].Message
		}
		return sorted[i].RuleID < sorted[j].RuleID
	})
	rendered := make([]string, 0, len(sorted))
	for _, finding := range sorted {
		rendered = append(rendered, fmt.Sprintf("[%s] %s", finding.RuleID, finding.Message))
	}
	return rendered
}

func summarizeEvidenceAuditFindings(findings []evidenceAuditFinding) string {
	if len(findings) == 0 {
		return "Rule hits: none."
	}
	counts := map[string]int{}
	for _, finding := range findings {
		counts[finding.RuleID]++
	}
	ruleIDs := make([]string, 0, len(counts))
	for ruleID := range counts {
		ruleIDs = append(ruleIDs, ruleID)
	}
	sort.Strings(ruleIDs)
	parts := make([]string, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		parts = append(parts, fmt.Sprintf("%s=%d", ruleID, counts[ruleID]))
	}
	return "Rule hits: " + strings.Join(parts, ", ") + "."
}

func inspectEvidenceRoutineSpecs(repo string) (map[string]bool, []evidenceAuditFinding, error) {
	dir := filepath.Join(repo, "routines")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	staticDirectReserved := map[string]bool{}
	findings := []evidenceAuditFinding{}
	checked := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		checked++
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, nil, err
		}
		var spec RoutineSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			return nil, nil, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		if spec.ID == "" {
			continue
		}
		direct := spec.OutputSchema.Evidence["direct"]
		if evidenceDescriptionReservesDirect(direct) {
			staticDirectReserved[spec.ID] = true
		}
		findings = append(findings, auditRoutineSpecEvidenceDescriptions(filepath.Join("routines", entry.Name()), spec)...)
	}
	if checked == 0 {
		return nil, nil, errors.New("no routine spec JSON files found")
	}
	return staticDirectReserved, findings, nil
}

func auditRoutineSpecEvidenceDescriptions(path string, spec RoutineSpec) []evidenceAuditFinding {
	findings := []evidenceAuditFinding{}
	evidence := spec.OutputSchema.Evidence
	direct := evidence["direct"]
	if containsAnyFold(direct, weakEvidenceTerms()) && !evidenceDescriptionReservesDirect(direct) {
		findings = append(findings, newEvidenceAuditFinding(
			evidenceRuleSpecDirectWeakLocal,
			"%s: local/static suspicion: direct evidence description for %s contains local/static wording: %q",
			path,
			spec.ID,
			direct,
		))
	}
	local := evidence["local"]
	if containsAnyFold(local, strongEvidenceClaimTerms()) && !containsAnyFold(local, evidenceNegationTerms()) {
		findings = append(findings, newEvidenceAuditFinding(
			evidenceRuleSpecLocalOverclaim,
			"%s: local/static suspicion: local evidence description for %s contains strong proof wording: %q",
			path,
			spec.ID,
			local,
		))
	}
	blocked := evidence["blocked"]
	if containsAnyFold(blocked, []string{"verified", "passed", "proven"}) && !containsAnyFold(blocked, evidenceNegationTerms()) {
		findings = append(findings, newEvidenceAuditFinding(
			evidenceRuleSpecBlockedOverclaim,
			"%s: local/static suspicion: blocked evidence description for %s may describe proof as blocked: %q",
			path,
			spec.ID,
			blocked,
		))
	}
	return findings
}

func evidenceDescriptionReservesDirect(text string) bool {
	return containsAnyFold(text, []string{
		"reserved",
		"does not emit direct",
		"does not claim direct",
		"future direct",
		"no direct",
	})
}

func evidenceAuditPaths(repo string) ([]string, error) {
	paths := []string{}
	addIfExists := func(path string) error {
		full := filepath.Join(repo, path)
		info, err := os.Stat(full)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			return err
		}
		if info.IsDir() {
			return nil
		}
		paths = append(paths, path)
		return nil
	}
	for _, path := range []string{
		"README.md",
		"README.zh-CN.md",
		"SKILL.md",
		filepath.Join("docs", "roadmap.md"),
		filepath.Join("examples", "ledger.example.json"),
		filepath.Join(".codex-orchestrator", "ledger.json"),
	} {
		if err := addIfExists(path); err != nil {
			return nil, err
		}
	}
	for _, dir := range []string{
		filepath.Join("docs", "routines"),
		"routines",
		filepath.Join("examples", "routine-reports"),
		filepath.Join(".codex-orchestrator"),
	} {
		entries, err := os.ReadDir(filepath.Join(repo, dir))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			switch {
			case strings.HasSuffix(entry.Name(), ".md"):
				paths = append(paths, path)
			case strings.HasSuffix(entry.Name(), ".json") && dir != filepath.Join(".codex-orchestrator"):
				paths = append(paths, path)
			case strings.HasSuffix(entry.Name(), ".json") && strings.Contains(entry.Name(), "routine"):
				paths = append(paths, path)
			}
		}
	}
	paths = uniqueSortedStrings(paths)
	if len(paths) == 0 {
		return nil, errors.New("no docs, routine specs, routine reports, or ledger-like JSON files found")
	}
	return paths, nil
}

func auditEvidenceText(path string, text string) []evidenceAuditFinding {
	findings := []evidenceAuditFinding{}
	for index, line := range strings.Split(text, "\n") {
		if shouldSkipEvidenceTextLine(line) {
			continue
		}
		if !containsAnyFold(line, weakEvidenceTerms()) || !containsAnyFold(line, strongEvidenceClaimTerms()) {
			continue
		}
		if !containsAnyFold(line, evidenceAssertionTerms()) {
			continue
		}
		findings = append(findings, newEvidenceAuditFinding(
			evidenceRuleTextOverclaim,
			"%s:%d: local/static suspicion: weak evidence wording appears near strong proof wording: %s",
			path,
			index+1,
			strings.TrimSpace(line),
		))
	}
	return findings
}

func shouldSkipEvidenceTextLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return true
	}
	if containsAnyFold(trimmed, evidenceNegationTerms()) {
		return true
	}
	if isEvidenceGlossaryLine(trimmed) {
		return true
	}
	if isBlockedEvidenceExplanationLine(trimmed) {
		return true
	}
	return false
}

func isEvidenceGlossaryLine(line string) bool {
	return mentionsMultipleEvidenceLabels(line) && containsAnyFold(line, []string{
		"evidence label",
		"evidence labels",
		"evidence bucket",
		"evidence buckets",
		"labels include",
		"buckets include",
		"证据标签",
		"证据桶",
	})
}

func isBlockedEvidenceExplanationLine(line string) bool {
	return containsAnyFold(line, []string{
		"blocked means",
		"blocked evidence label",
		"blocked evidence bucket",
		"blocked definition",
		"blocked label",
		"blocked bucket",
		"could not be proven safely",
		"claim could not be proven safely",
		"blocked 表示",
		"阻断定义",
		"阻断表示",
		"无法安全证明",
	})
}

func mentionsMultipleEvidenceLabels(line string) bool {
	count := 0
	for _, term := range []string{"direct", "proxy", "local", "blocked", "直接", "代理", "本地", "阻断"} {
		if containsAnyFold(line, []string{term}) {
			count++
		}
	}
	return count >= 2
}

func auditEvidenceJSON(path string, data []byte, staticDirectReserved map[string]bool) []evidenceAuditFinding {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return []evidenceAuditFinding{
			newEvidenceAuditFinding(
				evidenceRuleJSONParse,
				"%s: local/static suspicion: JSON could not be parsed for evidence-label audit: %v",
				path,
				err,
			),
		}
	}
	findings := []evidenceAuditFinding{}
	if looksLikeRoutineRunReport(raw) {
		findings = append(findings, auditRoutineRunEvidenceObject(path, raw, staticDirectReserved)...)
	}
	if runsRaw, ok := raw["routineRuns"]; ok {
		var runs []map[string]json.RawMessage
		if err := json.Unmarshal(runsRaw, &runs); err != nil {
			findings = append(findings, newEvidenceAuditFinding(
				evidenceRuleJSONParse,
				"%s: local/static suspicion: routineRuns could not be parsed for evidence-label audit: %v",
				path,
				err,
			))
		} else {
			for index, run := range runs {
				findings = append(findings, auditRoutineRunEvidenceObject(fmt.Sprintf("%s routineRuns[%d]", path, index), run, staticDirectReserved)...)
			}
		}
	}
	return findings
}

func looksLikeRoutineRunReport(raw map[string]json.RawMessage) bool {
	_, hasRoutineID := raw["routineId"]
	_, hasStatus := raw["status"]
	return hasRoutineID && hasStatus
}

func auditRoutineRunEvidenceObject(path string, raw map[string]json.RawMessage, staticDirectReserved map[string]bool) []evidenceAuditFinding {
	findings := []evidenceAuditFinding{}
	routineID := rawString(raw["routineId"])
	evidenceRaw, ok := raw["evidence"]
	if !ok {
		return []evidenceAuditFinding{
			newEvidenceAuditFinding(
				evidenceRuleReportMissingObject,
				"%s: local/static suspicion: RoutineRunReport for %s is missing evidence object.",
				path,
				emptyToUnknown(routineID),
			),
		}
	}
	var evidence map[string]json.RawMessage
	if err := json.Unmarshal(evidenceRaw, &evidence); err != nil {
		return []evidenceAuditFinding{
			newEvidenceAuditFinding(
				evidenceRuleReportBadObject,
				"%s: local/static suspicion: RoutineRunReport evidence for %s could not be parsed: %v",
				path,
				emptyToUnknown(routineID),
				err,
			),
		}
	}
	for _, bucket := range []string{"direct", "proxy", "local", "blocked"} {
		if _, ok := evidence[bucket]; !ok {
			findings = append(findings, newEvidenceAuditFinding(
				evidenceRuleReportMissingBucket,
				"%s: local/static suspicion: RoutineRunReport for %s is missing evidence bucket %q.",
				path,
				emptyToUnknown(routineID),
				bucket,
			))
		}
	}
	if staticDirectReserved[routineID] {
		directItems := rawStringSlice(evidence["direct"])
		if len(directItems) > 0 {
			findings = append(findings, newEvidenceAuditFinding(
				evidenceRuleReportReservedDirect,
				"%s: local/static suspicion: RoutineRunReport for static-only routine %s contains direct evidence even though the routine spec reserves direct evidence.",
				path,
				routineID,
			))
		}
	}
	return findings
}

func weakEvidenceTerms() []string {
	return []string{
		"local",
		"proxy",
		"blocked",
		"unit",
		"static",
		"fixture",
		"mock",
		"synthetic",
		"build",
		"compile",
		"scaffold",
		"本地",
		"代理",
		"阻断",
		"单元测试",
		"静态",
		"桩",
	}
}

func strongEvidenceClaimTerms() []string {
	return []string{
		"direct proof",
		"direct evidence",
		"runtime proof",
		"production proof",
		"prod proof",
		"pre proof",
		"device proof",
		"real-device proof",
		"production runtime",
		"live runtime",
		"runtime verified",
		"prod verified",
		"直接证明",
		"直接证据",
		"运行时证明",
		"生产证明",
		"真实设备证明",
	}
}

func evidenceNegationTerms() []string {
	return []string{
		"do not",
		"does not",
		"don't",
		"not claim",
		"not direct",
		"not production",
		"not runtime",
		"no direct",
		"no runtime",
		"never",
		"without",
		"reserved",
		"cannot",
		"could not",
		"不",
		"不会",
		"不要",
		"不得",
		"不许",
		"不能",
		"没有",
	}
}

func evidenceAssertionTerms() []string {
	return []string{
		"provide",
		"provides",
		"provided",
		"prove",
		"proves",
		"proved",
		"verified",
		"complete",
		"passed",
		"backs",
		"backed by",
		"counts as",
		"demonstrates",
		"confirms",
		" is direct",
		" are direct",
		"as direct",
		"提供",
		"证明了",
		"验证了",
		"通过",
		"算作",
	}
}

func containsAnyFold(text string, terms []string) bool {
	lower := strings.ToLower(text)
	for _, term := range terms {
		if strings.Contains(lower, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func rawString(raw json.RawMessage) string {
	var value string
	if len(raw) == 0 || json.Unmarshal(raw, &value) != nil {
		return ""
	}
	return value
}

func rawStringSlice(raw json.RawMessage) []string {
	var values []string
	if len(raw) == 0 || json.Unmarshal(raw, &values) != nil {
		return nil
	}
	return nonEmptyStrings(values)
}

func emptyToUnknown(value string) string {
	if strings.TrimSpace(value) == "" {
		return "(unknown)"
	}
	return value
}

func mapKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func staleRoadmapPhrases() []string {
	return []string{
		"还没有自动执行 routine",
		"还没有自动运行 routine",
		"no runnable routine",
		"no automatic routine execution",
	}
}

func validateRoutineSpec(spec RoutineSpec) []string {
	var issues []string
	if spec.SchemaVersion != 1 {
		issues = append(issues, "schemaVersion must be 1")
	}
	requiredStrings := map[string]string{
		"id":      spec.ID,
		"title":   spec.Title,
		"purpose": spec.Purpose,
		"trigger": spec.Trigger,
	}
	for key, value := range requiredStrings {
		if strings.TrimSpace(value) == "" {
			issues = append(issues, key+" is required")
		}
	}
	requiredLists := map[string][]string{
		"inputs":           spec.Inputs,
		"allowedActions":   spec.AllowedActions,
		"forbiddenActions": spec.ForbiddenActions,
		"gates":            spec.Gates,
		"evidenceLabels":   spec.EvidenceLabels,
		"escalation":       spec.Escalation,
		"requiredFields":   spec.OutputSchema.RequiredFields,
		"statusValues":     spec.OutputSchema.StatusValues,
	}
	for key, values := range requiredLists {
		if len(values) == 0 {
			issues = append(issues, key+" must not be empty")
		}
		for index, value := range values {
			if strings.TrimSpace(value) == "" {
				issues = append(issues, fmt.Sprintf("%s[%d] must not be empty", key, index))
			}
		}
	}
	for _, required := range []string{"direct", "proxy", "local", "blocked"} {
		if !containsString(spec.EvidenceLabels, required) {
			issues = append(issues, "evidenceLabels must include "+required)
		}
	}
	for _, required := range []string{"status", "evidence", "actionsTaken", "needsHuman", "blockedReason", "nextSuggestedAction"} {
		if !containsString(spec.OutputSchema.RequiredFields, required) {
			issues = append(issues, "outputSchema.requiredFields must include "+required)
		}
	}
	for _, required := range []string{"passed", "failed", "blocked"} {
		if !containsString(spec.OutputSchema.StatusValues, required) {
			issues = append(issues, "outputSchema.statusValues must include "+required)
		}
	}
	if spec.MaxRuntimeMinutes < 0 {
		issues = append(issues, "maxRuntimeMinutes cannot be negative")
	}
	if spec.ReviewBudgetMinutes < 0 {
		issues = append(issues, "reviewBudgetMinutes cannot be negative")
	}
	return issues
}

func printRoutineValidationReport(report RoutineValidationReport) {
	fmt.Printf("Routine directory: %s\n", report.Directory)
	fmt.Printf("Valid: %t\n", report.Valid)
	for _, issue := range report.Errors {
		fmt.Printf("Error: %s\n", issue)
	}
	for _, spec := range report.Specs {
		fmt.Println()
		fmt.Printf("- %s", spec.Path)
		if spec.ID != "" {
			fmt.Printf(" (%s)", spec.ID)
		}
		fmt.Printf(": valid=%t\n", spec.Valid)
		for _, issue := range spec.Errors {
			fmt.Printf("  - %s\n", issue)
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

func shellOutput(cwd string, timeout time.Duration, command string) (string, error) {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell, "-lc", command)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if ctx.Err() == context.DeadlineExceeded {
		return output, fmt.Errorf("command timed out after %s", timeout)
	}
	if err != nil {
		return output, err
	}
	return output, nil
}

func commandOutput(cwd string, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))
	if err != nil {
		if output == "" {
			return "", err
		}
		return "", fmt.Errorf("%s", output)
	}
	return output, nil
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

func defaultReleaseVerifierAssets() []string {
	return []string{
		"codex-orchestrator_darwin_amd64",
		"codex-orchestrator_darwin_amd64.tar.gz",
		"codex-orchestrator_darwin_arm64",
		"codex-orchestrator_darwin_arm64.tar.gz",
		"codex-orchestrator_linux_amd64",
		"codex-orchestrator_linux_amd64.tar.gz",
		"codex-orchestrator_linux_arm64",
		"codex-orchestrator_linux_arm64.tar.gz",
		"codex-orchestrator_windows_amd64.exe",
		"codex-orchestrator_windows_amd64.exe.zip",
	}
}

func isPrereleaseTag(tag string) bool {
	return strings.Contains(tag, "-alpha") || strings.Contains(tag, "-beta") || strings.Contains(tag, "-rc")
}

func releaseAssetNames(assets []githubReleaseAsset) []string {
	names := make([]string, 0, len(assets))
	for _, asset := range assets {
		if strings.TrimSpace(asset.Name) != "" {
			names = append(names, asset.Name)
		}
	}
	sort.Strings(names)
	return names
}

func missingStrings(expected []string, actual []string) []string {
	actualSet := map[string]bool{}
	for _, value := range actual {
		actualSet[value] = true
	}
	var missing []string
	for _, value := range expected {
		if !actualSet[value] {
			missing = append(missing, value)
		}
	}
	return missing
}

func formatStringList(values []string) string {
	if len(values) == 0 {
		return "(none)"
	}
	return strings.Join(values, ", ")
}

func isNotFoundError(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "not found") || strings.Contains(lower, "release not found") || strings.Contains(lower, "could not resolve to a release")
}

func isGitRepositoryError(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "not a git repository") || strings.Contains(lower, "not a git repo")
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
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
