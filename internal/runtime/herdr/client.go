// Package herdr implements a gascity runtime.Provider backed by herdr
// (https://herdr.dev) — a terminal workspace manager for AI coding agents.
//
// It shells out to the `herdr` CLI (which wraps herdr's local JSON socket API),
// mirroring the tmux provider's executor pattern, and parses the JSON envelope
// each verb emits. herdr is opt-in via the "herdr" runtime selector; tmux stays
// the default. See herdr-provider-design.md for the full interface mapping and
// the 0.7.1 validation notes.
//
// Model: one shared herdr *session* per city (≈ the tmux `-L gc` server). Within
// that session agents are grouped one *workspace* per rig (or per town) and one
// *tab* per agent, so each gascity session is its own switchable space rather
// than a tiled pane. Agents are addressable by name, 1:1 with gascity session
// names.
package herdr

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// client runs `herdr` CLI verbs against a named herdr session and decodes the
// response envelope ({"id":…,"result":…} | {"id":…,"error":{code,message}}).
type client struct {
	session  string     // herdr named session (shared per city)
	bin      string     // herdr binary (default "herdr")
	cityRoot string     // city root: the shared server's launch cwd, and the effectiveWorkDir fallback when a session's WorkDir doesn't exist yet (empty in city-less/standalone construction)
	serverMu sync.Mutex // serializes startServer: serverAlive → removeStaleSocket → launch → readiness
}

func newClient(session, cityRoot string) *client {
	return &client{session: session, bin: "herdr", cityRoot: cityRoot}
}

type herdrError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *herdrError) Error() string {
	return e.Code + ": " + e.Message
}

type envelope struct {
	Result json.RawMessage `json:"result"`
	Error  *herdrError     `json:"error"`
}

func decodeHerdrError(data []byte) *herdrError {
	var env envelope
	if json.Unmarshal(data, &env) == nil && env.Error != nil {
		return env.Error
	}
	return nil
}

