package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gastownhall/gascity/internal/beadmeta"
	"github.com/gastownhall/gascity/internal/beads"
	"github.com/gastownhall/gascity/internal/extmsg"
)

// wispStepInjectionContent resolves the agent's current in-progress formula
// step bead and returns it formatted as a <system-reminder> block, or "" if
// none is found or any error occurs. Designed for best-effort use in hook
// injection paths — callers must never fail hard on an empty return.
//
// Store priority: if GC_RIG_ROOT is set the rig store is queried (where
// rig-scoped polecat work beads live), otherwise the city store at cityPath.
// When cityPath is empty the function falls back to GC_CITY from the env.
func wispStepInjectionContent(cityPath, invocationAgent string) string {
	effective := cityPath
	if effective == "" {
		effective = strings.TrimSpace(os.Getenv("GC_CITY"))
	}
	store := openWispStepStore(effective)
	if store == nil {
		return ""
	}
	assignees := wispStepAssignees(invocationAgent)
	if len(assignees) == 0 {
		return ""
	}
	b, err := resolveActiveWispStep(store, assignees)
	if err != nil || b == nil {
		return ""
	}
	return formatWispStepReminder(b)
}

// openWispStepStore opens the bead store to query for active wisp steps.
// If GC_RIG_ROOT is set it opens that rig's store (where rig-scoped polecat
// work lives); otherwise it opens the city store at cityPath.
// Returns nil on any error — callers treat nil as "no store available".
func openWispStepStore(cityPath string) beads.Store {
	if rigRoot := strings.TrimSpace(os.Getenv("GC_RIG_ROOT")); rigRoot != "" {
		store, err := openStoreAtForCity(rigRoot, cityPath)
		if err == nil {
			return store
		}
	}
	if cityPath == "" {
		return nil
	}
	store, err := openCityStoreAt(cityPath)
	if err != nil {
		return nil
	}
	return store
}

// wispStepAssignees returns the deduped set of identity strings to match
// against bead assignees. The invocation agent takes precedence, followed by
// GC_ALIAS, GC_SESSION_NAME, and GC_SESSION_ID.
func wispStepAssignees(invocationAgent string) []string {
	seen := make(map[string]bool)
	var out []string
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v != "" && !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	add(invocationAgent)
	add(os.Getenv("GC_ALIAS"))
	add(os.Getenv("GC_SESSION_NAME"))
	add(os.Getenv("GC_SESSION_ID"))
	return out
}

// resolveActiveWispStep returns the agent's current formula step bead.
//
// Resolution follows assignee priority. For each identity, prefer its directly
// assigned in-progress molecule/wisp root; only when absent follow that same
// identity's molecule_id bridge. Resolve the selected root to its in-progress
// step child, or fall back to its entry step. Only when an identity has neither
// root pathway does resolution continue to the next identity.
//
// Returns nil, nil when no bead can be resolved. Never returns an error for
// not-found conditions — callers treat nil as "nothing to inject".
func resolveActiveWispStep(store beads.Store, assignees []string) (*beads.Bead, error) {
	if store == nil || len(assignees) == 0 {
		return nil, nil
	}

	for _, assignee := range assignees {
		root, err := resolveActiveMolecule(store, assignee)
		if err != nil {
			return nil, err
		}
		if root == nil {
			root = resolveMoleculeRootViaBridge(store, assignee)
		}
		if root == nil {
			continue
		}

		step, stepErr := resolveInProgressStepChild(store, root.ID)
		if stepErr != nil {
			log.Printf("wisp step inject: error resolving in-progress step children for molecule %s: %v", root.ID, stepErr)
			return nil, nil
		}
		if step != nil {
			return step, nil
		}

		log.Printf("wisp step inject: no in-progress step for molecule %s; resolving entry step", root.ID)
		return resolveEntryStepChild(store, root.ID)
	}

	return nil, nil
}

