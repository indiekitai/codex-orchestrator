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
	"runtime"
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
	ID                string              `json:"id"`
	Title             string              `json:"title,omitempty"`
	ThreadID          string              `json:"threadId,omitempty"`
	PendingWorktreeID string              `json:"pendingWorktreeId,omitempty"`
	Worktree          string              `json:"worktree,omitempty"`
	Branch            string              `json:"branch,omitempty"`
	BaseCommit        string              `json:"baseCommit,omitempty"`
	Status            string              `json:"status"`
	Budget            *BudgetMetadata     `json:"budget,omitempty"`
	WriteSet          map[string][]string `json:"writeSet,omitempty"`
	Gates             []string            `json:"gates,omitempty"`
	Evidence          map[string]any      `json:"evidence,omitempty"`
	LastObservation   map[string]string   `json:"lastObservation,omitempty"`
	History           []map[string]string `json:"history,omitempty"`
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
	ID                string          `json:"id,omitempty"`
	Status            string          `json:"status"`
	Action            string          `json:"action"`
	Note              string          `json:"note"`
	Signal            string          `json:"signal,omitempty"`
	State             LocalTaskState  `json:"state"`
	LedgerStatus      string          `json:"ledgerStatus,omitempty"`
	Branch            string          `json:"branch,omitempty"`
	Worktree          string          `json:"worktree,omitempty"`
	LastUpdatedAt     string          `json:"lastUpdatedAt,omitempty"`
	GitStatus         string          `json:"gitStatus,omitempty"`
	PendingWorktreeID string          `json:"pendingWorktreeId,omitempty"`
	Budget            *BudgetMetadata `json:"budget,omitempty"`
	BudgetPressure    *BudgetPressure `json:"budgetPressure,omitempty"`
}

type LocalTaskState struct {
	EvidenceLabel string `json:"evidenceLabel"`
	Lifecycle     string `json:"lifecycle"`
	Setup         string `json:"setup"`
	Worktree      string `json:"worktree"`
	Branch        string `json:"branch"`
	Diff          string `json:"diff"`
	Review        string `json:"review"`
	Cleanup       string `json:"cleanup"`
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
	TasksWithBudget           int `json:"tasksWithBudget"`
	TasksMissingBudget        int `json:"tasksMissingBudget"`
	TotalMaxRuntimeMinutes    int `json:"totalMaxRuntimeMinutes,omitempty"`
	TotalReviewBudgetMinutes  int `json:"totalReviewBudgetMinutes,omitempty"`
	RoutineSpecsWithBudget    int `json:"routineSpecsWithBudget,omitempty"`
	RoutineSpecsMissingBudget int `json:"routineSpecsMissingBudget,omitempty"`
}

type BudgetPressure struct {
	Status                 string   `json:"status"`
	EvidenceLabel          string   `json:"evidenceLabel"`
	Warnings               []string `json:"warnings,omitempty"`
	RuntimeElapsedMinutes  int      `json:"runtimeElapsedMinutes,omitempty"`
	RuntimeBudgetMinutes   int      `json:"runtimeBudgetMinutes,omitempty"`
	ReviewElapsedMinutes   int      `json:"reviewElapsedMinutes,omitempty"`
	ReviewBudgetMinutes    int      `json:"reviewBudgetMinutes,omitempty"`
	ReviewTimestampMissing bool     `json:"reviewTimestampMissing,omitempty"`
}

type BudgetPressureSummary struct {
	EvidenceLabel              string   `json:"evidenceLabel"`
	Warnings                   []string `json:"warnings,omitempty"`
	TasksMissingBudget         int      `json:"tasksMissingBudget"`
	TasksNearLimit             int      `json:"tasksNearLimit"`
	TasksExceeded              int      `json:"tasksExceeded"`
	TasksWithUnknownReviewTime int      `json:"tasksWithUnknownReviewTime"`
	RoutineSpecsMissingBudget  int      `json:"routineSpecsMissingBudget,omitempty"`
}

type RuntimeStatusItem struct {
	ID                string         `json:"id"`
	Title             string         `json:"title,omitempty"`
	LedgerStatus      string         `json:"ledgerStatus,omitempty"`
	ObservedStatus    string         `json:"observedStatus,omitempty"`
	Signal            string         `json:"signal,omitempty"`
	Branch            string         `json:"branch,omitempty"`
	Worktree          string         `json:"worktree,omitempty"`
	PendingWorktreeID string         `json:"pendingWorktreeId,omitempty"`
	LastUpdatedAt     string         `json:"lastUpdatedAt,omitempty"`
	Action            string         `json:"action,omitempty"`
	Note              string         `json:"note,omitempty"`
	State             LocalTaskState `json:"state"`
}

type RuntimeStatusReport struct {
	EvidenceLabel          string              `json:"evidenceLabel"`
	Summary                string              `json:"summary"`
	RecentWindowHours      int                 `json:"recentWindowHours"`
	MaxConcurrency         int                 `json:"maxConcurrency"`
	UsedDispatchSlots      int                 `json:"usedDispatchSlots"`
	AvailableDispatchSlots int                 `json:"availableDispatchSlots"`
	ActiveWorkers          []RuntimeStatusItem `json:"activeWorkers,omitempty"`
	PendingSetup           []RuntimeStatusItem `json:"pendingSetup,omitempty"`
	DirtyUncommitted       []RuntimeStatusItem `json:"dirtyUncommitted,omitempty"`
	CompletedUnreviewed    []RuntimeStatusItem `json:"completedUnreviewed,omitempty"`
	StaleNeedsInspection   []RuntimeStatusItem `json:"staleNeedsInspection,omitempty"`
	Blockers               []RuntimeStatusItem `json:"blockers,omitempty"`
	CleanupNeeded          []RuntimeStatusItem `json:"cleanupNeeded,omitempty"`
	RecentMergedOrCleaned  []RuntimeStatusItem `json:"recentMergedOrCleaned,omitempty"`
}

type routineBudgetCoverage struct {
	Total               int
	WithMaxRuntime      int
	WithReviewBudget    int
	WithBoth            int
	MissingMaxRuntime   []string
	MissingReviewBudget []string
	WithAnyBudget       int
	WithoutAnyBudget    []string
}

type ObserveSummary struct {
	Ledger             string                `json:"ledger"`
	Version            int                   `json:"version"`
	ProjectRoot        string                `json:"projectRoot"`
	DefaultBranch      string                `json:"defaultBranch"`
	ObservedAt         string                `json:"observedAt"`
	OverallStatus      string                `json:"overallStatus"`
	RecommendedActions []string              `json:"recommendedActions"`
	Counts             map[string]int        `json:"counts"`
	ReviewPressure     ReviewPressure        `json:"reviewPressure"`
	BudgetSummary      BudgetSummary         `json:"budgetSummary"`
	BudgetPressure     BudgetPressureSummary `json:"budgetPressure"`
	Integration        IntegrationState      `json:"integration"`
	RuntimeStatus      RuntimeStatusReport   `json:"runtimeStatus"`
	Observations       []Observation         `json:"observations"`
	RecentRoutineRuns  []RoutineRun          `json:"recentRoutineRuns,omitempty"`
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
	case "policy":
		return cmdPolicy(args[1:])
	case "eval":
		return cmdEval(args[1:])
	case "rules":
		return cmdRules(args[1:])
	case "record-routine-run":
		return cmdRecordRoutineRun(args[1:])
	case "completion":
		return cmdCompletion(args[1:])
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
  codex-orchestrator record-task --id ID (--worktree PATH --branch BRANCH | --pending-worktree-id ID) [--allowed PATH] [--forbidden PATH] [--gate CMD] [--max-runtime-minutes N] [--review-budget-minutes N]
  codex-orchestrator append-event --type TYPE [--task-id ID] [--status STATUS] [--worktree PATH] [--branch BRANCH] [--pending-worktree-id ID] [--note TEXT]
  codex-orchestrator observe [--repo PATH] [--ledger PATH] [--json] [--write-report PATH] [--write-summary PATH]
  codex-orchestrator heartbeat [--repo PATH] [--ledger PATH] [--interval 5m] [--count 0] [--write-report PATH]
  codex-orchestrator status [--repo PATH] [--ledger PATH] [--json] [--stale-after 15m]
  codex-orchestrator validate-routines [--dir routines] [--json]
  codex-orchestrator run-routine pr-reviewer --task-id TASK [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine stale-task-rescuer --task-id TASK [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine ci-fixer --task-id TASK [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine release-verifier --tag TAG [--repo PATH] [--expected-asset NAME] [--write-report PATH] [--json]
  codex-orchestrator run-routine docs-drift-checker [--repo PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine evidence-label-auditor [--repo PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine orchestration-policy-auditor [--repo PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine roadmap-next-task-suggester [--repo PATH] [--ledger PATH] [--write-report PATH] [--json]
  codex-orchestrator run-routine budget-policy-report [--repo PATH] [--ledger PATH] [--heartbeat-report PATH] [--write-report PATH] [--json]
  codex-orchestrator policy check [--repo PATH] [--eval-dir PATH] [--write-report PATH] [--json]
  codex-orchestrator eval run [--suite orchestration-policy-auditor] [--repo PATH] [--eval-dir PATH] [--write-report PATH] [--json]
  codex-orchestrator eval add-failure --id ID --text TEXT --expect OPA001=1 [--file README.md] [--suite orchestration-policy-auditor] [--repo PATH]
  codex-orchestrator rules propose (--from-review PATH | --text TEXT | --text-file PATH) [--write-report PATH] [--json]
  codex-orchestrator record-routine-run --routine ID --status passed|failed|blocked [--task-id TASK]
  codex-orchestrator record-routine-run --report-json PATH
  codex-orchestrator completion bash|zsh|fish

This helper is conservative: it does not create Codex sessions, merge, push,
delete branches, or clean worktrees.`)
}

func cmdCompletion(args []string) error {
	if len(args) != 1 {
		return errors.New("usage: codex-orchestrator completion bash|zsh|fish")
	}
	switch args[0] {
	case "bash":
		fmt.Print(completionBash())
	case "zsh":
		fmt.Print(completionZsh())
	case "fish":
		fmt.Print(completionFish())
	default:
		return fmt.Errorf("unsupported shell %q; expected bash, zsh, or fish", args[0])
	}
	return nil
}

func completionBash() string {
	return `# bash completion for codex-orchestrator
_codex_orchestrator()
{
  local cur prev commands routines
  COMPREPLY=()
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev="${COMP_WORDS[COMP_CWORD-1]}"
  commands="init record-task append-event observe heartbeat status validate-routines run-routine policy eval rules record-routine-run completion help"
  routines="pr-reviewer stale-task-rescuer ci-fixer release-verifier docs-drift-checker evidence-label-auditor orchestration-policy-auditor roadmap-next-task-suggester budget-policy-report"

  case "$prev" in
    codex-orchestrator)
      COMPREPLY=( $(compgen -W "$commands" -- "$cur") )
      return 0
      ;;
    run-routine)
      COMPREPLY=( $(compgen -W "$routines" -- "$cur") )
      return 0
      ;;
    completion)
      COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
      return 0
      ;;
  esac

  case "${COMP_WORDS[1]}" in
    init)
      COMPREPLY=( $(compgen -W "--ledger --project-root --help" -- "$cur") )
      ;;
    record-task)
      COMPREPLY=( $(compgen -W "--ledger --id --title --thread-id --pending-worktree-id --worktree --branch --base-commit --allowed --forbidden --gate --evidence --max-runtime-minutes --review-budget-minutes --budget-note --help" -- "$cur") )
      ;;
    append-event)
      COMPREPLY=( $(compgen -W "--ledger --type --task-id --status --pending-worktree-id --worktree --branch --note --help" -- "$cur") )
      ;;
    observe)
      COMPREPLY=( $(compgen -W "--repo --ledger --json --write-report --write-summary --stale-after --help" -- "$cur") )
      ;;
    status)
      COMPREPLY=( $(compgen -W "--repo --ledger --json --stale-after --help" -- "$cur") )
      ;;
    heartbeat)
      COMPREPLY=( $(compgen -W "--repo --ledger --interval --count --write-report --write-summary --help" -- "$cur") )
      ;;
    validate-routines)
      COMPREPLY=( $(compgen -W "--dir --json --help" -- "$cur") )
      ;;
    run-routine)
      COMPREPLY=( $(compgen -W "--ledger --task-id --repo --tag --expected-asset --heartbeat-report --write-report --json --help" -- "$cur") )
      ;;
    policy)
      if [[ ${COMP_WORDS[2]} == "check" ]]; then
        COMPREPLY=( $(compgen -W "--repo --eval-dir --write-report --json --help" -- "$cur") )
      else
        COMPREPLY=( $(compgen -W "check" -- "$cur") )
      fi
      ;;
    eval)
      if [[ ${COMP_WORDS[2]} == "run" ]]; then
        COMPREPLY=( $(compgen -W "--suite --repo --eval-dir --write-report --json --help" -- "$cur") )
      elif [[ ${COMP_WORDS[2]} == "add-failure" ]]; then
        COMPREPLY=( $(compgen -W "--suite --repo --eval-dir --id --description --file --text --text-file --expect --force --json --help" -- "$cur") )
      else
        COMPREPLY=( $(compgen -W "run add-failure" -- "$cur") )
      fi
      ;;
    rules)
      if [[ ${COMP_WORDS[2]} == "propose" ]]; then
        COMPREPLY=( $(compgen -W "--from-review --text --text-file --write-report --json --help" -- "$cur") )
      else
        COMPREPLY=( $(compgen -W "propose" -- "$cur") )
      fi
      ;;
    record-routine-run)
      COMPREPLY=( $(compgen -W "--ledger --routine --status --task-id --evidence-local --evidence-proxy --evidence-direct --evidence-blocked --action --next --needs-human --blocked-reason --report-json --help" -- "$cur") )
      ;;
  esac
}
complete -F _codex_orchestrator codex-orchestrator
`
}

func completionZsh() string {
	return `#compdef codex-orchestrator

local -a commands routines
commands=(
  'init:initialize a project-local ledger'
  'record-task:record a delegated task'
  'append-event:append a task or heartbeat event'
  'observe:inspect ledger and worktree state'
  'heartbeat:run observe on an interval and write reports'
  'status:print ledger status'
  'validate-routines:validate routine specs'
  'run-routine:run a read-only routine'
  'policy:run policy and eval checks'
  'eval:run local eval fixtures'
  'rules:propose review-only rule updates'
  'record-routine-run:record a routine report in the ledger'
  'completion:print shell completion'
  'help:show help'
)
routines=(
  'pr-reviewer'
  'stale-task-rescuer'
  'ci-fixer'
  'release-verifier'
  'docs-drift-checker'
  'evidence-label-auditor'
  'orchestration-policy-auditor'
  'roadmap-next-task-suggester'
  'budget-policy-report'
)

_arguments -C \
  '1:command:->command' \
  '*::arg:->args'

case $state in
  command)
    _describe 'command' commands
    ;;
  args)
    case $words[2] in
      run-routine)
        if (( CURRENT == 3 )); then
          _describe 'routine' routines
        else
          _values 'options' --ledger --task-id --repo --tag --expected-asset --heartbeat-report --write-report --json --help
        fi
        ;;
      policy)
        if (( CURRENT == 3 )); then
          _values 'subcommand' check
        else
          _values 'options' --repo --eval-dir --write-report --json --help
        fi
        ;;
      eval)
        if (( CURRENT == 3 )); then
          _values 'subcommand' run add-failure
        elif [[ $words[3] == "add-failure" ]]; then
          _values 'options' --suite --repo --eval-dir --id --description --file --text --text-file --expect --force --json --help
        else
          _values 'options' --suite --repo --eval-dir --write-report --json --help
        fi
        ;;
      rules)
        if (( CURRENT == 3 )); then
          _values 'subcommand' propose
        else
          _values 'options' --from-review --text --text-file --write-report --json --help
        fi
        ;;
      completion)
        _values 'shell' bash zsh fish
        ;;
      init)
        _values 'options' --ledger --project-root --help
        ;;
      record-task)
        _values 'options' --ledger --id --title --thread-id --pending-worktree-id --worktree --branch --base-commit --allowed --forbidden --gate --evidence --max-runtime-minutes --review-budget-minutes --budget-note --help
        ;;
      append-event)
        _values 'options' --ledger --type --task-id --status --pending-worktree-id --worktree --branch --note --help
        ;;
      observe)
        _values 'options' --repo --ledger --json --write-report --write-summary --stale-after --help
        ;;
      status)
        _values 'options' --repo --ledger --json --stale-after --help
        ;;
      heartbeat)
        _values 'options' --repo --ledger --interval --count --write-report --write-summary --help
        ;;
      validate-routines)
        _values 'options' --dir --json --help
        ;;
      record-routine-run)
        _values 'options' --ledger --routine --status --task-id --evidence-local --evidence-proxy --evidence-direct --evidence-blocked --action --next --needs-human --blocked-reason --report-json --help
        ;;
    esac
    ;;