// run executes `herdr --session <session> <args…>` and returns the result
// payload, or an error (transport failure or herdr-reported error).
func (c *client) run(ctx context.Context, args ...string) (json.RawMessage, error) {
	full := append([]string{"--session", c.session}, args...)
	out, err := exec.CommandContext(ctx, c.bin, full...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			if herr := decodeHerdrError(ee.Stderr); herr != nil {
				return nil, fmt.Errorf("herdr %v: %w", args, herr)
			}
			return nil, fmt.Errorf("herdr %v: %s", args, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("herdr %v: %w", args, err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return nil, nil // success with no payload (e.g. pane send-keys / pane run)
	}
	var env envelope
	if err := json.Unmarshal(out, &env); err != nil {
		return nil, fmt.Errorf("herdr %v: decode response: %w", args, err)
	}
	if env.Error != nil {
		return nil, fmt.Errorf("herdr %v: %w", args, env.Error)
	}
	return env.Result, nil
}

// agentInfo mirrors herdr's agent object.
type agentInfo struct {
	Name        string `json:"name"`
	PaneID      string `json:"pane_id"`
	WorkspaceID string `json:"workspace_id"`
	TabID       string `json:"tab_id"`
	TerminalID  string `json:"terminal_id"`
	AgentStatus string `json:"agent_status"`
	Cwd         string `json:"cwd"`
}

// startAgent → `herdr agent start <name> --kind <kind> --pane <paneID>
// [-- <agent args…>]`. Herdr 0.7.5 launches supported coding agents into an
// existing shell pane; placement creation already gives us that pane with the
// requested cwd and environment.
func (c *client) startAgent(ctx context.Context, name, kind, paneID string, argv []string) (agentInfo, error) {
	args := []string{"agent", "start", name, "--kind", kind, "--pane", paneID}
	if len(argv) > 1 {
		args = append(args, "--")
		args = append(args, argv[1:]...)
	}
	res, err := c.run(ctx, args...)
	if err != nil {
		return agentInfo{}, err
	}
	var wrap struct {
		Agent agentInfo `json:"agent"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return agentInfo{}, fmt.Errorf("herdr agent start: decode: %w", err)
	}
	return wrap.Agent, nil
}

// listAgents → `herdr agent list`.
func (c *client) listAgents(ctx context.Context) ([]agentInfo, error) {
	res, err := c.run(ctx, "agent", "list")
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Agents []agentInfo `json:"agents"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return nil, fmt.Errorf("herdr agent list: decode: %w", err)
	}
	return wrap.Agents, nil
}

// read → `herdr agent read <name> --source <source> [--lines n]`. Use
// "visible" for the current screen (the liveness/fingerprint snapshot);
// "recent"/"recent-unwrapped" are scrollback only.
func (c *client) read(ctx context.Context, name, source string, lines int) (string, error) {
	args := []string{"agent", "read", name, "--source", source}
	if lines > 0 {
		args = append(args, "--lines", strconv.Itoa(lines))
	}
	res, err := c.run(ctx, args...)
	if err != nil {
		return "", err
	}
	var wrap struct {
		Read struct {
			Text string `json:"text"`
		} `json:"read"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return "", fmt.Errorf("herdr agent read: decode: %w", err)
	}
	return wrap.Read.Text, nil
}

// proc is one process in a pane's foreground tree.
type proc struct {
	PID  int      `json:"pid"`
	Name string   `json:"name"`
	Argv []string `json:"argv"`
	Cwd  string   `json:"cwd"`
}

// processInfo → `herdr pane process-info --pane <paneID>`: shell PID + the
// foreground process tree (powers ProcessAlive and the hard-kill path).
func (c *client) processInfo(ctx context.Context, paneID string) (shellPID int, fg []proc, err error) {
	res, e := c.run(ctx, "pane", "process-info", "--pane", paneID)
	if e != nil {
		return 0, nil, e
	}
	var wrap struct {
		ProcessInfo struct {
			ShellPID            int    `json:"shell_pid"`
			ForegroundProcesses []proc `json:"foreground_processes"`
		} `json:"process_info"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return 0, nil, fmt.Errorf("herdr pane process-info: decode: %w", err)
	}
	return wrap.ProcessInfo.ShellPID, wrap.ProcessInfo.ForegroundProcesses, nil
}

// sendKeys → `herdr pane send-keys <paneID> <key…>` (raw keys, e.g. ctrl+c, enter).
func (c *client) sendKeys(ctx context.Context, paneID string, keys ...string) error {
	args := append([]string{"pane", "send-keys", paneID}, keys...)
	_, err := c.run(ctx, args...)
	return err
}

// paneRun → `herdr pane run <paneID> <command>` (pastes text into the pane).
func (c *client) paneRun(ctx context.Context, paneID, command string) error {
	_, err := c.run(ctx, "pane", "run", paneID, command)
	return err
}

// deliverNudge types a nudge into the agent's input and submits it, then
// confirms the submit actually landed. The text is injected with `pane run`
// (paste semantics: multi-line content is preserved and the paste's own trailing
// newline is swallowed by the TUI, so the text never submits on its own).
//
// Submission is the hard part. Two facts, learned empirically against herdr 0.7.1
// + the Claude Code TUI:
//
//   - The TUI must be at a ready input prompt: a submit delivered mid-boot is
//     swallowed. Callers deliver to a ready agent — Start waits for idle first
//     (see startupNudgeIdleTimeout); the Nudge path targets running agents.
//   - A submit that races the paste-commit is swallowed, stranding the prompt
//     typed-but-unsubmitted — the agent then idles forever with work it never
//     began (the missed startup-nudge stall).
//
// The prior open-loop form (settle → CR → settle → CR, via `agent send "\r"`) was
// not enough under concurrent restart-time boot load: both CRs raced the paste
// and the nudge stranded, and the swallowed result hid it. This is now
// closed-loop: press Enter as a real key event (`pane send-keys`, which submits
// reliably where a pasted `\r` did not), then verify via `agent get` that the
// agent actually left its idle prompt. Retry the Enter until it does, bounded so
// a nudge that legitimately produces no work cannot spin. A redundant Enter on an
// already-submitted/empty prompt is a harmless no-op. Returns an error if the
// submit never confirms, so the caller can surface it instead of silently
// leaving a stranded agent.
//
// Contract: inject + submit by pane id, confirm by agent name.
func (c *client) deliverNudge(ctx context.Context, paneID, name, text string) error {
	if err := c.paneRun(ctx, paneID, text); err != nil {
		return err
	}
	time.Sleep(submitSettleDelay) // let the paste commit before the first submit
	var lastErr error
	for attempt := 0; attempt < submitMaxAttempts; attempt++ {
		if err := c.sendKeys(ctx, paneID, "Enter"); err != nil {
			lastErr = err // transient send failure; verify + retry within the bound
		}
		time.Sleep(submitSettleDelay)
		info, ok, err := c.getAgent(ctx, name)
		switch {
		case err != nil:
			lastErr = err // transient read failure; retry within the bound
		case !ok:
			return fmt.Errorf("herdr deliverNudge: agent %q vanished before submit confirmed", name)
		case !strings.EqualFold(strings.TrimSpace(info.AgentStatus), "idle"):
			return nil // left the idle prompt → submit landed, agent is running
		}
	}
	if lastErr != nil {
		return fmt.Errorf("herdr deliverNudge: %q still idle after %d submit attempts: %w", name, submitMaxAttempts, lastErr)
	}
	return fmt.Errorf("herdr deliverNudge: %q still idle after %d submit attempts (nudge typed-but-unsubmitted?)", name, submitMaxAttempts)
}

// submitSettleDelay is how long deliverNudge waits for a `pane run` paste to
// commit in the TUI before each submit Enter and before re-reading agent status.
// A submit that races the paste is swallowed; ~1s clears it with margin even
// under the concurrent boot load of a town-wide restart.
const submitSettleDelay = 1 * time.Second

// submitMaxAttempts bounds the closed-loop submit: ~submitMaxAttempts·settle is
// the worst-case latency before deliverNudge gives up and returns an error. Sized
// to cover a slow paste-commit under restart-time load without spinning on a
// nudge that legitimately leaves the agent idle.
const submitMaxAttempts = 5

// closePane → `herdr pane close <paneID>`.
func (c *client) closePane(ctx context.Context, paneID string) error {
	_, err := c.run(ctx, "pane", "close", paneID)
	return err
}

// getAgent fetches one agent by name: (info, true, nil) if present,
// (zero, false, nil) if herdr reports it absent, (_, false, err) on failure.
func (c *client) getAgent(ctx context.Context, name string) (agentInfo, bool, error) {
	res, err := c.run(ctx, "agent", "get", name)
	if err != nil {
		if strings.Contains(err.Error(), "not_found") || strings.Contains(err.Error(), "not found") {
			return agentInfo{}, false, nil
		}
		return agentInfo{}, false, err
	}
	var wrap struct {
		Agent agentInfo `json:"agent"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return agentInfo{}, false, fmt.Errorf("herdr agent get: decode: %w", err)
	}
	return wrap.Agent, true, nil
}

// ── workspace / tab placement ────────────────────────────────────────────────
//
// herdr's tree is workspace › tab › pane. To give each agent its own switchable
// space (vs tiling every agent as a pane in one tab), Start groups agents one
// workspace per rig/town and one tab per agent. `workspace create` and `tab
// create` each auto-spawn a stray shell pane; the caller closes it so the tab
// holds only the agent.

type workspaceInfo struct {
	WorkspaceID string `json:"workspace_id"`
	Label       string `json:"label"`
}

type tabInfo struct {
	TabID string `json:"tab_id"`
	Label string `json:"label"`
}

type agentSessionInfo struct {
	Agent  string `json:"agent"`
	Kind   string `json:"kind"`
	Source string `json:"source"`
	Value  string `json:"value"`
}

type paneInfo struct {
	PaneID       string            `json:"pane_id"`
	TabID        string            `json:"tab_id"`
	WorkspaceID  string            `json:"workspace_id"`
	Agent        *string           `json:"agent"`
	AgentSession *agentSessionInfo `json:"agent_session"`
}

type placement struct {
	WorkspaceID string
	TabID       string
	PaneID      string
	CreatedPane bool
}

// placementUnavailableError marks an explicitly observed, stable placement
// collision. Operational errors (transport, decode, or lookup failures) are
// deliberately not wrapped: callers must not create recovery tabs from an
// uncertain view of the server.
type placementUnavailableError struct{ err error }

func (e *placementUnavailableError) Error() string { return e.err.Error() }
func (e *placementUnavailableError) Unwrap() error { return e.err }

func placementUnavailablef(format string, args ...any) error {
	return &placementUnavailableError{err: fmt.Errorf(format, args...)}
}

func isPlacementUnavailable(err error) bool {
	var unavailable *placementUnavailableError
	return errors.As(err, &unavailable)
}

// findWorkspace returns the id of the workspace whose label matches, or "".
// findWorkspace returns the unique workspace whose label matches. Duplicate
// exact labels are ambiguous and are never resolved by list order.
func (c *client) findWorkspace(ctx context.Context, label string) (string, error) {
	res, err := c.run(ctx, "workspace", "list")
	if err != nil {
		return "", err
	}
	var wrap struct {
		Workspaces []workspaceInfo `json:"workspaces"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return "", fmt.Errorf("herdr workspace list: decode: %w", err)
	}
	matches := make([]workspaceInfo, 0, 1)
	for _, w := range wrap.Workspaces {
		if w.Label == label {
			matches = append(matches, w)
		}
	}
	if len(matches) > 1 {
		ids := make([]string, len(matches))
		for i := range matches {
			ids[i] = matches[i].WorkspaceID
		}
		return "", fmt.Errorf("herdr workspace label %q is ambiguous: workspace_ids=%v", label, ids)
	}
	if len(matches) == 1 {
		return matches[0].WorkspaceID, nil
	}
	return "", nil
}

// workspaceCreate makes a workspace labeled label and returns its id plus the
// default tab and shell pane that herdr creates inside it. The pane is the
// launch target for `herdr agent start --pane`.
func (c *client) workspaceCreate(ctx context.Context, label, cwd string, env map[string]string) (wsID, tabID, paneID string, err error) {
	args := []string{"workspace", "create", "--label", label, "--no-focus"}
	args = appendPlacementEnv(args, cwd, env)
	res, err := c.run(ctx, args...)
	if err != nil {
		return "", "", "", err
	}
	var wrap struct {
		Workspace struct {
			WorkspaceID string `json:"workspace_id"`
		} `json:"workspace"`
		Tab struct {
			TabID string `json:"tab_id"`
		} `json:"tab"`
		RootPane struct {
			PaneID string `json:"pane_id"`
		} `json:"root_pane"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return "", "", "", fmt.Errorf("herdr workspace create: decode: %w", err)
	}
	return wrap.Workspace.WorkspaceID, wrap.Tab.TabID, wrap.RootPane.PaneID, nil
}

// findTab returns the id of the tab in wsID whose label matches, or "".
// findTab returns the unique tab in wsID whose label matches. Duplicate exact
// labels are reported rather than selecting an arbitrary placement.
func (c *client) findTab(ctx context.Context, wsID, label string) (string, error) {
	res, err := c.run(ctx, "tab", "list", "--workspace", wsID)
	if err != nil {
		return "", err
	}
	var wrap struct {
		Tabs []tabInfo `json:"tabs"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return "", fmt.Errorf("herdr tab list: decode: %w", err)
	}
	matches := make([]tabInfo, 0, 1)
	for _, tab := range wrap.Tabs {
		if tab.Label == label {
			matches = append(matches, tab)
		}
	}
	if len(matches) > 1 {
		ids := make([]string, len(matches))
		for i := range matches {
			ids[i] = matches[i].TabID
		}
		return "", fmt.Errorf("herdr tab label %q is ambiguous in workspace %q: tab_ids=%v", label, wsID, ids)
	}
	if len(matches) == 1 {
		return matches[0].TabID, nil
	}
	return "", nil
}

// tabCreate makes a tab labeled label in wsID and returns its id plus the shell
// pane that herdr creates inside it.
func (c *client) tabCreate(ctx context.Context, wsID, label, cwd string, env map[string]string) (tabID, paneID string, err error) {
	args := []string{"tab", "create", "--workspace", wsID, "--label", label, "--no-focus"}
	args = appendPlacementEnv(args, cwd, env)
	res, err := c.run(ctx, args...)
	if err != nil {
		return "", "", err
	}
	var wrap struct {
		Tab struct {
			TabID string `json:"tab_id"`
		} `json:"tab"`
		RootPane struct {
			PaneID string `json:"pane_id"`
		} `json:"root_pane"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return "", "", fmt.Errorf("herdr tab create: decode: %w", err)
	}
	return wrap.Tab.TabID, wrap.RootPane.PaneID, nil
}

// tabRename relabels a tab (cosmetic; best-effort at the call site).
func (c *client) tabRename(ctx context.Context, tabID, label string) error {
	_, err := c.run(ctx, "tab", "rename", tabID, label)
	return err
}

func (c *client) listTabPanes(ctx context.Context, wsID, tabID string) ([]paneInfo, error) {
	res, err := c.run(ctx, "pane", "list", "--workspace", wsID)
	if err != nil {
		return nil, err
	}
	var wrap struct {
		Panes []paneInfo `json:"panes"`
	}
	if err := json.Unmarshal(res, &wrap); err != nil {
		return nil, fmt.Errorf("herdr pane list: decode: %w", err)
	}
	panes := make([]paneInfo, 0, 1)
	for _, pane := range wrap.Panes {
		if pane.TabID == tabID {
			panes = append(panes, pane)
		}
	}
	return panes, nil
}

func shellProcess(name string) bool {
	name = strings.TrimPrefix(strings.ToLower(filepath.Base(strings.TrimSpace(name))), "-")
	switch name {
	case "sh", "bash", "dash", "zsh", "fish", "ksh", "ksh93", "mksh", "ash", "nu", "elvish", "xonsh", "pwsh":
		return true
	default:
		return false
	}
}

func (c *client) reusableShellPane(ctx context.Context, wsID, tabID string) (string, error) {
	panes, err := c.listTabPanes(ctx, wsID, tabID)
	if err != nil {
		return "", err
	}
	if len(panes) != 1 {
		return "", placementUnavailablef("workspace=%q tab=%q blocked: pane_count=%d panes=%s", wsID, tabID, len(panes), describePanes(panes))
	}
	pane := panes[0]
	if pane.Agent != nil || pane.AgentSession != nil {
		return "", placementUnavailablef("workspace=%q tab=%q pane=%q blocked: agent=%q agent_session=%q", wsID, tabID, pane.PaneID, stringValue(pane.Agent), describeAgentSession(pane.AgentSession))
	}
	_, foreground, err := c.processInfo(ctx, pane.PaneID)
	if err != nil {
		return "", fmt.Errorf("workspace=%q tab=%q pane=%q process preflight: %w", wsID, tabID, pane.PaneID, err)
	}
	if len(foreground) != 1 || !shellProcess(foreground[0].Name) {
		return "", placementUnavailablef("workspace=%q tab=%q pane=%q blocked: agent=%q agent_session=%q foreground=%v", wsID, tabID, pane.PaneID, stringValue(pane.Agent), describeAgentSession(pane.AgentSession), foreground)
	}
	return pane.PaneID, nil
}

func stringValue(value *string) string {
	if value == nil {
		return "<nil>"
	}
	return *value
}

func describeAgentSession(session *agentSessionInfo) string {
	if session == nil {
		return "<nil>"
	}
	return fmt.Sprintf("agent=%s kind=%s source=%s value=%s", session.Agent, session.Kind, session.Source, session.Value)
}

func describePanes(panes []paneInfo) string {
	parts := make([]string, len(panes))
	for i, pane := range panes {
		parts[i] = fmt.Sprintf("%s(agent=%s,session=%s)", pane.PaneID, stringValue(pane.Agent), describeAgentSession(pane.AgentSession))
	}
	return "[" + strings.Join(parts, ",") + "]"
}

// ensurePlacement resolves the unique intended placement and conservatively
// reuses an existing pane only when it is an unowned, shell-only pane.
func (c *client) ensurePlacement(ctx context.Context, wsLabel, tabLabel, cwd string, env map[string]string) (placement, error) {
	wsID, err := c.findWorkspace(ctx, wsLabel)
	if err != nil {
		return placement{}, err
	}
	if wsID == "" {
		wsID, tabID, paneID, err := c.workspaceCreate(ctx, wsLabel, cwd, env)
		if err != nil {
			return placement{}, err
		}
		_ = c.tabRename(ctx, tabID, tabLabel)
		return placement{WorkspaceID: wsID, TabID: tabID, PaneID: paneID, CreatedPane: true}, nil
	}
	tabID, err := c.findTab(ctx, wsID, tabLabel)
	if err != nil {
		return placement{}, err
	}
	if tabID == "" {
		tabID, paneID, err := c.tabCreate(ctx, wsID, tabLabel, cwd, env)
		return placement{WorkspaceID: wsID, TabID: tabID, PaneID: paneID, CreatedPane: err == nil}, err
	}
	paneID, err := c.reusableShellPane(ctx, wsID, tabID)
	if err != nil {
		return placement{WorkspaceID: wsID, TabID: tabID}, err
	}
	return placement{WorkspaceID: wsID, TabID: tabID, PaneID: paneID}, nil
}

func recoveryTabLabel(tabLabel string) string {
	sum := sha256.Sum256([]byte(tabLabel))
	suffix := fmt.Sprintf("-gc-%x", sum[:4])
	maxBase := 32 - len(suffix)
	if len(tabLabel) > maxBase {
		tabLabel = tabLabel[:maxBase]
	}
	return tabLabel + suffix
}

func isHerdrError(err error, code string) bool {
	var herr *herdrError
	return errors.As(err, &herr) && herr.Code == code
}

func appendPlacementEnv(args []string, cwd string, env map[string]string) []string {
	if cwd != "" {
		args = append(args, "--cwd", cwd)
	}
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		args = append(args, "--env", key+"="+env[key])
	}
	return args
}

// socketPath is the unix socket for this client's herdr session.
func (c *client) socketPath() string {
	home, _ := os.UserHomeDir()
	if c.session == "" || c.session == "default" {
		return filepath.Join(home, ".config", "herdr", "herdr.sock")
	}
	return filepath.Join(home, ".config", "herdr", "sessions", c.session, "herdr.sock")
}

// serverAlive reports whether the session-server is actually accepting
// connections on its socket. A bare os.Stat is insufficient: a herdr server
// that exits uncleanly leaves its socket inode behind — and herdr's own
// `session stop` can't remove it, since that too needs a live server to reach —
// so the stale socket answers connects with ECONNREFUSED. Presence != liveness;
// dial to find out for real.
func (c *client) serverAlive() bool {
	conn, err := net.DialTimeout("unix", c.socketPath(), 500*time.Millisecond)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// removeStaleSocket unlinks the socket inode when it exists but nothing live is
// listening, so a freshly launched server can bind. Guard with serverAlive
// first — only call once liveness has already returned false.
func (c *client) removeStaleSocket() {
	if fi, err := os.Stat(c.socketPath()); err == nil && fi.Mode()&os.ModeSocket != 0 {
		_ = os.Remove(c.socketPath())
	}
}

// startServer launches the headless herdr server for this session (detached)
// and waits for it to accept connections. Idempotent — no-op if already live.
func (c *client) startServer() error {
	c.serverMu.Lock()
	defer c.serverMu.Unlock()
	if c.serverAlive() {
		return nil
	}
	// A prior server may have died leaving a stale socket inode; serverAlive
	// just confirmed nothing live owns it, so clear it before launch or herdr
	// cannot bind — the exact failure that stranded provider swaps (agent list
	// → ECONNREFUSED → swap aborted → pool polecats stuck start-pending).
	c.removeStaleSocket()
	devnull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return fmt.Errorf("herdr server: open devnull: %w", err)
	}
	defer func() { _ = devnull.Close() }()
	cmd := exec.Command(c.bin, "--session", c.session, "server")
	cmd.Stdout, cmd.Stderr = devnull, devnull
	// Launch the shared daemon in the city root, not the inherited cwd (which is
	// often $HOME when gc is invoked from a login shell). Sessions whose --cwd is
	// empty/nonexistent fall back to this server cwd, so a $HOME-rooted server
	// stranded ephemeral pool spawns in $HOME (unprimed, re-prompted for trust).
	// Empty cityRoot (city-less construction) leaves cwd inherited, as before.
	cmd.Dir = c.cityRoot
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("herdr server start: %w", err)
	}
	_ = cmd.Process.Release() // detach; herdr owns the daemon lifetime
	for i := 0; i < 40; i++ {
		if c.serverAlive() {
			return nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return fmt.Errorf("herdr server for session %q did not become ready", c.session)
}

// stopServer stops this session's server (best-effort; tolerates not-running).
// `session stop` targets the session by name and must bypass run() (which
// prepends --session).
func (c *client) stopServer() error {
	_ = exec.Command(c.bin, "session", "stop", c.session).Run()
	return nil
}