// resolveActiveMolecule returns the agent's newest in-progress molecule or wisp
// root. On identical timestamps, molecule roots retain legacy preference over
// wisp roots. Returns nil, nil when none is found.
func resolveActiveMolecule(store beads.Store, assignee string) (*beads.Bead, error) {
	var best beads.Bead
	found := false
	candidateCount := 0

	for _, molType := range []string{"molecule", "wisp"} {
		results, err := store.List(beads.ListQuery{
			Status:   "in_progress",
			Type:     molType,
			Assignee: assignee,
			TierMode: beads.TierBoth,
		})
		if err != nil {
			return nil, fmt.Errorf("listing in-progress %s beads: %w", molType, err)
		}

		candidateCount += len(results)
		for _, candidate := range results {
			if !found || candidate.UpdatedAt.After(best.UpdatedAt) {
				best = candidate
				found = true
			}
		}
	}

	if !found {
		return nil, nil
	}
	if candidateCount > 1 {
		log.Printf("wisp step inject: %d in-progress molecule/wisp beads found; using most recent", candidateCount)
	}
	return &best, nil
}

// resolveMoleculeRootViaBridge finds the molecule root reachable from an
// attached (v1) source work bead. Attached formulas route only the source bead
// and stamp its molecule_id metadata with the (unrouted, unassigned) molecule
// root, so resolveActiveMolecule — which filters molecule roots by assignee —
// never matches. This bridges from the routed, assignee-owned source bead to
// its root via the molecule_id metadata key.
//
// Returns nil on any error or when no bridge bead is found — callers treat nil
// as "no bridge available" and fall through to the legacy path.
func resolveMoleculeRootViaBridge(store beads.Store, assignee string) *beads.Bead {
	results, err := store.List(beads.ListQuery{
		Status:   "in_progress",
		Assignee: assignee,
		TierMode: beads.TierBoth,
	})
	if err != nil {
		return nil
	}
	var bestRoot *beads.Bead
	var bestSource *beads.Bead
	for i := range results {
		rootID := strings.TrimSpace(results[i].Metadata[beadmeta.MoleculeIDMetadataKey])
		if rootID == "" {
			continue
		}
		root, err := store.Get(rootID)
		if err != nil {
			log.Printf("wisp step inject: molecule_id %q on bead %s did not resolve: %v", rootID, results[i].ID, err)
			continue
		}
		if root.Status != "in_progress" || (root.Type != "molecule" && root.Type != "wisp") {
			continue
		}
		if bestSource == nil || results[i].UpdatedAt.After(bestSource.UpdatedAt) {
			bestRoot = &root
			bestSource = &results[i]
		}
	}
	if bestRoot != nil {
		return bestRoot
	}
	return nil
}

// resolveInProgressStepChild returns the in-progress type=step child of moleculeID.
// When multiple are found, the most recently updated one is returned.
func resolveInProgressStepChild(store beads.Store, moleculeID string) (*beads.Bead, error) {
	results, err := store.List(beads.ListQuery{
		Status:   "in_progress",
		Type:     "step",
		ParentID: moleculeID,
		TierMode: beads.TierBoth,
	})
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		return nil, nil
	}
	if len(results) > 1 {
		ids := make([]string, len(results))
		for i, r := range results {
			ids[i] = r.ID
		}
		log.Printf("wisp step inject: %d in-progress steps for molecule %s (%s); using most recent", len(results), moleculeID, strings.Join(ids, ", "))
	}
	best := results[0]
	for _, r := range results[1:] {
		if r.UpdatedAt.After(best.UpdatedAt) {
			best = r
		}
	}
	return &best, nil
}

// resolveEntryStepChild returns the first open type=step child of moleculeID.
// This is the deterministic fallback when no step is in-progress: the formula's
// entry position — where execution should (re)start.
func resolveEntryStepChild(store beads.Store, moleculeID string) (*beads.Bead, error) {
	results, err := store.List(beads.ListQuery{
		Status:   "open",
		Type:     "step",
		ParentID: moleculeID,
		TierMode: beads.TierBoth,
		Limit:    1,
		Sort:     beads.SortCreatedAsc,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving entry step for molecule %s: %w", moleculeID, err)
	}
	if len(results) == 0 {
		return nil, nil
	}
	b := results[0]
	return &b, nil
}

// formatWispStepReminder formats a formula step bead as a <system-reminder>
// block for injection into agent context.
func formatWispStepReminder(b *beads.Bead) string {
	title := extmsg.SanitizeForSystemReminder(strings.TrimSpace(b.Title))
	desc := extmsg.SanitizeForSystemReminder(strings.TrimSpace(b.Description))
	return fmt.Sprintf(
		"<system-reminder>\nYour current active work assignment:\n\n## %s (%s)\n\n%s\n</system-reminder>\n",
		title, b.ID, desc,
	)
}