esac
`
}

func completionFish() string {
	return `# fish completion for codex-orchestrator
complete -c codex-orchestrator -f
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'init' -d 'Initialize a project-local ledger'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'record-task' -d 'Record a delegated task'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'append-event' -d 'Append a task or heartbeat event'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'observe' -d 'Inspect ledger and worktree state'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'heartbeat' -d 'Run observe on an interval and write reports'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'status' -d 'Print ledger status'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'validate-routines' -d 'Validate routine specs'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'run-routine' -d 'Run a read-only routine'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'policy' -d 'Run policy and eval checks'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'eval' -d 'Run local eval fixtures'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'rules' -d 'Propose review-only rule updates'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'record-routine-run' -d 'Record a routine report in the ledger'
complete -c codex-orchestrator -n '__fish_use_subcommand' -a 'completion' -d 'Print shell completion'
complete -c codex-orchestrator -n '__fish_seen_subcommand_from run-routine' -a 'pr-reviewer stale-task-rescuer ci-fixer release-verifier docs-drift-checker evidence-label-auditor orchestration-policy-auditor roadmap-next-task-suggester budget-policy-report'
complete -c codex-orchestrator -n '__fish_seen_subcommand_from policy' -a 'check'
complete -c codex-orchestrator -n '__fish_seen_subcommand_from eval' -a 'run add-failure'
complete -c codex-orchestrator -n '__fish_seen_subcommand_from rules' -a 'propose'
complete -c codex-orchestrator -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
complete -c codex-orchestrator -l ledger -d 'Ledger path'
complete -c codex-orchestrator -l json -d 'Print JSON'
complete -c codex-orchestrator -l write-report -d 'Write JSON report'
complete -c codex-orchestrator -l write-summary -d 'Write Markdown summary'
complete -c codex-orchestrator -l stale-after -d 'Stale threshold'
complete -c codex-orchestrator -l task-id -d 'Task id'
complete -c codex-orchestrator -l pending-worktree-id -d 'Opaque Codex App pending worktree setup id'
complete -c codex-orchestrator -l worktree -d 'Task worktree path'
complete -c codex-orchestrator -l branch -d 'Task branch'
complete -c codex-orchestrator -l repo -d 'Repository path'
complete -c codex-orchestrator -l tag -d 'Release tag'
complete -c codex-orchestrator -l expected-asset -d 'Expected release asset'
complete -c codex-orchestrator -l heartbeat-report -d 'Optional heartbeat report path'
`
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
	pendingWorktreeID := fs.String("pending-worktree-id", "", "opaque Codex App pending worktree setup id")
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
	if *worktree == "" && *pendingWorktreeID == "" {
		return errors.New("record-task requires --worktree or --pending-worktree-id")
	}
	if *worktree != "" && *branch == "" {
		return errors.New("record-task requires --branch when --worktree is set")
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
	taskStatus := *status
	if !flagProvided(fs, "status") && *worktree == "" && *pendingWorktreeID != "" {
		taskStatus = "pending-setup"
	}
	observationNote := "Task recorded."
	if *worktree == "" && *pendingWorktreeID != "" {
		observationNote = "Pending worktree setup recorded."
	}
	task := Task{
		ID:                *id,
		Title:             taskTitle,
		ThreadID:          *threadID,
		PendingWorktreeID: *pendingWorktreeID,
		Worktree:          *worktree,
		Branch:            *branch,
		BaseCommit:        base,
		Status:            taskStatus,
		Budget:            taskBudgetFromFlags(*maxRuntimeMinutes, *reviewBudgetMinutes, *budgetNote),
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
			"result": taskStatus,
			"note":   observationNote,
		},
		History: []map[string]string{{
			"at":     now,
			"type":   "record-task",
			"status": taskStatus,
			"note":   historyNote,
		}},
	}
	if *pendingWorktreeID != "" {
		task.History[0]["pendingWorktreeId"] = *pendingWorktreeID
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
		"status": taskStatus,
	}
	if *pendingWorktreeID != "" {
		event["pendingWorktreeId"] = *pendingWorktreeID
	}
	if *worktree != "" {
		event["worktree"] = *worktree
	}
	if *branch != "" {
		event["branch"] = *branch
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
	pendingWorktreeID := fs.String("pending-worktree-id", "", "opaque Codex App pending worktree setup id")
	worktree := fs.String("worktree", "", "task worktree path to record on the ledger task")
	branch := fs.String("branch", "", "task branch to record on the ledger task")
	note := fs.String("note", "", "event note")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *eventType == "" {
		return errors.New("append-event requires --type")
	}
	if *taskID == "" && (*pendingWorktreeID != "" || *worktree != "" || *branch != "") {
		return errors.New("append-event requires --task-id when updating pending-worktree-id, worktree, or branch")
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
	if *pendingWorktreeID != "" {
		event["pendingWorktreeId"] = *pendingWorktreeID
	}
	if *worktree != "" {
		event["worktree"] = *worktree
	}
	if *branch != "" {
		event["branch"] = *branch
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
		if *pendingWorktreeID != "" {
			task.PendingWorktreeID = *pendingWorktreeID
		}
		if *worktree != "" {
			task.Worktree = *worktree
		}
		if *branch != "" {
			task.Branch = *branch
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
	repo := fs.String("repo", ".", "repository path used to resolve the default ledger")
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	jsonOut := fs.Bool("json", false, "print JSON")
	writeReport := fs.String("write-report", "", "write JSON report")
	writeSummary := fs.String("write-summary", "", "write Markdown summary")
	staleAfter := fs.Duration("stale-after", 15*time.Minute, "stale threshold")
	if err := fs.Parse(args); err != nil {
		return err
	}
	resolvedLedger := resolveDefaultLedgerPath(*repo, *ledgerPath, flagProvided(fs, "ledger"))
	summary, err := observeWithOptions(resolvedLedger, *staleAfter)
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
	repo := fs.String("repo", ".", "repository path used to resolve the default ledger")
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
	resolvedLedger := resolveDefaultLedgerPath(*repo, *ledgerPath, flagProvided(fs, "ledger"))
	resolvedEvents := *eventsPath
	if resolvedEvents == "" {
		resolvedEvents = eventsPathForLedger(resolvedLedger)
	}
	iteration := 0
	for {
		iteration++
		summary, err := observeWithOptions(resolvedLedger, *staleAfter)
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
	repo := fs.String("repo", ".", "repository path used to resolve the default ledger")
	ledgerPath := fs.String("ledger", defaultLedger, "ledger path")
	jsonOut := fs.Bool("json", false, "print JSON")
	staleAfter := fs.Duration("stale-after", 15*time.Minute, "stale threshold")
	if err := fs.Parse(args); err != nil {
		return err
	}
	resolvedLedger := resolveDefaultLedgerPath(*repo, *ledgerPath, flagProvided(fs, "ledger"))
	ledger, err := loadLedger(resolvedLedger)
	if err != nil {
		return err
	}
	ledgerCounts := map[string]int{}
	for _, task := range ledger.Tasks {
		status := task.Status
		if status == "" {
			status = "unknown"
		}
		ledgerCounts[status]++
	}
	summary, err := observeWithOptions(resolvedLedger, *staleAfter)
	if err != nil {
		return err
	}
	result := map[string]any{
		"ledger":            resolvedLedger,
		"projectRoot":       ledger.ProjectRoot,
		"defaultBranch":     ledger.DefaultBranch,
		"taskCount":         len(ledger.Tasks),
		"routineRunCount":   len(ledger.RoutineRuns),
		"overallStatus":     summary.OverallStatus,
		"counts":            summary.Counts,
		"ledgerCounts":      ledgerCounts,
		"reviewPressure":    summary.ReviewPressure,
		"budgetSummary":     summary.BudgetSummary,
		"budgetPressure":    summary.BudgetPressure,
		"integration":       summary.Integration,
		"runtimeStatus":     summary.RuntimeStatus,
		"tasks":             ledger.Tasks,
		"observations":      summary.Observations,
		"recentRoutineRuns": summary.RecentRoutineRuns,
	}
	if *jsonOut {
		return printJSON(result)
	}
	fmt.Printf("Ledger: %s\n", resolvedLedger)
	fmt.Printf("Project: %s default=%s\n", ledger.ProjectRoot, ledger.DefaultBranch)
	fmt.Printf("Tasks: %d overall=%s\n", len(ledger.Tasks), summary.OverallStatus)
	fmt.Printf("Runtime status (%s): %s\n", summary.RuntimeStatus.EvidenceLabel, summary.RuntimeStatus.Summary)
	fmt.Printf("Dispatch slots: used=%d/%d available=%d\n",
		summary.RuntimeStatus.UsedDispatchSlots,
		summary.RuntimeStatus.MaxConcurrency,
		summary.RuntimeStatus.AvailableDispatchSlots,
	)
	printRuntimeStatusReport(summary.RuntimeStatus)
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
	case "orchestration-policy-auditor":
		return cmdRunOrchestrationPolicyAuditorRoutine(args[1:])
	case "roadmap-next-task-suggester":
		return cmdRunRoadmapNextTaskSuggesterRoutine(args[1:])
	case "budget-policy-report":
		return cmdRunBudgetPolicyReportRoutine(args[1:])
	default:
		return fmt.Errorf("unsupported routine %q", args[0])
	}
}

func cmdPolicy(args []string) error {
	if len(args) == 0 {
		return errors.New("policy requires a subcommand: check")
	}
	switch args[0] {
	case "check":
		return cmdPolicyCheck(args[1:])
	default:
		return fmt.Errorf("unsupported policy subcommand %q", args[0])
	}
}

func cmdEval(args []string) error {
	if len(args) == 0 {
		return errors.New("eval requires a subcommand: run or add-failure")
	}
	switch args[0] {
	case "run":
		return cmdEvalRun(args[1:])
	case "add-failure":
		return cmdEvalAddFailure(args[1:])
	default:
		return fmt.Errorf("unsupported eval subcommand %q", args[0])
	}
}

func cmdRules(args []string) error {
	if len(args) == 0 {
		return errors.New("rules requires a subcommand: propose")
	}
	switch args[0] {
	case "propose":
		return cmdRulesPropose(args[1:])
	default:
		return fmt.Errorf("unsupported rules subcommand %q", args[0])
	}
}

func cmdEvalRun(args []string) error {
	fs := flag.NewFlagSet("eval run", flag.ExitOnError)
	suite := fs.String("suite", "orchestration-policy-auditor", "eval suite id")
	repo := fs.String("repo", ".", "repository path used to resolve relative eval dirs")
	evalDir := fs.String("eval-dir", "", "eval fixture directory; defaults to eval/SUITE")
	writeReport := fs.String("write-report", "", "write eval run report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runEvalSuite(*repo, *suite, *evalDir)
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
	fmt.Printf("Wrote eval run report: %s\n", *writeReport)
	return nil
}

func cmdRulesPropose(args []string) error {
	fs := flag.NewFlagSet("rules propose", flag.ExitOnError)
	fromReview := fs.String("from-review", "", "read local review text from a Markdown/text file")
	text := fs.String("text", "", "inline local evidence text")
	textFile := fs.String("text-file", "", "read local evidence text from a file")
	writeReport := fs.String("write-report", "", "write rule proposal report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runRulesPropose(*fromReview, *text, *textFile)
	if *writeReport != "" {
		if err := writeJSON(*writeReport, report); err != nil {
			return err
		}
	}
	if *jsonOut {
		return printJSON(report)
	}
	if *writeReport == "" {
		fmt.Print(renderRuleProposalReport(report))
		return nil
	}
	fmt.Printf("Wrote rules proposal report: %s\n", *writeReport)
	return nil
}

func runRulesPropose(fromReview string, text string, textFile string) RuleProposalReport {
	report := RuleProposalReport{
		SchemaVersion:    1,
		Command:          "rules propose",
		GeneratedAt:      nowISO(),
		Status:           "blocked",
		Source:           RuleSource{},
		EvidenceLabel:    "blocked",
		NeedsHumanReview: true,
		Proposals:        []RuleProposal{},
		Evidence:         []string{},
		BlockedReason:    "insufficient local input",
		NextAction:       "Provide --from-review, --text, or --text-file with local evidence, then rerun rules propose.",
	}
	body, source, err := readRulesProposalInput(fromReview, text, textFile)
	report.Source = source
	if err != nil {
		report.Evidence = append(report.Evidence, "Blocked: "+err.Error())
		report.BlockedReason = err.Error()
		return report
	}
	if len(strings.Fields(body)) < 4 {
		report.Evidence = append(report.Evidence, "Blocked: local input is too short to support a reviewable rule proposal.")
		report.BlockedReason = "local input is too short"
		return report
	}

	sourceName := source.Path
	if sourceName == "" {
		sourceName = "inline text"
	}
	findings := auditOrchestrationPolicyText(sourceName, body)
	ruleCounts := countPolicyAuditFindings(findings)
	report.Status = "passed"
	report.EvidenceLabel = "local"
	report.BlockedReason = ""
	report.Evidence = append(report.Evidence, fmt.Sprintf("Read local rule proposal input from %s.", sourceName))
	if len(findings) > 0 {
		report.Evidence = append(report.Evidence, fmt.Sprintf("Applied existing OPA policy heuristics read-only: %s.", formatRuleCounts(ruleCounts)))
	} else {
		report.Evidence = append(report.Evidence, "Existing OPA policy heuristics produced no named rule hit; generated one generic review-only proposal from local text.")
	}
	report.Proposals = buildRuleProposals(sourceName, findings)
	report.Evidence = append(report.Evidence, fmt.Sprintf("Generated %d review-only proposal(s); no rule, skill, README, policy, AGENTS, or CLAUDE file was edited.", len(report.Proposals)))
	report.NextAction = "A human reviewer should accept, rewrite, or reject these proposed rule updates before editing any live rules."
	return report
}

func readRulesProposalInput(fromReview string, text string, textFile string) (string, RuleSource, error) {
	fromReview = strings.TrimSpace(fromReview)
	text = strings.TrimSpace(text)
	textFile = strings.TrimSpace(textFile)
	count := 0
	for _, value := range []string{fromReview, text, textFile} {
		if value != "" {
			count++
		}
	}
	if count == 0 {
		return "", RuleSource{}, errors.New("rules propose requires one of --from-review, --text, or --text-file")
	}
	if count > 1 {
		return "", RuleSource{}, errors.New("use only one of --from-review, --text, or --text-file")
	}
	if fromReview != "" {
		data, err := os.ReadFile(expandPath(fromReview))
		if err != nil {
			return "", RuleSource{Kind: "review", Path: fromReview}, err
		}
		return strings.TrimSpace(string(data)), RuleSource{Kind: "review", Path: fromReview}, nil
	}
	if textFile != "" {
		data, err := os.ReadFile(expandPath(textFile))
		if err != nil {
			return "", RuleSource{Kind: "text-file", Path: textFile}, err
		}
		return strings.TrimSpace(string(data)), RuleSource{Kind: "text-file", Path: textFile}, nil
	}
	return text, RuleSource{Kind: "text"}, nil
}

func buildRuleProposals(sourceName string, findings []policyAuditFinding) []RuleProposal {
	ruleIDs := make([]string, 0, len(findings))
	seen := map[string]bool{}
	for _, finding := range findings {
		if seen[finding.RuleID] {
			continue
		}
		seen[finding.RuleID] = true
		ruleIDs = append(ruleIDs, finding.RuleID)
	}
	sort.Strings(ruleIDs)
	if len(ruleIDs) == 0 {
		return []RuleProposal{{
			Title:            "Review local orchestration rule update",
			Body:             "Consider adding or tightening a project rule only after a human confirms this local evidence represents a repeated orchestration failure. Keep the accepted rule specific, testable, and paired with the verification evidence it requires.",
			Source:           sourceName,
			EvidenceLabel:    "local",
			NeedsHumanReview: true,
		}}
	}
	proposals := make([]RuleProposal, 0, len(ruleIDs))
	for _, ruleID := range ruleIDs {
		title, proposalBody := proposalForPolicyRule(ruleID)
		proposals = append(proposals, RuleProposal{
			Title:            title,
			Body:             proposalBody,
			Source:           sourceName,
			EvidenceLabel:    "local",
			NeedsHumanReview: true,
			RuleIDs:          []string{ruleID},
		})
	}
	return proposals
}

func proposalForPolicyRule(ruleID string) (string, string) {
	switch ruleID {
	case policyRuleDryRunBarrier:
		return "Require explicit approval before dispatch after dry run", "Proposed rule: after any dry-run or planning-only pass, do not dispatch workers or mutate task state until the user explicitly approves live execution. Verification should cite the approval text or mark the dispatch path blocked."
	case policyRuleMainFallbackGuard:
		return "Block main-checkout fallback after worker setup failure", "Proposed rule: if a delegated worktree or branch cannot be created, stop and report setup failure instead of implementing in the orchestrator checkout or another unapproved path. Verification should include the failed setup evidence and the exact input needed to continue."
	case policyRuleContinuationGuard:
		return "Continue the parent queue after one child task completes", "Proposed rule: completing one delegated task is not completion for the parent orchestration unless the parent objective is fully verified. After each child closes, reread the queue, classify remaining safe work, and record whether the parent continues, blocks, or completes."
	case policyRuleWorkerBoundary:
		return "Require delegated worker boundary instructions", "Proposed rule: every delegated worker prompt must name its allowed paths, forbidden paths, branch/worktree boundary, verification gates, and the prohibition on nested delegation unless the user explicitly requests it."
	case policyRuleEvidenceBoundary:
		return "Prevent local or proxy evidence from becoming direct proof", "Proposed rule: local, static, fixture, or proxy checks must stay labeled as local or proxy evidence. Only the routine that directly observes the target runtime, device, deployment, or external surface may record direct evidence."
	default:
		return "Review local orchestration rule update", "Proposed rule: convert the confirmed repeated failure into a narrow, reviewable rule with an explicit trigger, forbidden action, verification surface, and blocked-stop condition."
	}
}

func renderRuleProposalReport(report RuleProposalReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# codex-orchestrator rules proposal\n\n")
	fmt.Fprintf(&b, "- status: `%s`\n", report.Status)
	fmt.Fprintf(&b, "- generatedAt: `%s`\n", report.GeneratedAt)
	fmt.Fprintf(&b, "- sourceKind: `%s`\n", report.Source.Kind)
	if report.Source.Path != "" {
		fmt.Fprintf(&b, "- sourcePath: `%s`\n", report.Source.Path)
	}
	fmt.Fprintf(&b, "- evidenceLabel: `%s`\n", report.EvidenceLabel)
	fmt.Fprintf(&b, "- needsHumanReview: `%t`\n", report.NeedsHumanReview)
	if report.BlockedReason != "" {
		fmt.Fprintf(&b, "- blockedReason: %s\n", report.BlockedReason)
	}
	if len(report.Evidence) > 0 {
		fmt.Fprintf(&b, "\n## Evidence\n\n")
		for _, item := range report.Evidence {
			fmt.Fprintf(&b, "- %s\n", item)
		}
	}
	if len(report.Proposals) > 0 {
		fmt.Fprintf(&b, "\n## Proposals\n\n")
		for _, proposal := range report.Proposals {
			fmt.Fprintf(&b, "### %s\n\n", proposal.Title)
			fmt.Fprintf(&b, "- source: `%s`\n", proposal.Source)
			fmt.Fprintf(&b, "- evidenceLabel: `%s`\n", proposal.EvidenceLabel)
			fmt.Fprintf(&b, "- needsHumanReview: `%t`\n", proposal.NeedsHumanReview)
			if len(proposal.RuleIDs) > 0 {
				fmt.Fprintf(&b, "- ruleIds: `%s`\n", strings.Join(proposal.RuleIDs, ", "))
			}
			fmt.Fprintf(&b, "\n%s\n\n", proposal.Body)
		}
	}
	fmt.Fprintf(&b, "## Next Action\n\n%s\n", report.NextAction)
	return b.String()
}

func cmdEvalAddFailure(args []string) error {
	fs := flag.NewFlagSet("eval add-failure", flag.ExitOnError)
	suite := fs.String("suite", "orchestration-policy-auditor", "eval suite id")
	repo := fs.String("repo", ".", "repository path used to resolve relative eval dirs")
	evalDir := fs.String("eval-dir", "", "eval fixture directory; defaults to eval/SUITE")
	id := fs.String("id", "", "fixture id; used as the JSON filename")
	description := fs.String("description", "", "fixture description")
	fileName := fs.String("file", "README.md", "synthetic file path stored inside the fixture")
	text := fs.String("text", "", "failure text to store in the fixture")
	textFile := fs.String("text-file", "", "read failure text from a file instead of --text")
	force := fs.Bool("force", false, "overwrite an existing fixture")
	jsonOut := fs.Bool("json", false, "print JSON result")
	var expects stringList
	fs.Var(&expects, "expect", "expected rule hit in RULE=N form; may be repeated")
	if err := fs.Parse(args); err != nil {
		return err
	}
	result, err := addFailureEvalFixture(*repo, *suite, *evalDir, *id, *description, *fileName, *text, *textFile, []string(expects), *force)
	if err != nil {
		return err
	}
	if *jsonOut {
		return printJSON(result)
	}
	fmt.Printf("Wrote eval fixture: %s\n", result.Path)
	fmt.Printf("Expected rule hits: %s\n", formatRuleCounts(result.ExpectedRuleHits))
	return nil
}

func runEvalSuite(repo string, suite string, evalDir string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "eval-run",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Ran local eval fixtures read-only",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked eval run precondition, then rerun eval run.",
	}
	repo = expandPath(repo)
	if repo == "" {
		repo = "."
	}
	suite = strings.TrimSpace(suite)
	if suite == "" {
		suite = "orchestration-policy-auditor"
	}
	if evalDir == "" {
		evalDir = filepath.Join("eval", suite)
	}
	switch suite {
	case "orchestration-policy-auditor":
		result := runOrchestrationPolicyEvalFixtures(repo, evalDir)
		report.Evidence["local"] = append(report.Evidence["local"], "Eval suite: "+suite)
		report.Evidence["local"] = append(report.Evidence["local"], result.Evidence...)
		report.Evidence["blocked"] = append(report.Evidence["blocked"], result.Blocked...)
		if result.BlockedReason != "" {
			report.BlockedReason = result.BlockedReason
			return report
		}
		if !result.Passed {
			report.Status = "failed"
			report.NextSuggestedAction = "Fix policy rule or fixture expectation mismatches, then rerun eval run."
			return report
		}
		report.Status = "passed"
		report.BlockedReason = ""
		report.NextSuggestedAction = "Record this local/static eval report if sufficient; no repository scan, Codex App session, or runtime proof was produced."
		return report
	default:
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Unsupported eval suite: "+suite)
		report.BlockedReason = "unsupported eval suite"
		return report
	}
}

func cmdPolicyCheck(args []string) error {
	fs := flag.NewFlagSet("policy check", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository path to inspect")
	evalDir := fs.String("eval-dir", filepath.Join("eval", "orchestration-policy-auditor"), "policy eval fixture directory; relative paths are resolved under --repo")
	writeReport := fs.String("write-report", "", "write policy check report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runPolicyCheck(*repo, *evalDir)
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
	fmt.Printf("Wrote policy check report: %s\n", *writeReport)
	return nil
}

func runPolicyCheck(repo string, evalDir string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "policy-check",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Ran read-only orchestration policy auditor",
			"Ran local policy eval fixtures when available",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked policy check precondition, then rerun policy check.",
	}
	repo = expandPath(repo)
	if repo == "" {
		repo = "."
	}
	auditor := runOrchestrationPolicyAuditorRoutine(repo)
	report.Evidence["local"] = append(report.Evidence["local"], fmt.Sprintf("orchestration-policy-auditor status: %s", auditor.Status))
	report.Evidence["local"] = append(report.Evidence["local"], auditor.Evidence["local"]...)
	report.Evidence["blocked"] = append(report.Evidence["blocked"], auditor.Evidence["blocked"]...)

	evalResult := runOrchestrationPolicyEvalFixtures(repo, evalDir)
	report.Evidence["local"] = append(report.Evidence["local"], evalResult.Evidence...)
	report.Evidence["blocked"] = append(report.Evidence["blocked"], evalResult.Blocked...)

	if auditor.Status == "blocked" {
		report.BlockedReason = firstNonEmpty(auditor.BlockedReason, "orchestration policy auditor blocked")
		return report
	}
	if evalResult.BlockedReason != "" {
		report.BlockedReason = evalResult.BlockedReason
		return report
	}
	if auditor.Status == "failed" || !evalResult.Passed {
		report.Status = "failed"
		report.NextSuggestedAction = "Review policy findings or fixture mismatches, fix confirmed rule/docs issues, then rerun policy check."
		return report
	}
	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Record this local/static policy check if sufficient; no Codex App session, daemon, or runtime proof was produced."
	return report
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

	changedPaths := parseNameStatusPaths(nameStatusOut)
	checklistFailures, checklistWarnings := evaluateReviewChecklist(task, changedPaths)
	report.ActionsTaken = append(report.ActionsTaken, "Generated local/static automated review checklist")
	checklistFailed := len(checklistFailures) > 0
	if len(checklistFailures) > 0 {
		report.Evidence["local"] = append(report.Evidence["local"], checklistFailures...)
	}
	report.Evidence["local"] = append(report.Evidence["local"], checklistWarnings...)
	if reviewChecklistNeedsHuman(checklistWarnings) {
		report.NeedsHuman = true
	}

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

	if checklistFailed {
		report.Status = "failed"
		report.NextSuggestedAction = "Return to the same task worker for a bounded fixup of the automated review checklist failures, then rerun pr-reviewer."
		return report, nil
	}

	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Review the local/static checklist and rerun narrow task gates before any separate merge decision; record it with record-routine-run --report-json only if the evidence is sufficient."
	return report, nil
}

func parseNameStatusPaths(nameStatus string) []string {
	paths := []string{}
	for _, line := range strings.Split(nameStatus, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "(") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		for _, path := range fields[1:] {
			path = normalizeRepoPath(path)
			if path != "" {
				paths = append(paths, path)
			}
		}
	}
	return uniqueSortedStrings(paths)
}

func evaluateReviewChecklist(task Task, changedPaths []string) ([]string, []string) {
	failures := []string{}
	warnings := []string{}
	allowed := cleanPathPatterns(task.WriteSet["allowed"])
	forbidden := cleanPathPatterns(task.WriteSet["forbidden"])
	if len(allowed) == 0 && len(forbidden) == 0 {
		warnings = append(warnings, "Automated review checklist: ledger writeSet has no allowed/forbidden paths; path-boundary check is advisory-only.")
	} else {
		warnings = append(warnings, fmt.Sprintf("Automated review checklist: ledger writeSet allowed=%s forbidden=%s.", formatStringList(allowed), formatStringList(forbidden)))
		if len(allowed) > 0 {
			outsideAllowed := pathsOutsidePatterns(changedPaths, allowed)
			if len(outsideAllowed) > 0 {
				failures = append(failures, "Automated review checklist failed: changed path(s) outside ledger allowed writeSet: "+formatStringList(outsideAllowed)+".")
			} else {
				warnings = append(warnings, "Automated review checklist: all changed paths fit the ledger allowed writeSet.")
			}
		}
		if len(forbidden) > 0 {
			forbiddenHits := pathsMatchingPatterns(changedPaths, forbidden)
			if len(forbiddenHits) > 0 {
				failures = append(failures, "Automated review checklist failed: changed path(s) match ledger forbidden writeSet: "+formatStringList(forbiddenHits)+".")
			} else {
				warnings = append(warnings, "Automated review checklist: no changed paths matched the ledger forbidden writeSet.")
			}
		}
	}

	warnings = append(warnings, reviewSignalEvidence(changedPaths)...)
	if len(task.Gates) > 0 {
		warnings = append(warnings, "Automated review checklist: suggested narrow gate(s) from ledger task: "+formatStringList(task.Gates)+".")
	} else {
		warnings = append(warnings, "Automated review checklist: no ledger task gates are recorded; reviewer should choose the narrowest credible local gate before merge.")
	}
	return failures, warnings
}

func reviewSignalEvidence(changedPaths []string) []string {
	signals := []string{}
	if anyPathHasPrefix(changedPaths, "docs/reviews/") {
		signals = append(signals, "Automated review checklist: review artifact signal found under docs/reviews/.")
	} else {
		signals = append(signals, "Automated review checklist warning: no changed review artifact under docs/reviews/ was locally detectable.")
	}
	if anyPathLooksLikeArtifact(changedPaths) {
		signals = append(signals, "Automated review checklist: artifact/report evidence path signal found in changed files.")
	} else {
		signals = append(signals, "Automated review checklist warning: no changed artifact/report evidence path was locally detectable.")
	}
	if anyPathContainsFold(changedPaths, []string{"self-review", "self_review", "selfreview", "handoff", "review"}) {
		signals = append(signals, "Automated review checklist: self-review or handoff filename signal found in changed files.")
	} else {
		signals = append(signals, "Automated review checklist warning: worker self-review is not locally detectable from changed filenames; verify the final handoff before merge.")
	}
	if anyPathContainsFold(changedPaths, []string{"evidence", "proof", "report", "review"}) {
		signals = append(signals, "Automated review checklist: evidence-label review signal found in changed filenames.")
	} else {
		signals = append(signals, "Automated review checklist warning: evidence-label review signal is not locally detectable from changed filenames; verify direct/proxy/local/blocked claims manually.")
	}
	return signals
}

func reviewChecklistNeedsHuman(warnings []string) bool {
	for _, warning := range warnings {
		if strings.Contains(strings.ToLower(warning), "warning:") {
			return true
		}
	}
	return false
}

func cleanPathPatterns(patterns []string) []string {
	cleaned := []string{}
	for _, pattern := range patterns {
		pattern = normalizeRepoPath(pattern)
		if pattern != "" {
			cleaned = append(cleaned, pattern)
		}
	}
	return uniqueSortedStrings(cleaned)
}

func pathsOutsidePatterns(paths []string, patterns []string) []string {
	outside := []string{}
	for _, path := range paths {
		if !pathMatchesAnyPattern(path, patterns) {
			outside = append(outside, path)
		}
	}
	return uniqueSortedStrings(outside)
}

func pathsMatchingPatterns(paths []string, patterns []string) []string {
	matches := []string{}
	for _, path := range paths {
		if pathMatchesAnyPattern(path, patterns) {
			matches = append(matches, path)
		}
	}
	return uniqueSortedStrings(matches)
}

func pathMatchesAnyPattern(path string, patterns []string) bool {
	for _, pattern := range patterns {
		if repoPathMatchesPattern(path, pattern) {
			return true
		}
	}
	return false
}

func repoPathMatchesPattern(path string, pattern string) bool {
	path = normalizeRepoPath(path)
	pattern = normalizeRepoPath(pattern)
	if path == "" || pattern == "" {
		return false
	}
	if pattern == "." || pattern == "**" || pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.HasSuffix(pattern, "/") {
		prefix := strings.TrimSuffix(pattern, "/")
		return path == prefix || strings.HasPrefix(path, prefix+"/")
	}
	if strings.ContainsAny(pattern, "*?[") {
		if matched, err := filepath.Match(pattern, path); err == nil && matched {
			return true
		}
	}
	return path == pattern || strings.HasPrefix(path, pattern+"/")
}

func normalizeRepoPath(path string) string {
	path = strings.TrimSpace(path)
	path = strings.TrimPrefix(path, "./")
	path = filepath.ToSlash(path)
	return strings.Trim(path, "/")
}

func anyPathHasPrefix(paths []string, prefix string) bool {
	prefix = normalizeRepoPath(prefix)
	for _, path := range paths {
		path = normalizeRepoPath(path)
		if path == prefix || strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func anyPathLooksLikeArtifact(paths []string) bool {
	for _, path := range paths {
		normalized := strings.ToLower(normalizeRepoPath(path))
		if strings.Contains(normalized, "artifact") ||
			strings.Contains(normalized, "report") ||
			strings.Contains(normalized, "evidence") ||
			strings.HasPrefix(normalized, "examples/routine-reports/") ||
			strings.HasPrefix(normalized, ".codex-orchestrator/") {
			return true
		}
	}
	return false
}

func anyPathContainsFold(paths []string, terms []string) bool {
	for _, path := range paths {
		if containsAnyFold(path, terms) {
			return true
		}
	}
	return false
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

	for _, doc := range routineReferenceDocs() {
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

func routineReferenceDocs() []string {
	return []string{
		"README.md",
		"README.zh-CN.md",
		"SKILL.md",
		filepath.Join("docs", "routines", "README.md"),
		filepath.Join("docs", "v2-usage.md"),
	}
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
	report.ActionsTaken = append(report.ActionsTaken, "Applied deterministic local/static evidence-label policy/eval rules ELA001-ELA010")
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

func cmdRunOrchestrationPolicyAuditorRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine orchestration-policy-auditor", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository path to inspect")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runOrchestrationPolicyAuditorRoutine(*repo)
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

func runOrchestrationPolicyAuditorRoutine(repo string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "orchestration-policy-auditor",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Inspected repo-local orchestration docs, prompts, routine specs, and ledger/event JSON read-only",
		},
		NeedsHuman:          false,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked orchestration-policy-auditor precondition, then rerun the routine.",
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

	paths, err := orchestrationPolicyAuditPaths(repo)
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not collect orchestration policy audit paths: "+err.Error())
		report.BlockedReason = "could not collect orchestration policy audit inputs"
		return report
	}
	findings := []policyAuditFinding{}
	for _, path := range paths {
		data, readErr := os.ReadFile(filepath.Join(repo, path))
		if readErr != nil {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Could not read %s: %v", path, readErr))
			report.BlockedReason = "could not read orchestration policy audit input"
			return report
		}
		findings = append(findings, auditOrchestrationPolicyText(path, string(data))...)
	}
	report.ActionsTaken = append(report.ActionsTaken, "Applied deterministic local/static orchestration policy rules OPA001-OPA008")
	report.Evidence["local"] = append(report.Evidence["local"], fmt.Sprintf("Scanned %d repo-local orchestration policy input file(s).", len(paths)))
	report.Evidence["local"] = append(report.Evidence["local"], summarizePolicyAuditFindings(findings))

	if len(findings) > 0 {
		report.Status = "failed"
		report.Evidence["local"] = append(report.Evidence["local"], renderPolicyAuditFindings(findings)...)
		report.NextSuggestedAction = "Review these local/static orchestration policy suspicions, fix confirmed prompt/docs/routine issues, then rerun orchestration-policy-auditor."
		return report
	}

	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Record this local/static orchestration policy report if sufficient; no Codex App session or runtime proof was produced."
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
	report.ActionsTaken = append(report.ActionsTaken, "Parsed roadmap candidate sections from docs/roadmap.md")

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
		if roadmapCandidateMarkedCompleted(candidate.Name) {
			skipped = append(skipped, fmt.Sprintf("%s: skipped because the roadmap candidate text already marks it completed/done/covered.", candidate.Name))
			continue
		}
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

func cmdRunBudgetPolicyReportRoutine(args []string) error {
	fs := flag.NewFlagSet("run-routine budget-policy-report", flag.ExitOnError)
	repo := fs.String("repo", ".", "repository path to inspect")
	ledgerPath := fs.String("ledger", "", "optional ledger path; defaults to REPO/.codex-orchestrator/ledger.json when present")
	heartbeatReport := fs.String("heartbeat-report", "", "optional heartbeat report path; defaults to REPO/.codex-orchestrator/heartbeat-report.json when present")
	writeReport := fs.String("write-report", "", "write routine report JSON")
	jsonOut := fs.Bool("json", false, "print JSON report")
	if err := fs.Parse(args); err != nil {
		return err
	}
	report := runBudgetPolicyReportRoutine(*repo, *ledgerPath, *heartbeatReport)
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

func runBudgetPolicyReportRoutine(repo string, ledgerPath string, heartbeatReportPath string) RoutineRunReport {
	report := RoutineRunReport{
		RoutineID: "budget-policy-report",
		Status:    "blocked",
		Evidence: map[string][]string{
			"direct":  {},
			"proxy":   {},
			"local":   {},
			"blocked": {},
		},
		ActionsTaken: []string{
			"Inspected roadmap, routine docs, routine specs, optional ledger, and optional heartbeat report read-only",
			"Separated local/static budget metadata and heartbeat warnings from unknown live timing states",
			"Performed no scheduler, priority, worker-control, merge, push, delete, cleanup, or ledger mutation action",
		},
		NeedsHuman:          true,
		BlockedReason:       "",
		NextSuggestedAction: "Fix the blocked budget-policy-report precondition, then rerun the routine.",
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

	for _, doc := range []string{filepath.Join("docs", "roadmap.md"), filepath.Join("docs", "routines", "README.md")} {
		data, err := os.ReadFile(filepath.Join(repo, doc))
		if err != nil {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Could not read %s: %v", doc, err))
			report.BlockedReason = "could not inspect budget-policy docs"
			return report
		}
		issues := budgetPolicyBoundaryIssues(doc, string(data))
		if len(issues) > 0 {
			report.Status = "failed"
			report.Evidence["local"] = append(report.Evidence["local"], issues...)
			report.NextSuggestedAction = "Restore review-only budget-policy boundary wording in roadmap/routine docs, then rerun budget-policy-report."
			return report
		}
		report.Evidence["local"] = append(report.Evidence["local"], doc+" preserves budget-policy-report review-only boundary wording.")
	}

	coverage, err := inspectRoutineBudgetCoverage(filepath.Join(repo, "routines"))
	if err != nil {
		report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not inspect routines directory: "+err.Error())
		report.BlockedReason = "could not inspect routine specs"
		return report
	}
	report.Evidence["local"] = append(report.Evidence["local"], coverage.summary())
	if len(coverage.MissingMaxRuntime) > 0 || len(coverage.MissingReviewBudget) > 0 {
		report.Evidence["local"] = append(report.Evidence["local"], "Budget metadata gaps are local/static advisory warnings only; this routine did not enforce dispatch eligibility or worker limits.")
	}

	if resolvedLedger, ok := resolveOptionalLedgerPath(repo, ledgerPath); ok {
		ledger, err := loadLedger(resolvedLedger)
		if err != nil {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not inspect repo-local ledger "+resolvedLedger+": "+err.Error())
			report.BlockedReason = "could not inspect optional ledger"
			return report
		}
		summary := summarizeTaskBudgets(ledger.Tasks)
		report.Evidence["local"] = append(report.Evidence["local"], fmt.Sprintf(
			"Ledger budget metadata summary from %s: tasksWithBudget=%d tasksMissingBudget=%d totalMaxRuntimeMinutes=%d totalReviewBudgetMinutes=%d.",
			resolvedLedger,
			summary.TasksWithBudget,
			summary.TasksMissingBudget,
			summary.TotalMaxRuntimeMinutes,
			summary.TotalReviewBudgetMinutes,
		))
		report.Evidence["blocked"] = append(report.Evidence["blocked"], ledgerUnknownTimingEvidence(ledger.Tasks)...)
	} else {
		report.Evidence["local"] = append(report.Evidence["local"], "Repo-local ledger is absent; task budget metadata inspection skipped.")
	}

	if resolvedHeartbeat, ok := resolveOptionalHeartbeatReportPath(repo, heartbeatReportPath); ok {
		heartbeat, err := loadObserveSummary(resolvedHeartbeat)
		if err != nil {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], "Could not inspect heartbeat report "+resolvedHeartbeat+": "+err.Error())
			report.BlockedReason = "could not inspect optional heartbeat report"
			return report
		}
		report.Evidence["local"] = append(report.Evidence["local"], fmt.Sprintf(
			"Heartbeat budgetSummary from %s: tasksWithBudget=%d tasksMissingBudget=%d routineSpecsWithBudget=%d routineSpecsMissingBudget=%d.",
			resolvedHeartbeat,
			heartbeat.BudgetSummary.TasksWithBudget,
			heartbeat.BudgetSummary.TasksMissingBudget,
			heartbeat.BudgetSummary.RoutineSpecsWithBudget,
			heartbeat.BudgetSummary.RoutineSpecsMissingBudget,
		))
		report.Evidence["local"] = append(report.Evidence["local"], "Heartbeat budgetPressure evidenceLabel: "+emptyDefault(heartbeat.BudgetPressure.EvidenceLabel, "unknown/local-static-missing"))
		if len(heartbeat.BudgetPressure.Warnings) > 0 {
			report.Evidence["local"] = append(report.Evidence["local"], "Heartbeat budgetPressure warnings copied as local/static evidence: "+strings.Join(heartbeat.BudgetPressure.Warnings, " | "))
		} else {
			report.Evidence["local"] = append(report.Evidence["local"], "Heartbeat budgetPressure warnings: none recorded.")
		}
		if heartbeat.BudgetPressure.TasksWithUnknownReviewTime > 0 {
			report.Evidence["blocked"] = append(report.Evidence["blocked"], fmt.Sprintf("Heartbeat reports %d task(s) with unknown review elapsed time.", heartbeat.BudgetPressure.TasksWithUnknownReviewTime))
		}
	} else {
		report.Evidence["local"] = append(report.Evidence["local"], "Repo-local heartbeat report is absent; heartbeat budgetPressure copy skipped.")
	}

	report.Evidence["blocked"] = append(report.Evidence["blocked"], "Live Codex App session runtime, worker wall-clock state, and human review elapsed time were not available from direct runtime APIs; unknown live timing remains blocked/unknown.")
	report.Evidence["local"] = append(report.Evidence["local"], "No scheduler, priority engine, automatic killing, dispatch enforcement, merge, push, delete, cleanup, or worker-control action was performed.")
	report.Status = "passed"
	report.BlockedReason = ""
	report.NextSuggestedAction = "Review this local/static budget-policy report in Codex App or human review before changing concurrency, dispatch, pause, kill, merge, push, cleanup, or budget-enforcement behavior."
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
	observedAt := time.Now()
	observations := make([]Observation, 0, len(ledger.Tasks))
	for _, task := range ledger.Tasks {
		observations = append(observations, inspectTask(task, staleAfter))
	}
	integration := inspectIntegration(ledger.ProjectRoot)
	counts := countObservationStatuses(observations)
	pressure := calculateReviewPressure(counts, ledger.MaxConcurrency)
	budget := summarizeTaskBudgets(ledger.Tasks)
	addRoutineBudgetSummary(&budget, filepath.Join(expandPath(ledger.ProjectRoot), "routines"))
	budgetPressure := calculateBudgetPressure(ledger.Tasks, observations, observedAt, budget.RoutineSpecsMissingBudget)
	runtimeStatus := buildRuntimeStatusReport(ledger.Tasks, observations, pressure, observedAt)
	overall, actions := summarizeObservations(integration, counts, pressure)
	return ObserveSummary{
		Ledger:             ledgerPath,
		Version:            ledger.Version,
		ProjectRoot:        ledger.ProjectRoot,
		DefaultBranch:      ledger.DefaultBranch,
		ObservedAt:         observedAt.Format(time.RFC3339),
		OverallStatus:      overall,
		RecommendedActions: actions,
		Counts:             counts,
		ReviewPressure:     pressure,
		BudgetSummary:      budget,
		BudgetPressure:     budgetPressure,
		Integration:        integration,
		RuntimeStatus:      runtimeStatus,
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

func addRoutineBudgetSummary(summary *BudgetSummary, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var spec RoutineSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			continue
		}
		if spec.MaxRuntimeMinutes > 0 || spec.ReviewBudgetMinutes > 0 {
			summary.RoutineSpecsWithBudget++
		} else {
			summary.RoutineSpecsMissingBudget++
		}
	}
}

func calculateBudgetPressure(tasks []Task, observations []Observation, observedAt time.Time, routineSpecsMissingBudget int) BudgetPressureSummary {
	summary := BudgetPressureSummary{EvidenceLabel: "local/static"}
	for index := range tasks {
		pressure := taskBudgetPressure(tasks[index], observations[index].Status, observedAt)
		observations[index].BudgetPressure = pressure
		if pressure == nil {
			continue
		}
		switch pressure.Status {
		case "missing":
			summary.TasksMissingBudget++
		case "near-limit":
			summary.TasksNearLimit++
		case "exceeded":
			summary.TasksExceeded++
		case "unknown":
			if pressure.ReviewTimestampMissing {
				summary.TasksWithUnknownReviewTime++
			}
		}
		summary.Warnings = append(summary.Warnings, pressure.Warnings...)
	}
	if routineSpecsMissingBudget > 0 {
		summary.RoutineSpecsMissingBudget = routineSpecsMissingBudget
		summary.Warnings = append(summary.Warnings, fmt.Sprintf("%d routine spec(s) are missing local budget metadata.", routineSpecsMissingBudget))
	}
	return summary
}

func buildRuntimeStatusReport(tasks []Task, observations []Observation, pressure ReviewPressure, observedAt time.Time) RuntimeStatusReport {
	const recentWindowHours = 24
	recentAfter := observedAt.Add(-recentWindowHours * time.Hour)
	report := RuntimeStatusReport{
		EvidenceLabel:          "local/static",
		RecentWindowHours:      recentWindowHours,
		MaxConcurrency:         pressure.MaxConcurrency,
		UsedDispatchSlots:      pressure.MaxConcurrency - pressure.AvailableSlots,
		AvailableDispatchSlots: pressure.AvailableSlots,
	}
	for index, observation := range observations {
		task := tasks[index]
		item := runtimeStatusItem(task, observation)
		switch observation.Status {
		case "active":
			report.ActiveWorkers = append(report.ActiveWorkers, item)
		case "pending-setup":
			report.PendingSetup = append(report.PendingSetup, item)
		case "completed-unreviewed":
			report.CompletedUnreviewed = append(report.CompletedUnreviewed, item)
		case "blocked":
			report.Blockers = append(report.Blockers, item)
		case "cleanup-needed":
			report.CleanupNeeded = append(report.CleanupNeeded, item)
		case "stale-needs-inspection":
			if observation.Signal == "dirty-uncommitted" {
				report.DirtyUncommitted = append(report.DirtyUncommitted, item)
			} else {
				report.StaleNeedsInspection = append(report.StaleNeedsInspection, item)
			}
		}
		if recent, ok := recentMergedOrCleanedItem(task, observation, recentAfter); ok {
			report.RecentMergedOrCleaned = append(report.RecentMergedOrCleaned, recent)
		}
	}
	sortRuntimeStatusItems(report.ActiveWorkers)
	sortRuntimeStatusItems(report.PendingSetup)
	sortRuntimeStatusItems(report.DirtyUncommitted)
	sortRuntimeStatusItems(report.CompletedUnreviewed)
	sortRuntimeStatusItems(report.StaleNeedsInspection)
	sortRuntimeStatusItems(report.Blockers)
	sortRuntimeStatusItems(report.CleanupNeeded)
	sortRuntimeStatusItemsByLatest(report.RecentMergedOrCleaned)
	report.Summary = runtimeStatusSummary(report)
	return report
}

func runtimeStatusItem(task Task, observation Observation) RuntimeStatusItem {
	item := RuntimeStatusItem{
		ID:                task.ID,
		Title:             task.Title,
		LedgerStatus:      task.Status,
		ObservedStatus:    observation.Status,
		Signal:            observation.Signal,
		Branch:            task.Branch,
		Worktree:          task.Worktree,
		PendingWorktreeID: task.PendingWorktreeID,
		LastUpdatedAt:     observation.LastUpdatedAt,
		Action:            observation.Action,
		Note:              observation.Note,
		State:             observation.State,
	}
	if item.Title == item.ID {
		item.Title = ""
	}
	return item
}

func recentMergedOrCleanedItem(task Task, observation Observation, recentAfter time.Time) (RuntimeStatusItem, bool) {
	if observation.Status == "cleanup-needed" {
		return RuntimeStatusItem{}, false
	}
	event, ok := latestTaskEvent(task, func(event map[string]string) bool {
		status := event["status"]
		if status == "" {
			status = event["result"]
		}
		switch status {
		case "merged", "released", "cleaned":
			return true
		default:
			return false
		}
	})
	if !ok {
		return RuntimeStatusItem{}, false
	}
	at, err := time.Parse(time.RFC3339, event["at"])
	if err != nil || at.Before(recentAfter) {
		return RuntimeStatusItem{}, false
	}
	item := runtimeStatusItem(task, observation)
	item.ObservedStatus = event["status"]
	if item.ObservedStatus == "" {
		item.ObservedStatus = event["result"]
	}
	item.LastUpdatedAt = event["at"]
	if note := strings.TrimSpace(event["note"]); note != "" {
		item.Note = note
	}
	return item, true
}

func sortRuntimeStatusItems(items []RuntimeStatusItem) {
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
}

func sortRuntimeStatusItemsByLatest(items []RuntimeStatusItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].LastUpdatedAt == items[j].LastUpdatedAt {
			return items[i].ID < items[j].ID
		}
		return items[i].LastUpdatedAt > items[j].LastUpdatedAt
	})
}

func runtimeStatusSummary(report RuntimeStatusReport) string {
	parts := []string{
		fmt.Sprintf("active=%d", len(report.ActiveWorkers)),
		fmt.Sprintf("pending=%d", len(report.PendingSetup)),
		fmt.Sprintf("dirty=%d", len(report.DirtyUncommitted)),
		fmt.Sprintf("review=%d", len(report.CompletedUnreviewed)),
		fmt.Sprintf("blocked=%d", len(report.Blockers)),
		fmt.Sprintf("cleanup=%d", len(report.CleanupNeeded)),
		fmt.Sprintf("recentMergedOrCleaned=%d", len(report.RecentMergedOrCleaned)),
		fmt.Sprintf("availableSlots=%d", report.AvailableDispatchSlots),
	}
	if len(report.StaleNeedsInspection) > 0 {
		parts = append(parts, fmt.Sprintf("stale=%d", len(report.StaleNeedsInspection)))
	}
	return strings.Join(parts, " ")
}

func taskBudgetPressure(task Task, observedStatus string, observedAt time.Time) *BudgetPressure {
	if isTerminalStatus(observedStatus) {
		return nil
	}
	if task.Budget == nil {
		return &BudgetPressure{
			Status:        "missing",
			EvidenceLabel: "local/static",
			Warnings:      []string{fmt.Sprintf("Task %s is missing runtime/review budget metadata.", task.ID)},
		}
	}
	pressure := &BudgetPressure{
		Status:        "ok",
		EvidenceLabel: "local/static",
	}
	if task.Budget.MaxRuntimeMinutes > 0 {
		if startedAt, ok := taskRecordedAt(task); ok {
			elapsed := elapsedMinutes(startedAt, observedAt)
			pressure.RuntimeElapsedMinutes = elapsed
			pressure.RuntimeBudgetMinutes = task.Budget.MaxRuntimeMinutes
			addBudgetLimitWarning(pressure, task.ID, "runtime", elapsed, task.Budget.MaxRuntimeMinutes)
		}
	}
	if task.Budget.ReviewBudgetMinutes > 0 && observedStatus == "completed-unreviewed" {
		pressure.ReviewBudgetMinutes = task.Budget.ReviewBudgetMinutes
		if reviewAt, ok := taskReviewReadyAt(task); ok {
			elapsed := elapsedMinutes(reviewAt, observedAt)
			pressure.ReviewElapsedMinutes = elapsed
			addBudgetLimitWarning(pressure, task.ID, "review", elapsed, task.Budget.ReviewBudgetMinutes)
		} else {
			pressure.Status = worstBudgetStatus(pressure.Status, "unknown")
			pressure.ReviewTimestampMissing = true
			pressure.Warnings = append(pressure.Warnings, fmt.Sprintf("Task %s is review-ready but has no recorded review-ready timestamp; review budget elapsed time is unknown.", task.ID))
		}
	}
	if len(pressure.Warnings) == 0 {
		return nil
	}
	return pressure
}

func addBudgetLimitWarning(pressure *BudgetPressure, taskID string, kind string, elapsed int, limit int) {
	if limit <= 0 {
		return
	}
	switch {
	case elapsed >= limit:
		pressure.Status = worstBudgetStatus(pressure.Status, "exceeded")
		pressure.Warnings = append(pressure.Warnings, fmt.Sprintf("Task %s %s budget exceeded: %dm elapsed of %dm.", taskID, kind, elapsed, limit))
	case elapsed*100 >= limit*80:
		pressure.Status = worstBudgetStatus(pressure.Status, "near-limit")
		pressure.Warnings = append(pressure.Warnings, fmt.Sprintf("Task %s %s budget near limit: %dm elapsed of %dm.", taskID, kind, elapsed, limit))
	}
}

func worstBudgetStatus(current string, next string) string {
	rank := map[string]int{"ok": 0, "unknown": 1, "missing": 2, "near-limit": 3, "exceeded": 4}
	if rank[next] > rank[current] {
		return next
	}
	return current
}

func taskRecordedAt(task Task) (time.Time, bool) {
	return firstTaskTimestamp(task, func(event map[string]string) bool {
		return event["type"] == "record-task" || event["at"] != ""
	})
}

func taskReviewReadyAt(task Task) (time.Time, bool) {
	return firstTaskTimestamp(task, func(event map[string]string) bool {
		return event["status"] == "completed-unreviewed" || event["result"] == "completed-unreviewed"
	})
}

func firstTaskTimestamp(task Task, match func(map[string]string) bool) (time.Time, bool) {
	var best time.Time
	found := false
	events := append([]map[string]string{}, task.History...)
	if len(task.LastObservation) > 0 {
		events = append(events, task.LastObservation)
	}
	for _, event := range events {
		if !match(event) || event["at"] == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339, event["at"])
		if err != nil {
			continue
		}
		if !found || at.Before(best) {
			best = at
			found = true
		}
	}
	return best, found
}

func elapsedMinutes(start time.Time, end time.Time) int {
	if end.Before(start) {
		return 0
	}
	return int(end.Sub(start).Minutes())
}

func latestTaskEvent(task Task, match func(map[string]string) bool) (map[string]string, bool) {
	var best map[string]string
	var bestAt time.Time
	found := false
	events := append([]map[string]string{}, task.History...)
	if len(task.LastObservation) > 0 {
		events = append(events, task.LastObservation)
	}
	for _, event := range events {
		if !match(event) || event["at"] == "" {
			continue
		}
		at, err := time.Parse(time.RFC3339, event["at"])
		if err != nil {
			continue
		}
		if !found || at.After(bestAt) {
			bestAt = at
			best = event
			found = true
		}
	}
	return best, found
}

func taskLastUpdatedAt(task Task) string {
	event, ok := latestTaskEvent(task, func(event map[string]string) bool {
		return event["at"] != ""
	})
	if !ok {
		return ""
	}
	return event["at"]
}

func taskObservation(task Task, status string, action string, note string, gitStatus string, signal string) Observation {
	return Observation{
		ID:                task.ID,
		Status:            status,
		Action:            action,
		Note:              note,
		Signal:            signal,
		State:             localTaskState(task, status, signal, gitStatus),
		LedgerStatus:      task.Status,
		Branch:            task.Branch,
		Worktree:          task.Worktree,
		LastUpdatedAt:     taskLastUpdatedAt(task),
		GitStatus:         gitStatus,
		PendingWorktreeID: task.PendingWorktreeID,
		Budget:            task.Budget,
	}
}

func localTaskState(task Task, status string, signal string, gitStatus string) LocalTaskState {
	state := LocalTaskState{
		EvidenceLabel: "local/static",
		Lifecycle:     status,
		Setup:         "unknown",
		Worktree:      "unknown",
		Branch:        "unknown",
		Diff:          "not-inspected",
		Review:        "not-ready",
		Cleanup:       "not-needed",
	}
	if isTerminalStatus(status) {
		state.Setup = "complete"
		state.Worktree = "not-present-or-not-inspected"
		state.Branch = "not-inspected"
		state.Review = "terminal"
		state.Cleanup = "complete"
		switch status {
		case "merged":
			state.Cleanup = "unknown"
		case "released":
			state.Cleanup = "unknown"
		}
	}
	if task.Worktree == "" {
		state.Worktree = "not-recorded"
		if task.PendingWorktreeID != "" {
			state.Setup = "pending-worktree-id"
		} else {
			state.Setup = "missing-worktree-path"
		}
	} else {
		state.Worktree = "recorded"
	}
	if task.Branch == "" {
		state.Branch = "not-recorded"
	} else {
		state.Branch = "expected-recorded"
	}
	if gitStatus != "" {
		state.Setup = "worktree-present"
		state.Worktree = "present"
		state.Branch = "matched"
	}
	switch signal {
	case "pending-setup":
		state.Setup = "pending-worktree-id"
		state.Worktree = "not-recorded"
		state.Branch = "not-recorded"
	case "pending-setup-stale":
		state.Setup = "pending-worktree-id-stale"
		state.Worktree = "not-recorded"
		state.Branch = "not-recorded"
	case "missing-worktree":
		state.Setup = "worktree-missing"
		state.Worktree = "missing"
		state.Branch = "not-inspected"
	case "missing-worktree-stale":
		state.Setup = "worktree-missing-stale"
		state.Worktree = "missing"
		state.Branch = "not-inspected"
	case "missing-worktree-path":
		state.Setup = "missing-worktree-path"
		state.Worktree = "not-recorded"
		state.Branch = "not-inspected"
	case "git-status-error":
		state.Setup = "worktree-present"
		state.Worktree = "present"
		state.Branch = "not-inspected"
	case "branch-mismatch":
		state.Branch = "mismatch"
	case "branch-detached":
		state.Branch = "detached"
	case "dirty-uncommitted":
		state.Diff = "dirty-uncommitted"
	case "completed-clean-commit":
		state.Diff = "clean-task-commit"
		state.Review = "required"
	case "active-clean":
		state.Diff = "clean-no-task-commit"
	case "stale-active":
		state.Diff = "clean-no-task-commit-stale"
	case "stale-no-base-commit", "no-base-commit":
		state.Diff = "unknown-base"
	case "cleanup-pending":
		state.Cleanup = "needed"
		state.Review = "accepted"
	case "terminal-quiet":
		state.Setup = "complete"
		state.Worktree = "not-present-or-not-inspected"
		state.Branch = "not-inspected"
		state.Review = "terminal"
		state.Cleanup = "complete"
	}
	switch status {
	case "completed-unreviewed":
		state.Review = "required"
	case "cleanup-needed":
		state.Cleanup = "needed"
	case "blocked":
		if state.Review == "not-ready" {
			state.Review = "blocked"
		}
	case "merged", "released":
		if state.Cleanup == "unknown" {
			state.Cleanup = "not-present-or-not-inspected"
		}
	case "cleaned":
		state.Cleanup = "complete"
	}
	return state
}

func inspectTask(task Task, staleAfter time.Duration) Observation {
	if isTerminalStatus(task.Status) {
		if task.Worktree != "" {
			worktree := expandPath(task.Worktree)
			if _, err := os.Stat(worktree); err == nil && task.Status != "rejected" {
				return taskObservation(task, "cleanup-needed", "remove accepted task worktree and delete local task branch if safe", fmt.Sprintf("Task is %s but worktree still exists: %s", task.Status, worktree), "", "cleanup-pending")
			}
		}
		return taskObservation(task, task.Status, "quiet", fmt.Sprintf("Task is recorded as %s.", task.Status), "", "terminal-quiet")
	}
	if task.Worktree == "" {
		if task.PendingWorktreeID != "" {
			statusValue := "pending-setup"
			action := "wait for Codex App worktree setup to finish, then append worktree and branch"
			note := fmt.Sprintf("Pending worktree setup id recorded: %s", task.PendingWorktreeID)
			signal := "pending-setup"
			if isTaskStale(task, staleAfter) {
				statusValue = "stale-needs-inspection"
				action = "inspect pending setup and decide whether to re-dispatch or abandon"
				note = fmt.Sprintf("Pending worktree setup id %s is older than %s.", task.PendingWorktreeID, staleAfter)
				signal = "pending-setup-stale"
			}
			return taskObservation(task, statusValue, action, note, "", signal)
		}
		return taskObservation(task, "blocked", "record missing worktree path", "Task has no worktree path in ledger.", "", "missing-worktree-path")
	}
	worktree := expandPath(task.Worktree)
	if _, err := os.Stat(worktree); err != nil {
		statusValue := "pending-setup"
		action := "verify setup or mark stale if expired"
		note := fmt.Sprintf("Worktree does not exist: %s", worktree)
		signal := "missing-worktree"
		if isTaskStale(task, staleAfter) {
			statusValue = "stale-needs-inspection"
			action = "inspect pending setup and decide whether to re-dispatch or abandon"
			note = fmt.Sprintf("Worktree does not exist and the last observation is older than %s: %s", staleAfter, worktree)
			signal = "missing-worktree-stale"
		}
		return taskObservation(task, statusValue, action, note, "", signal)
	}
	status, err := gitOutput(worktree, "status", "--short", "--branch")
	if err != nil {
		return taskObservation(task, "blocked", "inspect worktree git state", err.Error(), "", "git-status-error")
	}
	branch := currentBranch(status)
	if task.Branch != "" && branch != "" && branch != task.Branch {
		return taskObservation(task, "blocked", "fix branch mismatch before review", fmt.Sprintf("Expected %s, found %s.", task.Branch, branch), status, "branch-mismatch")
	}
	if task.Branch != "" && branch == "" {
		return taskObservation(task, "blocked", "reattach worker worktree to the recorded task branch before review", fmt.Sprintf("Expected %s, but git status did not report an attached branch.", task.Branch), status, "branch-detached")
	}
	if hasDirtyChanges(status) {
		return taskObservation(task, "stale-needs-inspection", "inspect uncommitted scoped diff or nudge same worker", "Worktree has uncommitted changes.", status, "dirty-uncommitted")
	}
	commitsAfterBase, known := hasCommitsAfterBase(worktree, task.BaseCommit)
	if known && commitsAfterBase {
		return taskObservation(task, "completed-unreviewed", "orchestrator review required before merge", "Clean worktree has commits after baseCommit.", status, "completed-clean-commit")
	}
	if !known {
		statusValue := task.Status
		if statusValue == "" {
			statusValue = "active"
		}
		if statusValue == "active" && isTaskStale(task, staleAfter) {
			return taskObservation(task, "stale-needs-inspection", "inspect manually", fmt.Sprintf("Task has no comparable baseCommit and the last observation is older than %s.", staleAfter), status, "stale-no-base-commit")
		}
		return taskObservation(task, statusValue, "inspect manually", "Could not compare baseCommit; ledger may be a template or base is missing.", status, "no-base-commit")
	}
	if task.Status == "active" && isTaskStale(task, staleAfter) {
		return taskObservation(task, "stale-needs-inspection", "inspect recent thread messages or nudge same worker", fmt.Sprintf("Clean worktree has no commits after baseCommit, and last observation is older than %s.", staleAfter), status, "stale-active")
	}
	return taskObservation(task, "active", "quiet", "Clean worktree has no commits after baseCommit.", status, "active-clean")
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
	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	tmp, err := os.CreateTemp(dir, "."+filepath.Base(target)+".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(0o644); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, target); err != nil {
		return err
	}
	cleanup = false
	return nil
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
	fmt.Fprintf(&b, "- runtimeStatus: `%s`\n", summary.RuntimeStatus.Summary)
	fmt.Fprintf(&b, "- tasksWithBudget: `%d`\n", summary.BudgetSummary.TasksWithBudget)
	if summary.BudgetSummary.TotalMaxRuntimeMinutes > 0 {
		fmt.Fprintf(&b, "- totalMaxRuntimeMinutes: `%d`\n", summary.BudgetSummary.TotalMaxRuntimeMinutes)
	}
	if summary.BudgetSummary.TotalReviewBudgetMinutes > 0 {
		fmt.Fprintf(&b, "- totalReviewBudgetMinutes: `%d`\n", summary.BudgetSummary.TotalReviewBudgetMinutes)
	}
	if summary.BudgetSummary.RoutineSpecsWithBudget > 0 || summary.BudgetSummary.RoutineSpecsMissingBudget > 0 {
		fmt.Fprintf(&b, "- routineSpecsWithBudget: `%d`\n", summary.BudgetSummary.RoutineSpecsWithBudget)
		fmt.Fprintf(&b, "- routineSpecsMissingBudget: `%d`\n", summary.BudgetSummary.RoutineSpecsMissingBudget)
	}
	if len(summary.BudgetPressure.Warnings) > 0 {
		fmt.Fprintf(&b, "- budgetPressure: `%s`\n", summary.BudgetPressure.EvidenceLabel)
	}
	if summary.Integration.Error != "" {
		fmt.Fprintf(&b, "- integrationError: `%s`\n", summary.Integration.Error)
	}
	if len(summary.BudgetPressure.Warnings) > 0 {
		fmt.Fprintf(&b, "\n## Budget Pressure\n\n")
		for _, warning := range summary.BudgetPressure.Warnings {
			fmt.Fprintf(&b, "- %s\n", warning)
		}
	}
	if len(summary.RecommendedActions) > 0 {
		fmt.Fprintf(&b, "\n## Recommended Actions\n\n")
		for _, action := range summary.RecommendedActions {
			fmt.Fprintf(&b, "- %s\n", action)
		}
	}
	renderRuntimeStatusMarkdown(&b, summary.RuntimeStatus)
	fmt.Fprintf(&b, "\n## Tasks\n\n")
	if len(summary.Observations) == 0 {
		fmt.Fprintf(&b, "- No tasks recorded.\n")
	} else {
		for _, item := range summary.Observations {
			fmt.Fprintf(&b, "- `%s`: `%s` - %s\n", item.ID, item.Status, item.Action)
			if item.Note != "" {
				fmt.Fprintf(&b, "  - note: %s\n", item.Note)
			}
			if item.PendingWorktreeID != "" {
				fmt.Fprintf(&b, "  - pendingWorktreeId: `%s`\n", item.PendingWorktreeID)
			}
			if state := formatLocalTaskState(item.State); state != "" {
				fmt.Fprintf(&b, "  - state: %s\n", state)
			}
			if item.Branch != "" {
				fmt.Fprintf(&b, "  - branch: `%s`\n", item.Branch)
			}
			if item.Worktree != "" {
				fmt.Fprintf(&b, "  - worktree: `%s`\n", item.Worktree)
			}
			if item.LastUpdatedAt != "" {
				fmt.Fprintf(&b, "  - lastUpdatedAt: `%s`\n", item.LastUpdatedAt)
			}
			if budget := formatBudget(item.Budget); budget != "" {
				fmt.Fprintf(&b, "  - budget: %s\n", budget)
			}
			if item.BudgetPressure != nil {
				fmt.Fprintf(&b, "  - budgetPressure: %s", item.BudgetPressure.Status)
				if item.BudgetPressure.RuntimeBudgetMinutes > 0 {
					fmt.Fprintf(&b, " runtime=%dm/%dm", item.BudgetPressure.RuntimeElapsedMinutes, item.BudgetPressure.RuntimeBudgetMinutes)
				}
				if item.BudgetPressure.ReviewBudgetMinutes > 0 {
					fmt.Fprintf(&b, " review=%dm/%dm", item.BudgetPressure.ReviewElapsedMinutes, item.BudgetPressure.ReviewBudgetMinutes)
				}
				fmt.Fprintf(&b, " evidence=%s\n", item.BudgetPressure.EvidenceLabel)
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

func renderRuntimeStatusMarkdown(b *strings.Builder, report RuntimeStatusReport) {
	fmt.Fprintf(b, "\n## Runtime Status\n\n")
	fmt.Fprintf(b, "- evidenceLabel: `%s`\n", report.EvidenceLabel)
	fmt.Fprintf(b, "- summary: `%s`\n", report.Summary)
	fmt.Fprintf(b, "- dispatchSlots: `%d/%d` used, `%d` available\n", report.UsedDispatchSlots, report.MaxConcurrency, report.AvailableDispatchSlots)
	renderRuntimeStatusCategoryMarkdown(b, "Active Workers", report.ActiveWorkers)
	renderRuntimeStatusCategoryMarkdown(b, "Pending Setup", report.PendingSetup)
	renderRuntimeStatusCategoryMarkdown(b, "Dirty Uncommitted", report.DirtyUncommitted)
	renderRuntimeStatusCategoryMarkdown(b, "Completed Unreviewed", report.CompletedUnreviewed)
	renderRuntimeStatusCategoryMarkdown(b, "Blockers", report.Blockers)
	renderRuntimeStatusCategoryMarkdown(b, "Cleanup Needed", report.CleanupNeeded)
	renderRuntimeStatusCategoryMarkdown(b, fmt.Sprintf("Recent Merged Or Cleaned (last %dh)", report.RecentWindowHours), report.RecentMergedOrCleaned)
	renderRuntimeStatusCategoryMarkdown(b, "Stale Needs Inspection", report.StaleNeedsInspection)
}

func renderRuntimeStatusCategoryMarkdown(b *strings.Builder, title string, items []RuntimeStatusItem) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "\n### %s\n\n", title)
	for _, item := range items {
		fmt.Fprintf(b, "- `%s`: `%s` - %s\n", item.ID, item.ObservedStatus, item.Action)
		if item.Note != "" {
			fmt.Fprintf(b, "  - note: %s\n", item.Note)
		}
		if item.Branch != "" {
			fmt.Fprintf(b, "  - branch: `%s`\n", item.Branch)
		}
		if item.Worktree != "" {
			fmt.Fprintf(b, "  - worktree: `%s`\n", item.Worktree)
		}
		if item.PendingWorktreeID != "" {
			fmt.Fprintf(b, "  - pendingWorktreeId: `%s`\n", item.PendingWorktreeID)
		}
		if state := formatLocalTaskState(item.State); state != "" {
			fmt.Fprintf(b, "  - state: %s\n", state)
		}
		if item.LastUpdatedAt != "" {
			fmt.Fprintf(b, "  - lastUpdatedAt: `%s`\n", item.LastUpdatedAt)
		}
	}
}

func printRuntimeStatusReport(report RuntimeStatusReport) {
	printRuntimeStatusCategory("Active workers", report.ActiveWorkers)
	printRuntimeStatusCategory("Pending setup", report.PendingSetup)
	printRuntimeStatusCategory("Dirty uncommitted", report.DirtyUncommitted)
	printRuntimeStatusCategory("Completed unreviewed", report.CompletedUnreviewed)
	printRuntimeStatusCategory("Blockers", report.Blockers)
	printRuntimeStatusCategory("Cleanup needed", report.CleanupNeeded)
	printRuntimeStatusCategory(fmt.Sprintf("Recent merged or cleaned (last %dh)", report.RecentWindowHours), report.RecentMergedOrCleaned)
	printRuntimeStatusCategory("Stale needs inspection", report.StaleNeedsInspection)
}

func printRuntimeStatusCategory(title string, items []RuntimeStatusItem) {
	if len(items) == 0 {
		return
	}
	fmt.Printf("%s:\n", title)
	for _, item := range items {
		fmt.Printf("- %s: %s", item.ID, item.ObservedStatus)
		if item.Branch != "" {
			fmt.Printf(" branch=%s", item.Branch)
		}
		if item.PendingWorktreeID != "" {
			fmt.Printf(" pendingWorktreeId=%s", item.PendingWorktreeID)
		}
		if state := formatLocalTaskState(item.State); state != "" {
			fmt.Printf(" state={%s}", state)
		}
		if item.LastUpdatedAt != "" {
			fmt.Printf(" updated=%s", item.LastUpdatedAt)
		}
		fmt.Println()
		if item.Note != "" {
			fmt.Printf("  note: %s\n", item.Note)
		}
		if item.Action != "" {
			fmt.Printf("  action: %s\n", item.Action)
		}
		if item.Worktree != "" {
			fmt.Printf("  worktree: %s\n", item.Worktree)
		}
	}
}

func compactEvent(event map[string]any) map[string]string {
	result := map[string]string{}
	for _, key := range []string{"at", "type", "status", "taskId", "pendingWorktreeId", "worktree", "branch", "note"} {
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

func formatLocalTaskState(state LocalTaskState) string {
	parts := []string{}
	if state.Lifecycle != "" {
		parts = append(parts, "lifecycle="+state.Lifecycle)
	}
	if state.Setup != "" {
		parts = append(parts, "setup="+state.Setup)
	}
	if state.Worktree != "" {
		parts = append(parts, "worktree="+state.Worktree)
	}
	if state.Branch != "" {
		parts = append(parts, "branch="+state.Branch)
	}
	if state.Diff != "" {
		parts = append(parts, "diff="+state.Diff)
	}
	if state.Review != "" {
		parts = append(parts, "review="+state.Review)
	}
	if state.Cleanup != "" {
		parts = append(parts, "cleanup="+state.Cleanup)
	}
	if state.EvidenceLabel != "" {
		parts = append(parts, "evidence="+state.EvidenceLabel)
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
	fmt.Printf("Runtime status (%s): %s\n", summary.RuntimeStatus.EvidenceLabel, summary.RuntimeStatus.Summary)
	fmt.Printf("Dispatch slots: used=%d/%d available=%d\n",
		summary.RuntimeStatus.UsedDispatchSlots,
		summary.RuntimeStatus.MaxConcurrency,
		summary.RuntimeStatus.AvailableDispatchSlots,
	)
	fmt.Printf("Budget: tasksWithBudget=%d totalMaxRuntimeMinutes=%d totalReviewBudgetMinutes=%d\n",
		summary.BudgetSummary.TasksWithBudget,
		summary.BudgetSummary.TotalMaxRuntimeMinutes,
		summary.BudgetSummary.TotalReviewBudgetMinutes,
	)
	if summary.BudgetSummary.RoutineSpecsWithBudget > 0 || summary.BudgetSummary.RoutineSpecsMissingBudget > 0 {
		fmt.Printf("Routine budgets: specsWithBudget=%d specsMissingBudget=%d\n",
			summary.BudgetSummary.RoutineSpecsWithBudget,
			summary.BudgetSummary.RoutineSpecsMissingBudget,
		)
	}
	for _, warning := range summary.BudgetPressure.Warnings {
		fmt.Printf("Budget pressure (%s): %s\n", summary.BudgetPressure.EvidenceLabel, warning)
	}
	if summary.Integration.Error != "" {
		fmt.Printf("Integration: blocked (%s)\n", summary.Integration.Error)
	} else {
		fmt.Printf("Integration dirty: %t\n", summary.Integration.Dirty)
	}
	printRuntimeStatusReport(summary.RuntimeStatus)
	for _, action := range summary.RecommendedActions {
		fmt.Printf("Action: %s\n", action)
	}
	for _, item := range summary.Observations {
		fmt.Println()
		fmt.Printf("- %s: %s\n", item.ID, item.Status)
		fmt.Printf("  action: %s\n", item.Action)
		fmt.Printf("  note: %s\n", item.Note)
		if item.PendingWorktreeID != "" {
			fmt.Printf("  pendingWorktreeId: %s\n", item.PendingWorktreeID)
		}
		if item.Branch != "" {
			fmt.Printf("  branch: %s\n", item.Branch)
		}
		if item.Worktree != "" {
			fmt.Printf("  worktree: %s\n", item.Worktree)
		}
		if item.LastUpdatedAt != "" {
			fmt.Printf("  lastUpdatedAt: %s\n", item.LastUpdatedAt)
		}
		if budget := formatBudget(item.Budget); budget != "" {
			fmt.Printf("  budget: %s\n", budget)
		}
		if item.BudgetPressure != nil {
			fmt.Printf("  budgetPressure: %s evidence=%s\n", item.BudgetPressure.Status, item.BudgetPressure.EvidenceLabel)
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
	collectNextPriorities := false
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
			collectNextPriorities = strings.Contains(currentSection, "下一阶段优先级")
			continue
		case strings.HasPrefix(trimmed, "### "):
			collectV3Candidates = false
			collectRemaining = false
			collectNextPriorities = false
			continue
		case inV3 && strings.HasPrefix(trimmed, "候选 routine"):
			collectV3Candidates = true
			collectRemaining = false
			continue
		case trimmed == "剩余：":
			collectRemaining = true
			collectV3Candidates = false
			collectNextPriorities = false
			continue
		case collectNextPriorities && strings.HasPrefix(trimmed, "暂不进入"):
			collectNextPriorities = false
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
		if collectNextPriorities {
			switch {
			case hasNumberedListPrefix(trimmed):
				addCandidate(trimmed[strings.Index(trimmed, ".")+1:], index+1, "next-priority")
				continue
			case trimmed == "" || strings.HasPrefix(trimmed, "- "):
				continue
			}
		}
	}
	if len(candidates) == 0 {
		return nil, errors.New("no v3 candidates, explicit remaining tasks, or next-stage priorities found")
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
	case "next-priority":
		return 2
	default:
		return 3
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

func roadmapCandidateMarkedCompleted(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return false
	}
	statusParts := roadmapCandidateStatusParts(value)
	for _, part := range statusParts {
		if roadmapTextSaysCompleted(part) {
			return true
		}
	}
	return roadmapTextSaysCompleted(value)
}

func roadmapCandidateStatusParts(value string) []string {
	parts := []string{}
	for _, sep := range []string{"：", ":", " - ", " -- ", " — ", " – "} {
		if index := strings.Index(value, sep); index >= 0 {
			status := strings.TrimSpace(value[index+len(sep):])
			if status != "" {
				parts = append(parts, status)
			}
		}
	}
	for _, pair := range [][2]string{{"（", "）"}, {"(", ")"}} {
		start := strings.Index(value, pair[0])
		end := strings.LastIndex(value, pair[1])
		if start >= 0 && end > start {
			status := strings.TrimSpace(value[start+len(pair[0]) : end])
			if status != "" {
				parts = append(parts, status)
			}
		}
	}
	return parts
}

func roadmapTextSaysCompleted(value string) bool {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return false
	}
	if containsAnyFold(value, []string{
		"已完成",
		"已经完成",
		"已补",
		"已覆盖",
		"已经覆盖",
		"已具备",
		"已经具备",
		"already completed",
		"already done",
		"already covered",
		"already implemented",
		"has been completed",
		"have been completed",
		"was completed",
		"were completed",
		"is completed",
		"is complete",
		"now complete",
		"completed already",
		"done already",
		"covered already",
	}) {
		return true
	}
	for _, suffix := range []string{" completed", " done", " covered", " implemented", " shipped"} {
		if strings.HasSuffix(value, suffix) {
			return true
		}
	}
	return false
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

func resolveOptionalHeartbeatReportPath(repo string, explicit string) (string, bool) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, true
	}
	defaultPath := filepath.Join(repo, defaultStateDir, "heartbeat-report.json")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath, true
	}
	return "", false
}

func loadObserveSummary(path string) (ObserveSummary, error) {
	var summary ObserveSummary
	data, err := os.ReadFile(expandPath(path))
	if err != nil {
		return summary, err
	}
	if err := json.Unmarshal(data, &summary); err != nil {
		return summary, err
	}
	return summary, nil
}

func inspectRoutineBudgetCoverage(dir string) (routineBudgetCoverage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return routineBudgetCoverage{}, err
	}
	coverage := routineBudgetCoverage{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return coverage, err
		}
		var spec RoutineSpec
		if err := json.Unmarshal(data, &spec); err != nil {
			return coverage, fmt.Errorf("%s: %w", entry.Name(), err)
		}
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			id = strings.TrimSuffix(entry.Name(), ".json")
		}
		coverage.Total++
		if spec.MaxRuntimeMinutes > 0 {
			coverage.WithMaxRuntime++
		} else {
			coverage.MissingMaxRuntime = append(coverage.MissingMaxRuntime, id)
		}
		if spec.ReviewBudgetMinutes > 0 {
			coverage.WithReviewBudget++
		} else {
			coverage.MissingReviewBudget = append(coverage.MissingReviewBudget, id)
		}
		if spec.MaxRuntimeMinutes > 0 && spec.ReviewBudgetMinutes > 0 {
			coverage.WithBoth++
		}
		if spec.MaxRuntimeMinutes > 0 || spec.ReviewBudgetMinutes > 0 {
			coverage.WithAnyBudget++
		} else {
			coverage.WithoutAnyBudget = append(coverage.WithoutAnyBudget, id)
		}
	}
	sort.Strings(coverage.MissingMaxRuntime)
	sort.Strings(coverage.MissingReviewBudget)
	sort.Strings(coverage.WithoutAnyBudget)
	if coverage.Total == 0 {
		return coverage, errors.New("no routine spec JSON files found")
	}
	return coverage, nil
}

func (coverage routineBudgetCoverage) summary() string {
	parts := []string{
		fmt.Sprintf("Routine budget metadata coverage: total=%d withMaxRuntime=%d withReviewBudget=%d withBoth=%d withAnyBudget=%d withoutAnyBudget=%d.",
			coverage.Total,
			coverage.WithMaxRuntime,
			coverage.WithReviewBudget,
			coverage.WithBoth,
			coverage.WithAnyBudget,
			len(coverage.WithoutAnyBudget),
		),
	}
	if len(coverage.MissingMaxRuntime) > 0 {
		parts = append(parts, "missing maxRuntimeMinutes: "+formatStringList(coverage.MissingMaxRuntime)+".")
	}
	if len(coverage.MissingReviewBudget) > 0 {
		parts = append(parts, "missing reviewBudgetMinutes: "+formatStringList(coverage.MissingReviewBudget)+".")
	}
	return strings.Join(parts, " ")
}

func budgetPolicyBoundaryIssues(path string, text string) []string {
	lower := strings.ToLower(text)
	issues := []string{}
	if !strings.Contains(lower, "budget-policy-report") {
		issues = append(issues, path+" is missing budget-policy-report boundary wording.")
	}
	if !strings.Contains(lower, "review-only") && !strings.Contains(text, "只读") {
		issues = append(issues, path+" is missing review-only/read-only budget-policy wording.")
	}
	for _, phrase := range []string{"kill workers", "automatic killing", "dispatch enforcement", "make dispatch eligibility decisions"} {
		if strings.Contains(lower, phrase) && !containsNearbyNegation(lower, phrase) {
			issues = append(issues, fmt.Sprintf("%s contains budget control wording without nearby negation: %q.", path, phrase))
		}
	}
	sort.Strings(issues)
	return issues
}

func containsNearbyNegation(text string, phrase string) bool {
	index := strings.Index(text, phrase)
	if index < 0 {
		return false
	}
	start := index - 80
	if start < 0 {
		start = 0
	}
	window := text[start : index+len(phrase)]
	for _, negation := range []string{"not ", "no ", "does not ", "must not ", "without ", "不会", "不能", "不得", "不要", "不做", "不引入"} {
		if strings.Contains(window, negation) {
			return true
		}
	}
	return false
}

func ledgerUnknownTimingEvidence(tasks []Task) []string {
	evidence := []string{}
	for _, task := range tasks {
		if task.Budget == nil {
			continue
		}
		if task.Budget.MaxRuntimeMinutes > 0 {
			if _, ok := taskRecordedAt(task); !ok {
				evidence = append(evidence, fmt.Sprintf("Task %s has a runtime budget but no usable ledger start timestamp; runtime elapsed time is unknown.", task.ID))
			}
		}
		if task.Budget.ReviewBudgetMinutes > 0 && task.Status == "completed-unreviewed" {
			if _, ok := taskReviewReadyAt(task); !ok {
				evidence = append(evidence, fmt.Sprintf("Task %s is review-ready but has no recorded review-ready timestamp; human review elapsed time is unknown.", task.ID))
			}
		}
	}
	sort.Strings(evidence)
	return evidence
}

func emptyDefault(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func resolveDefaultLedgerPath(repo string, ledgerPath string, ledgerExplicit bool) string {
	if ledgerExplicit {
		return ledgerPath
	}
	repo = expandPath(repo)
	if strings.TrimSpace(repo) == "" {
		repo = "."
	}
	return filepath.Join(repo, defaultLedger)
}

func flagProvided(fs *flag.FlagSet, name string) bool {
	provided := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			provided = true
		}
	})
	return provided
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
	case strings.Contains(key, "runtimestatusreport"),
		strings.Contains(key, "statusreport"),
		strings.Contains(key, "statemodel"),
		strings.Contains(key, "worktreestatemodel"),
		strings.Contains(key, "reviewchecklist"),
		strings.Contains(key, "evidencelabellinter"),
		strings.Contains(key, "docsdriftguard"),
		strings.Contains(key, "casestudy"),
		strings.Contains(key, "bootstrapdocs"):
		return true, "it can stay bounded to repo-local docs/spec/ledger analysis without merge, push, release, or session mutation"
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

type policyAuditFinding struct {
	RuleID  string
	Message string
}

type policyEvalFixture struct {
	SchemaVersion    int               `json:"schemaVersion"`
	ID               string            `json:"id"`
	Description      string            `json:"description,omitempty"`
	Files            map[string]string `json:"files"`
	ExpectedRuleHits map[string]int    `json:"expectedRuleHits"`
}

type policyEvalResult struct {
	Passed        bool
	Evidence      []string
	Blocked       []string
	BlockedReason string
}

type evalAddFailureResult struct {
	Suite            string         `json:"suite"`
	ID               string         `json:"id"`
	Path             string         `json:"path"`
	File             string         `json:"file"`
	ExpectedRuleHits map[string]int `json:"expectedRuleHits"`
	ActualRuleHits   map[string]int `json:"actualRuleHits"`
	Overwritten      bool           `json:"overwritten"`
}

type RuleProposalReport struct {
	SchemaVersion    int            `json:"schemaVersion"`
	Command          string         `json:"command"`
	GeneratedAt      string         `json:"generatedAt"`
	Status           string         `json:"status"`
	Source           RuleSource     `json:"source"`
	EvidenceLabel    string         `json:"evidenceLabel"`
	NeedsHumanReview bool           `json:"needsHumanReview"`
	Proposals        []RuleProposal `json:"proposals"`
	Evidence         []string       `json:"evidence"`
	BlockedReason    string         `json:"blockedReason,omitempty"`
	NextAction       string         `json:"nextAction"`
}

type RuleSource struct {
	Kind string `json:"kind"`
	Path string `json:"path,omitempty"`
}

type RuleProposal struct {
	Title            string   `json:"title"`
	Body             string   `json:"body"`
	Source           string   `json:"source"`
	EvidenceLabel    string   `json:"evidenceLabel"`
	NeedsHumanReview bool     `json:"needsHumanReview"`
	RuleIDs          []string `json:"ruleIds,omitempty"`
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
	evidenceRuleTextPromotionTarget  = "ELA010"

	policyRuleDryRunBarrier     = "OPA001"
	policyRuleMainFallbackGuard = "OPA002"
	policyRuleContinuationGuard = "OPA003"
	policyRuleWorkerBoundary    = "OPA004"
	policyRuleEvidenceBoundary  = "OPA005"
	policyRuleHeartbeatBinding  = "OPA006"
	policyRulePendingLedger     = "OPA007"
	policyRuleBudgetBoundary    = "OPA008"
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

func newPolicyAuditFinding(ruleID string, format string, args ...any) policyAuditFinding {
	return policyAuditFinding{
		RuleID:  ruleID,
		Message: fmt.Sprintf(format, args...),
	}
}

func renderPolicyAuditFindings(findings []policyAuditFinding) []string {
	sorted := append([]policyAuditFinding{}, findings...)
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

func countPolicyAuditFindings(findings []policyAuditFinding) map[string]int {
	counts := map[string]int{}
	for _, finding := range findings {
		counts[finding.RuleID]++
	}
	return counts
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

func formatRuleCounts(counts map[string]int) string {
	if len(counts) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(counts))
	for key := range counts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%d", key, counts[key]))
	}
	return strings.Join(parts, ", ")
}

func runOrchestrationPolicyEvalFixtures(repo string, evalDir string) policyEvalResult {
	result := policyEvalResult{
		Passed:   true,
		Evidence: []string{},
		Blocked:  []string{},
	}
	fixtureDir := resolveRepoRelativePath(repo, evalDir)
	entries, err := os.ReadDir(fixtureDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			result.Evidence = append(result.Evidence, "Policy eval fixtures not found; skipped: "+fixtureDir)
			return result
		}
		result.Passed = false
		result.Blocked = append(result.Blocked, "Could not read policy eval fixtures: "+err.Error())
		result.BlockedReason = "could not read policy eval fixtures"
		return result
	}
	checked := 0
	failures := []string{}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		checked++
		path := filepath.Join(fixtureDir, entry.Name())
		fixture, loadErr := loadPolicyEvalFixture(path)
		if loadErr != nil {
			result.Passed = false
			result.Blocked = append(result.Blocked, "Could not load policy eval fixture "+path+": "+loadErr.Error())
			result.BlockedReason = "could not load policy eval fixture"
			return result
		}
		findings := []policyAuditFinding{}
		fileNames := make([]string, 0, len(fixture.Files))
		for name := range fixture.Files {
			fileNames = append(fileNames, name)
		}
		sort.Strings(fileNames)
		for _, name := range fileNames {
			findings = append(findings, auditOrchestrationPolicyText(name, fixture.Files[name])...)
		}
		actual := countPolicyAuditFindings(findings)
		if !sameRuleCounts(actual, fixture.ExpectedRuleHits) {
			result.Passed = false
			failures = append(failures, fmt.Sprintf(
				"%s: expected %s, got %s.",
				fixture.ID,
				formatRuleCounts(fixture.ExpectedRuleHits),
				formatRuleCounts(actual),
			))
		}
	}
	if checked == 0 {
		result.Passed = false
		result.Blocked = append(result.Blocked, "Policy eval fixture directory has no JSON fixtures: "+fixtureDir)
		result.BlockedReason = "policy eval fixtures are missing"
		return result
	}
	result.Evidence = append(result.Evidence, fmt.Sprintf("Ran %d orchestration policy eval fixture(s) from %s.", checked, fixtureDir))
	if len(failures) > 0 {
		sort.Strings(failures)
		result.Evidence = append(result.Evidence, failures...)
	} else {
		result.Evidence = append(result.Evidence, "Policy eval fixtures: passed.")
	}
	return result
}

func addFailureEvalFixture(repo string, suite string, evalDir string, id string, description string, fileName string, text string, textFile string, expects []string, force bool) (evalAddFailureResult, error) {
	result := evalAddFailureResult{}
	repo = expandPath(repo)
	if repo == "" {
		repo = "."
	}
	suite = strings.TrimSpace(suite)
	if suite == "" {
		suite = "orchestration-policy-auditor"
	}
	if suite != "orchestration-policy-auditor" {
		return result, fmt.Errorf("unsupported eval suite %q", suite)
	}
	if evalDir == "" {
		evalDir = filepath.Join("eval", suite)
	}
	id = strings.TrimSpace(id)
	if err := validateFixtureID(id); err != nil {
		return result, err
	}
	fileName = strings.TrimSpace(fileName)
	if fileName == "" {
		return result, errors.New("eval add-failure requires --file")
	}
	if filepath.IsAbs(fileName) || strings.Contains(fileName, "..") {
		return result, errors.New("eval add-failure --file must be a relative fixture file path without '..'")
	}
	body, err := failureText(text, textFile)
	if err != nil {
		return result, err
	}
	expected, err := parseRuleExpectations(expects)
	if err != nil {
		return result, err
	}
	if len(expected) == 0 {
		return result, errors.New("eval add-failure requires at least one --expect RULE=N")
	}
	fixture := policyEvalFixture{
		SchemaVersion:    1,
		ID:               id,
		Description:      strings.TrimSpace(description),
		Files:            map[string]string{fileName: body},
		ExpectedRuleHits: expected,
	}
	findings := auditOrchestrationPolicyText(fileName, body)
	actual := countPolicyAuditFindings(findings)
	if !sameRuleCounts(actual, expected) {
		return result, fmt.Errorf("fixture %s expected %s, but text produced %s", id, formatRuleCounts(expected), formatRuleCounts(actual))
	}
	fixtureDir := resolveRepoRelativePath(repo, evalDir)
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		return result, err
	}
	path := filepath.Join(fixtureDir, id+".json")
	_, statErr := os.Stat(path)
	exists := statErr == nil
	if exists && !force {
		return result, fmt.Errorf("eval fixture already exists: %s; pass --force to overwrite", path)
	}
	if statErr != nil && !errors.Is(statErr, os.ErrNotExist) {
		return result, statErr
	}
	data, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		return result, err
	}
	data = append(data, '\n')
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return result, err
	}
	return evalAddFailureResult{
		Suite:            suite,
		ID:               id,
		Path:             path,
		File:             fileName,
		ExpectedRuleHits: expected,
		ActualRuleHits:   actual,
		Overwritten:      exists,
	}, nil
}

func loadPolicyEvalFixture(path string) (policyEvalFixture, error) {
	var fixture policyEvalFixture
	data, err := os.ReadFile(path)
	if err != nil {
		return fixture, err
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		return fixture, err
	}
	if fixture.SchemaVersion != 1 {
		return fixture, fmt.Errorf("unsupported schemaVersion %d", fixture.SchemaVersion)
	}
	if strings.TrimSpace(fixture.ID) == "" {
		return fixture, errors.New("fixture id is required")
	}
	if len(fixture.Files) == 0 {
		return fixture, errors.New("fixture files are required")
	}
	if fixture.ExpectedRuleHits == nil {
		fixture.ExpectedRuleHits = map[string]int{}
	}
	return fixture, nil
}

func validateFixtureID(id string) error {
	if id == "" {
		return errors.New("eval add-failure requires --id")
	}
	for _, r := range id {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_':
		default:
			return fmt.Errorf("fixture id %q must contain only lowercase letters, digits, '-' or '_'", id)
		}
	}
	if strings.Contains(id, "..") || strings.ContainsAny(id, `/\`) {
		return fmt.Errorf("fixture id %q must not contain path separators", id)
	}
	return nil
}

func failureText(text string, textFile string) (string, error) {
	text = strings.TrimSpace(text)
	textFile = strings.TrimSpace(textFile)
	if text != "" && textFile != "" {
		return "", errors.New("use either --text or --text-file, not both")
	}
	if textFile != "" {
		data, err := os.ReadFile(expandPath(textFile))
		if err != nil {
			return "", err
		}
		text = strings.TrimSpace(string(data))
	}
	if text == "" {
		return "", errors.New("eval add-failure requires --text or --text-file")
	}
	return text, nil
}

func parseRuleExpectations(values []string) (map[string]int, error) {
	expectations := map[string]int{}
	for _, value := range values {
		parts := strings.Split(strings.TrimSpace(value), "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid --expect %q; expected RULE=N", value)
		}
		ruleID := strings.TrimSpace(parts[0])
		countText := strings.TrimSpace(parts[1])
		if !isKnownPolicyRule(ruleID) {
			return nil, fmt.Errorf("unsupported policy rule %q", ruleID)
		}
		count, err := parseNonNegativeInt(countText)
		if err != nil {
			return nil, fmt.Errorf("invalid count in --expect %q: %w", value, err)
		}
		if count == 0 {
			delete(expectations, ruleID)
			continue
		}
		expectations[ruleID] = count
	}
	return expectations, nil
}

func parseNonNegativeInt(value string) (int, error) {
	if value == "" {
		return 0, errors.New("empty integer")
	}
	total := 0
	for _, r := range value {
		if r < '0' || r > '9' {
			return 0, fmt.Errorf("not a non-negative integer: %s", value)
		}
		total = total*10 + int(r-'0')
	}
	return total, nil
}

func isKnownPolicyRule(ruleID string) bool {
	switch ruleID {
	case policyRuleDryRunBarrier,
		policyRuleMainFallbackGuard,
		policyRuleContinuationGuard,
		policyRuleWorkerBoundary,
		policyRuleEvidenceBoundary,
		policyRuleHeartbeatBinding,
		policyRulePendingLedger,
		policyRuleBudgetBoundary:
		return true
	default:
		return false
	}
}

func sameRuleCounts(left map[string]int, right map[string]int) bool {
	normalizedLeft := positiveRuleCounts(left)
	normalizedRight := positiveRuleCounts(right)
	if len(normalizedLeft) != len(normalizedRight) {
		return false
	}
	for key, leftValue := range normalizedLeft {
		if normalizedRight[key] != leftValue {
			return false
		}
	}
	return true
}

func positiveRuleCounts(counts map[string]int) map[string]int {
	normalized := map[string]int{}
	for key, value := range counts {
		if value > 0 {
			normalized[key] = value
		}
	}
	return normalized
}

func resolveRepoRelativePath(repo string, path string) string {
	path = expandPath(path)
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(repo, path)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func summarizePolicyAuditFindings(findings []policyAuditFinding) string {
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
		filepath.Join("docs", "reviews"),
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

func orchestrationPolicyAuditPaths(repo string) ([]string, error) {
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
		filepath.Join("docs", "beta-usability-package.md"),
		filepath.Join("docs", "v2-persistent-ledger-and-heartbeat.md"),
		filepath.Join(".codex-orchestrator", "ledger.json"),
		filepath.Join(".codex-orchestrator", "events.jsonl"),
	} {
		if err := addIfExists(path); err != nil {
			return nil, err
		}
	}
	for _, dir := range []string{
		"routines",
		filepath.Join("docs", "routines"),
		filepath.Join("docs", "reviews"),
		filepath.Join("examples", "routine-reports"),
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
			if strings.HasSuffix(entry.Name(), ".md") || strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".jsonl") {
				paths = append(paths, filepath.Join(dir, entry.Name()))
			}
		}
	}
	paths = uniqueSortedStrings(paths)
	if len(paths) == 0 {
		return nil, errors.New("no orchestration docs, routine specs, routine reports, or ledger/event files found")
	}
	return paths, nil
}

func auditEvidenceText(path string, text string) []evidenceAuditFinding {
	findings := []evidenceAuditFinding{}
	for index, line := range strings.Split(text, "\n") {
		if shouldSkipEvidenceTextLine(line) {
			continue
		}
		if containsAnyFold(line, weakEvidenceTerms()) && containsAnyFold(line, strongEvidenceClaimTerms()) && containsAnyFold(line, evidenceAssertionTerms()) {
			findings = append(findings, newEvidenceAuditFinding(
				evidenceRuleTextOverclaim,
				"%s:%d: local/static suspicion: weak evidence wording appears near strong proof wording: %s",
				path,
				index+1,
				strings.TrimSpace(line),
			))
		}
		if weakEvidencePromotedToStrongTarget(line) {
			findings = append(findings, newEvidenceAuditFinding(
				evidenceRuleTextPromotionTarget,
				"%s:%d: local/static suspicion: weak evidence appears to be promoted to direct/pre/prod/device/runtime/payment proof: %s",
				path,
				index+1,
				strings.TrimSpace(line),
			))
		}
	}
	return findings
}

func weakEvidencePromotedToStrongTarget(line string) bool {
	if !containsAnyFold(line, weakEvidenceTerms()) {
		return false
	}
	if !containsAnyFold(line, strongEvidenceTargetTerms()) {
		return false
	}
	if containsAnyFold(line, explicitDirectEvidenceTerms()) {
		return false
	}
	return containsAnyFold(line, evidencePromotionPhrases())
}

func auditOrchestrationPolicyText(path string, text string) []policyAuditFinding {
	findings := []policyAuditFinding{}
	paragraphs := splitPolicyParagraphs(text)
	for _, paragraph := range paragraphs {
		line := paragraph.startLine
		body := strings.TrimSpace(paragraph.text)
		if body == "" || shouldSkipPolicyParagraph(body) {
			continue
		}
		switch {
		case violatesDryRunBarrier(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRuleDryRunBarrier,
				"%s:%d: dry-run wording appears to allow dispatch/session creation without explicit confirmation: %s",
				path,
				line,
				compactForFinding(body),
			))
		case violatesMainFallbackGuard(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRuleMainFallbackGuard,
				"%s:%d: fallback wording appears to allow implementation in the orchestrator/main checkout after setup failure: %s",
				path,
				line,
				compactForFinding(body),
			))
		case violatesContinuationGuard(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRuleContinuationGuard,
				"%s:%d: heartbeat/child-task completion wording may stop the larger queue without a ledger/roadmap/repo-truth check: %s",
				path,
				line,
				compactForFinding(body),
			))
		case violatesWorkerBoundary(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRuleWorkerBoundary,
				"%s:%d: worker/delegation prompt lacks one of the core boundaries: isolated worktree, allowed/forbidden paths, no subagents/Paseo, self-review, or no merge/push/cleanup: %s",
				path,
				line,
				compactForFinding(body),
			))
		case violatesEvidenceBoundary(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRuleEvidenceBoundary,
				"%s:%d: evidence wording appears to allow local/proxy/weak evidence to be promoted to direct proof: %s",
				path,
				line,
				compactForFinding(body),
			))
		case violatesBudgetBoundary(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRuleBudgetBoundary,
				"%s:%d: budget-policy wording appears to promote local/static budget evidence or imply helper budget enforcement/scheduling behavior: %s",
				path,
				line,
				compactForFinding(body),
			))
		case violatesHeartbeatBindingGuard(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRuleHeartbeatBinding,
				"%s:%d: heartbeat automation wording appears to bind to the literal placeholder \"current\" instead of a real thread id or verified app binding: %s",
				path,
				line,
				compactForFinding(body),
			))
		case violatesPendingLedgerGuard(body):
			findings = append(findings, newPolicyAuditFinding(
				policyRulePendingLedger,
				"%s:%d: pending worktree setup wording appears to keep pendingWorktreeId only in transient prompt/chat state instead of durable ledger truth: %s",
				path,
				line,
				compactForFinding(body),
			))
		}
	}
	return findings
}

type policyParagraph struct {
	startLine int
	text      string
}

func splitPolicyParagraphs(text string) []policyParagraph {
	lines := strings.Split(text, "\n")
	paragraphs := []policyParagraph{}
	current := []string{}
	startLine := 1
	flush := func(endLine int) {
		if len(current) == 0 {
			return
		}
		paragraphs = append(paragraphs, policyParagraph{
			startLine: startLine,
			text:      strings.Join(current, " "),
		})
		current = nil
		startLine = endLine + 1
	}
	for index, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush(index + 1)
			startLine = index + 2
			continue
		}
		if len(current) == 0 {
			startLine = index + 1
		}
		current = append(current, trimmed)
	}
	flush(len(lines))
	return paragraphs
}

func shouldSkipPolicyParagraph(text string) bool {
	return containsAnyFold(text, []string{
		"forbiddenActions",
		"allowedActions",
		"outputSchema",
		"eval fixture",
		"eval fixtures",
		"fixture eval",
		"expectedRuleHits",
		"cover the failures",
		"covers the failures",
		"initial fixtures",
		"第一批 fixture",
		"覆盖真实编排失败类别",
		"真实踩过的编排问题",
	})
}

func violatesDryRunBarrier(text string) bool {
	if !containsAnyFold(text, []string{"dry run", "dry-run", "先做 dry"}) {
		return false
	}
	if containsAnyFold(text, []string{"OPA001", "barrier", "rule", "规则", "屏障"}) {
		return false
	}
	if !containsAnyFold(text, []string{"dispatch", "create session", "create worker", "start worker", "派发", "创建 session", "创建 worker"}) {
		return false
	}
	return !containsAnyFold(text, []string{"confirm", "confirmation", "approve", "approval", "explicit", "wait", "after user", "用户确认", "明确批准", "等待确认", "先给出计划"})
}

func violatesMainFallbackGuard(text string) bool {
	if !containsAnyFold(text, []string{"fallback", "setup failed", "worktree failed", "invalid reference", "回退", "创建失败", "派发失败"}) {
		return false
	}
	if !containsAnyFold(text, []string{"main checkout", "orchestrator checkout", "main workspace", "主仓库", "主工作区", "统领工作区"}) {
		return false
	}
	if !containsAnyFold(text, []string{"implement", "edit", "write code", "fix it", "complete the task", "实现", "修改", "写代码", "接管"}) {
		return false
	}
	return !containsAnyFold(text, []string{"do not", "must not", "never", "blocked", "report", "不得", "不要", "不能", "不许", "报告 blocker"})
}

func violatesContinuationGuard(text string) bool {
	if !containsAnyFold(text, []string{"delete heartbeat", "stop heartbeat", "stop the loop", "child task", "single task", "任务完成", "删除 heartbeat", "停止"}) {
		return false
	}
	if !containsAnyFold(text, []string{"complete", "completed", "merged", "cleaned", "完成", "合并", "清理"}) {
		return false
	}
	return !containsAnyFold(text, []string{"ledger", "roadmap", "repo truth", "queue", "next task", "continue", "replace heartbeat", "队列", "路线图", "继续", "下一个", "检查"})
}

func violatesWorkerBoundary(text string) bool {
	if containsAnyFold(text, []string{"OPA004"}) {
		return false
	}
	if !containsAnyFold(text, []string{"codex_delegation", "worker prompt:", "delegated implementation worker", "delegated proof/implementation worker", "你是 delegated", "你是 TastyFuture 的 delegated"}) {
		return false
	}
	if !containsAnyFold(text, []string{"worktree", "branch", "session", "工作树", "分支"}) {
		return false
	}
	required := [][]string{
		{"worktree", "isolated", "separate", "独立", "隔离"},
		{"forbidden path", "forbidden paths", "forbidden files", "forbidden directories", "禁止路径", "禁止文件"},
		{"subagent", "sub agent", "Paseo", "二级", "子 agent"},
		{"self-review", "self review", "自审"},
		{"do not merge", "not merge", "do not push", "not push", "不 merge", "不 push", "不要 merge", "不要 push", "不合并", "不推送"},
	}
	for _, group := range required {
		if !containsAnyFold(text, group) {
			return true
		}
	}
	return false
}

func violatesEvidenceBoundary(text string) bool {
	lower := strings.ToLower(text)
	if !containsAnyFold(text, weakEvidenceTerms()) || !containsAnyFold(text, []string{"direct", "direct proof", "direct evidence", "直接", "直接证明"}) {
		return false
	}
	if containsAnyFold(text, evidenceNegationTerms()) {
		return false
	}
	return strings.Contains(lower, "promote local") ||
		strings.Contains(lower, "promote proxy") ||
		strings.Contains(lower, "promote weak") ||
		strings.Contains(lower, "promote to direct") ||
		strings.Contains(lower, "treat as direct") ||
		strings.Contains(lower, "counts as direct") ||
		strings.Contains(lower, "claim as direct") ||
		strings.Contains(lower, "upgrade to direct") ||
		strings.Contains(lower, "写成 direct") ||
		strings.Contains(lower, "写成直接") ||
		strings.Contains(lower, "算作 direct") ||
		strings.Contains(lower, "算作直接") ||
		strings.Contains(lower, "当成 direct") ||
		strings.Contains(lower, "当成直接") ||
		strings.Contains(lower, "升级为 direct") ||
		strings.Contains(lower, "升级为直接")
}

func violatesBudgetBoundary(text string) bool {
	if containsAnyFold(text, []string{"OPA008"}) || containsAnyFold(text, evidenceNegationTerms()) {
		return false
	}
	if containsAnyFold(text, []string{"catch claims", "catches claims", "forbidden from", "must keep these categories separate", "merge, push, delete branches", "必须", "禁止"}) {
		return false
	}
	if !containsAnyFold(text, []string{"budget", "预算"}) {
		return false
	}
	localBudgetEvidence := containsAnyFold(text, []string{
		"local/static",
		"local static",
		"ledger timestamp",
		"recorded timestamp",
		"task timestamp",
		"heartbeat budgetpressure",
		"heartbeat budget pressure",
		"heartbeat warning",
		"budget warning",
		"budgetpressure",
		"helper evidence",
		"local ledger",
		"ledger",
		"heartbeat",
		"本地",
		"静态",
	})
	if localBudgetEvidence && containsAnyFold(text, evidenceAssertionTerms()) && containsAnyFold(text, []string{
		"direct runtime proof",
		"direct proof",
		"runtime proof",
		"live runtime",
		"worker runtime proof",
		"session runtime proof",
		"proves live worker",
		"proves the worker",
		"证明运行时",
		"直接证明",
	}) {
		return true
	}
	helperControl := containsAnyFold(text, []string{
		"helper",
		"budget-policy-report",
		"budget policy report",
		"heartbeat",
		"budget warning",
		"budget warnings",
		"budgetpressure",
	})
	if helperControl && containsAnyFold(text, []string{
		"automatically paused",
		"automatically pauses",
		"auto-paused",
		"auto pauses",
		"paused workers",
		"pause workers",
		"pause worker",
		"kill workers",
		"kills workers",
		"killed workers",
		"automatic killing",
		"enforced dispatch eligibility",
		"enforces dispatch eligibility",
		"make dispatch eligibility decisions",
		"scheduler",
		"schedules workers",
		"schedule workers",
		"prioritizer",
		"prioritizes workers",
		"priority engine",
		"worker-control",
		"worker control",
		"暂停 worker",
		"杀掉 worker",
		"调度",
		"排序",
	}) {
		return true
	}
	lower := strings.ToLower(text)
	return (strings.Contains(lower, "helper") || strings.Contains(lower, "heartbeat")) &&
		strings.Contains(lower, "enforce") &&
		(strings.Contains(lower, "dispatch") || strings.Contains(lower, "worker"))
}

func violatesHeartbeatBindingGuard(text string) bool {
	if containsAnyFold(text, []string{"OPA006"}) || containsAnyFold(text, evidenceNegationTerms()) {
		return false
	}
	if !containsAnyFold(text, []string{"heartbeat", "automation", "target_thread_id", "targetThreadId", "定时", "心跳"}) {
		return false
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "not correctly bound") || strings.Contains(lower, "must be a real thread id") {
		return false
	}
	return strings.Contains(lower, `target_thread_id = "current"`) ||
		strings.Contains(lower, `target_thread_id="current"`) ||
		strings.Contains(lower, `targetthreadid: "current"`) ||
		strings.Contains(lower, `targetthreadid = "current"`) ||
		strings.Contains(lower, `targetthreadid current`) ||
		strings.Contains(lower, `target thread id current`) ||
		strings.Contains(lower, `target_thread_id current`)
}

func violatesPendingLedgerGuard(text string) bool {
	if containsAnyFold(text, []string{"OPA007"}) {
		return false
	}
	if !containsAnyFold(text, []string{"pendingWorktreeId", "pending worktree", "pending setup", "pending-worktree", "pending id"}) {
		return false
	}
	if !containsAnyFold(text, []string{"ledger", "heartbeat prompt", "chat", "memory", "automation prompt", "聊天", "记忆"}) {
		return false
	}
	lower := strings.ToLower(text)
	if strings.Contains(lower, "durable ledger truth immediately") ||
		strings.Contains(lower, "record that pending setup in durable ledger truth") ||
		strings.Contains(lower, "do not keep pending setup state only") {
		return false
	}
	return strings.Contains(lower, "only in heartbeat") ||
		strings.Contains(lower, "only in the heartbeat") ||
		strings.Contains(lower, "only in chat") ||
		strings.Contains(lower, "only in memory") ||
		strings.Contains(lower, "skip ledger") ||
		strings.Contains(lower, "without ledger") ||
		strings.Contains(lower, "do not record") && strings.Contains(lower, "ledger") ||
		strings.Contains(lower, "don't record") && strings.Contains(lower, "ledger") ||
		strings.Contains(lower, "not record") && strings.Contains(lower, "ledger") ||
		strings.Contains(lower, "只写") && (strings.Contains(lower, "heartbeat") || strings.Contains(lower, "prompt") || strings.Contains(lower, "聊天")) ||
		strings.Contains(lower, "不写") && strings.Contains(lower, "ledger") ||
		strings.Contains(lower, "不用") && strings.Contains(lower, "ledger")
}

func compactForFinding(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	if len(text) <= 220 {
		return text
	}
	return text[:217] + "..."
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
	if isEvidenceFixtureDescriptionLine(trimmed) {
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
		"label evidence as",
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

func isEvidenceFixtureDescriptionLine(line string) bool {
	return containsAnyFold(line, []string{
		"candidate fixture",
		"candidate fixtures",
		"fixture should",
		"should fail",
		"should be flagged",
		"should trigger",
		"must fail",
		"must be flagged",
		"expected rule hit",
		"expectedRuleHits",
		"failure fixture",
		"bad example",
		"known bad",
		"claims such as",
		"such as",
		"for example",
		"example:",
		"local timestamp proves live runtime",
		"checks whether",
		"check whether",
		"check if",
		"catch claims",
		"catches claims",
		"prevent",
		"prevents",
		"linter",
		"lint rule",
		"this MVP emits",
		"MVP evidence",
		"or claim direct/proxy runtime proof",
		"or claim runtime proof",
		"or claim production/runtime proof",
		"特别检查",
		"检查",
		"防止",
		"证据升级",
		"local/static timestamp",
		"反例",
		"失败样例",
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
		"pre verification",
		"pre verified",
		"device proof",
		"real-device proof",
		"payment proof",
		"hardware proof",
		"production runtime",
		"live runtime",
		"runtime verified",
		"prod verified",
		"device verified",
		"payment verified",
		"hardware verified",
		"直接证明",
		"直接证据",
		"运行时证明",
		"生产证明",
		"真实设备证明",
		"支付证明",
		"硬件证明",
	}
}

func strongEvidenceTargetTerms() []string {
	return []string{
		"direct",
		"direct proof",
		"direct evidence",
		"runtime",
		"runtime proof",
		"runtime verified",
		"pre proof",
		"pre verified",
		"production",
		"production proof",
		"production runtime",
		"prod proof",
		"prod verified",
		"device",
		"device proof",
		"device verified",
		"real-device",
		"hardware",
		"hardware proof",
		"hardware verified",
		"payment",
		"payment proof",
		"payment verified",
		"PAX",
		"terminal",
		"直接",
		"直接证明",
		"运行时",
		"预发",
		"生产",
		"设备",
		"真机",
		"硬件",
		"支付",
	}
}

func explicitDirectEvidenceTerms() []string {
	return []string{
		"explicit direct evidence",
		"direct evidence attached",
		"direct evidence recorded",
		"direct proof attached",
		"direct proof recorded",
		"runtime log attached",
		"runtime logs attached",
		"device screenshot attached",
		"payment receipt attached",
		"human-reviewed direct",
		"有直接证据",
		"直接证据已记录",
		"已附直接证据",
	}
}

func evidencePromotionPhrases() []string {
	return []string{
		"as direct",
		"as pre",
		"as prod",
		"as production",
		"as device",
		"as runtime",
		"as payment",
		"count as direct",
		"count as pre",
		"count as prod",
		"count as production",
		"count as device",
		"count as runtime",
		"count as payment",
		"counts as direct",
		"counts as pre",
		"counts as prod",
		"counts as production",
		"counts as device",
		"counts as runtime",
		"counts as payment",
		"treat as direct",
		"treat as pre",
		"treat as prod",
		"treat as production",
		"treat as device",
		"treat as runtime",
		"treat as payment",
		"promote to direct",
		"promote to pre",
		"promote to prod",
		"promote to production",
		"promote to device",
		"promote to runtime",
		"promote to payment",
		"upgrade to direct",
		"upgrade to pre",
		"upgrade to prod",
		"upgrade to production",
		"upgrade to device",
		"upgrade to runtime",
		"upgrade to payment",
		"写成 direct",
		"写成 pre",
		"写成 prod",
		"写成生产",
		"写成设备",
		"写成运行时",
		"写成支付",
		"算作 direct",
		"算作 pre",
		"算作 prod",
		"算作生产",
		"算作设备",
		"算作运行时",
		"算作支付",
		"升级为 direct",
		"升级为生产",
		"升级为设备",
		"升级为运行时",
		"升级为支付",
	}
}

func evidenceNegationTerms() []string {
	return []string{
		"do not",
		"does not",
		"don't",
		"must not",
		"not claim",
		"not direct",
		"not production",
		"not runtime",
		"no direct",
		"no runtime",
		"no scheduler",
		"forbidden",
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
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	shell, args := shellCommand(command)
	cmd := exec.CommandContext(ctx, shell, args...)
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

func shellCommand(command string) (string, []string) {
	if runtime.GOOS == "windows" {
		shell := os.Getenv("COMSPEC")
		if shell == "" {
			shell = "cmd"
		}
		return shell, []string{"/C", command}
	}
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell, []string{"-c", command}
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
