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
	sockPath string     // test override for socketPath (unit tests point it at a fake server)
	serverMu sync.Mutex // serializes startServer: serverAlive → removeStaleSocket → launch → readiness
}

func newClient(session, cityRoot string) *client {
	return &client{session: session, bin: "herdr", cityRoot: cityRoot}
}

type herdrError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	Result json.RawMessage `json:"result"`
	Error  *herdrError     `json:"error"`
}

// run executes `herdr --session <session> <args…>` and returns the result
// payload, or an error (transport failure or herdr-reported error).
func (c *client) run(ctx context.Context, args ...string) (json.RawMessage, error) {
	full := append([]string{"--session", c.session}, args...)
	out, err := exec.CommandContext(ctx, c.bin, full...).Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return nil, fmt.Errorf("herdr %v: %s", args, ee.Stderr)
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
		return nil, fmt.Errorf("herdr %v: %s: %s", args, env.Error.Code, env.Error.Message)
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
	// Revision is the pane's output revision counter. The activity tracker
	// diffs it for sessions herdr cannot classify (agent_status "unknown").
	// Verified live on 0.7.3: it moves only while a client renders the pane;
	// a headless server holds it at 0.
	Revision uint64 `json:"revision"`
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
//
//nolint:unparam // source documents herdr's read API; every current caller snapshots the visible screen
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
	var lastErr error
	// Paste phase: re-paste only while the screen provably didn't take it. A
	// failed verification read is not proof — re-verify on the next attempt
	// rather than re-pasting into a box that may already hold the text.
	pasted := false
	var pasteScreen string // pre-submit baseline: the screen with the paste in the box
	needPaste := true
	var before string
	for attempt := 0; attempt < submitMaxAttempts && !pasted; attempt++ {
		if needPaste {
			b, rerr := c.read(ctx, name, "visible", 0)
			if rerr != nil {
				lastErr = rerr // transient read failure; retry within the bound
			}
			before = strings.TrimSpace(b)
			if err := c.paneRun(ctx, paneID, text); err != nil {
				return err
			}
			needPaste = false
		}
		time.Sleep(submitSettleDelay) // let the paste commit before we check it
		after, rerr := c.read(ctx, name, "visible", 0)
		if rerr != nil {
			lastErr = rerr // can't verify this paste; re-verify next attempt
			continue
		}
		if strings.TrimSpace(after) != before {
			pasted = true
			pasteScreen = strings.TrimSpace(after)
		} else {
			needPaste = true // provably swallowed — pane not input-ready yet; re-paste
		}
	}
	if !pasted {
		if lastErr != nil {
			return fmt.Errorf("herdr deliverNudge: %q paste never landed after %d attempts: %w", name, submitMaxAttempts, lastErr)
		}
		return fmt.Errorf("herdr deliverNudge: %q paste never landed after %d attempts", name, submitMaxAttempts)
	}
	// Submit phase: retry the Enter only — a re-paste here duplicates the turn.
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
		cur, rerr := c.read(ctx, name, "visible", 0)
		if rerr != nil {
			lastErr = rerr
			continue // can't tell whether the box consumed it; another Enter is safe
		}
		if strings.TrimSpace(cur) != pasteScreen {
			return nil // box consumed the paste → submit landed; agent hasn't visibly started yet (hook/first-token latency)
		}
	}
	if lastErr != nil {
		return fmt.Errorf("herdr deliverNudge: %q submit not confirmed after %d attempts: %w", name, submitMaxAttempts, lastErr)
	}
	return fmt.Errorf("herdr deliverNudge: %q still idle after %d attempts (submit unconfirmed)", name, submitMaxAttempts)
}

// submitSettleDelay is how long deliverNudge waits for a `pane run` paste to
// commit in the TUI before each submit Enter and before re-reading agent status.
// A submit that races the paste is swallowed; ~1s clears it with margin even
// under the concurrent boot load of a town-wide restart.
// Var (not const) so tests can shrink it; production code never writes it.
var submitSettleDelay = 1 * time.Second

// submitMaxAttempts bounds each of deliverNudge's two phases independently
// (paste-until-landed, then Enter-until-confirmed): with one settle wait per
// attempt, ~2·submitMaxAttempts·settle is the worst-case latency before
// deliverNudge gives up and returns an error. Sized to cover a slow shell→TUI
// handoff under restart-time load without spinning on a nudge that
// legitimately leaves the agent idle.
// Var (not const) so tests can shrink it; production code never writes it.
var submitMaxAttempts = 5

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

// findWorkspace returns the id of the workspace whose label matches, or "".
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
	for _, w := range wrap.Workspaces {
		if w.Label == label {
			return w.WorkspaceID, nil
		}
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
	for _, t := range wrap.Tabs {
		if t.Label == label {
			return t.TabID, nil
		}
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

// ensurePlacement resolves where an agent's pane should live: it finds or
// creates the per-rig/town workspace, then creates the per-agent tab when
// absent. It returns the tab and its shell pane, prepared with cwd and env.
func (c *client) ensurePlacement(ctx context.Context, wsLabel, tabLabel, cwd string, env map[string]string) (tabID, paneID string, err error) {
	wsID, err := c.findWorkspace(ctx, wsLabel)
	if err != nil {
		return "", "", err
	}
	if wsID == "" {
		_, tabID, paneID, err = c.workspaceCreate(ctx, wsLabel, cwd, env)
		if err != nil {
			return "", "", err
		}
		_ = c.tabRename(ctx, tabID, tabLabel)
		return tabID, paneID, nil
	}
	tabID, err = c.findTab(ctx, wsID, tabLabel)
	if err != nil {
		return "", "", err
	}
	if tabID != "" {
		return "", "", fmt.Errorf("herdr tab %q already exists without a registered agent", tabLabel)
	}
	return c.tabCreate(ctx, wsID, tabLabel, cwd, env)
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
	if c.sockPath != "" {
		return c.sockPath
	}
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
