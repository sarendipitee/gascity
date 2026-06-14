package main

import (
	"context"
	"log"
	"path"
	"strings"
	"time"

	"github.com/gastownhall/gascity/internal/agent"
	"github.com/gastownhall/gascity/internal/beads"
	"github.com/gastownhall/gascity/internal/config"
	"github.com/gastownhall/gascity/internal/session"
	"github.com/gastownhall/gascity/internal/sling"
)

// sessionBeadAssigneeIdentities returns every identifier under which a work
// bead could be assigned to this session: the session bead ID, session_name,
// configured_named_identity, current alias, and any prior aliases preserved
// in alias_history. Pool polecat aliases (e.g. "nux") are first-class
// assignment identities, so leaving them out of orphan-detection causes
// in-progress work to be reset under a live owner — see the
// SkipsLiveSessionAssignedByAlias regression tests.
func sessionBeadAssigneeIdentities(sb beads.Bead) []string {
	identities := make([]string, 0, 5)
	if id := strings.TrimSpace(sb.ID); id != "" {
		identities = append(identities, id)
	}
	if sn := strings.TrimSpace(sb.Metadata["session_name"]); sn != "" {
		identities = append(identities, sn)
	}
	if ni := strings.TrimSpace(sb.Metadata["configured_named_identity"]); ni != "" {
		identities = append(identities, ni)
	}
	if al := strings.TrimSpace(sb.Metadata["alias"]); al != "" {
		identities = append(identities, al)
	}
	for _, prior := range session.AliasHistory(sb.Metadata) {
		if prior = strings.TrimSpace(prior); prior != "" {
			identities = append(identities, prior)
		}
	}
	return identities
}

type releasedPoolAssignment struct {
	ID    string
	Index int
}

// PoolSessionName derives the tmux session name for a pool worker session.
// Format: {basename(template)}-{beadID} (e.g., "claude-mc-xyz").
// Named sessions with an alias use the alias instead.
func PoolSessionName(template, beadID string) string {
	base := path.Base(template)
	return agent.SanitizeQualifiedNameForSession(base) + "-" + beadID
}

// GCSweepSessionBeads closes open session beads that have no remaining
// open/in-progress work beads anywhere — primary store OR any attached
// rig store. Work-bead assignment is verified by a live cross-store
// query inside closeSessionBeadIfUnassigned, so the caller does not
// pass a work snapshot — that pattern was retired to prevent pre-close
// tick snapshots from poisoning close decisions. Returns the IDs of
// session beads that were closed.
func GCSweepSessionBeads(store beads.Store, rigStores map[string]beads.Store, sessionBeads []beads.Bead) []string {
	var closed []string
	for _, sb := range sessionBeads {
		if sb.Status == "closed" {
			continue
		}
		if !closeSessionBeadIfUnassigned(store, rigStores, nil, sb, "gc_swept", time.Now().UTC(), nil) {
			continue
		}
		closed = append(closed, sb.ID)
	}
	return closed
}

// releaseOrphanedPoolAssignmentsWhenSnapshotsComplete skips orphan release
// unless both the assigned-work and open-session snapshots are complete.
func releaseOrphanedPoolAssignmentsWhenSnapshotsComplete(
	store beads.Store,
	cfg *config.City,
	cityPath string,
	openSessionBeads []beads.Bead,
	result DesiredStateResult,
	rigStores map[string]beads.Store,
) []releasedPoolAssignment {
	// Partial input snapshots can make active work look orphaned for this
	// tick only: missing work affects drain decisions, and missing sessions
	// affects assigned-work orphan release.
	if result.snapshotQueryPartial() {
		return nil
	}
	return releaseOrphanedPoolAssignments(store, cfg, cityPath, openSessionBeads, result.AssignedWorkBeads, result.AssignedWorkStores, result.AssignedWorkStoreRefs, rigStores)
}

// releaseOrphanedPoolAssignments reopens active pool-routed work whose
// assignee no longer maps to any open session bead. This also recovers
// pool-routed work left in_progress with no assignee, which cannot be claimed
// again until it is moved back to open.
func releaseOrphanedPoolAssignments(
	store beads.Store,
	cfg *config.City,
	cityPath string,
	openSessionBeads []beads.Bead,
	assignedWorkBeads []beads.Bead,
	assignedWorkStores []beads.Store,
	assignedWorkStoreRefs []string,
	rigStores map[string]beads.Store,
) []releasedPoolAssignment {
	if store == nil || cfg == nil || len(assignedWorkBeads) == 0 {
		return nil
	}
	storeAware := len(assignedWorkStores) > 0
	if storeAware && len(assignedWorkStores) != len(assignedWorkBeads) {
		log.Printf("releaseOrphanedPoolAssignments: assigned work/store length mismatch: work=%d stores=%d", len(assignedWorkBeads), len(assignedWorkStores))
	}
	storeRefAware := len(assignedWorkStoreRefs) == len(assignedWorkBeads)
	if len(assignedWorkStoreRefs) > 0 && !storeRefAware {
		log.Printf("releaseOrphanedPoolAssignments: assigned work/store-ref length mismatch: work=%d storeRefs=%d", len(assignedWorkBeads), len(assignedWorkStoreRefs))
	}

	openIdentifiers := makeOpenSessionStoreRefIndex(cityPath, cfg, openSessionBeads, storeRefAware)
	legacyOpenIdentifiers := make(map[string]struct{}, len(openSessionBeads)*5)
	for _, sb := range openSessionBeads {
		if sb.Status == "closed" {
			continue
		}
		for _, id := range sessionBeadAssigneeIdentities(sb) {
			legacyOpenIdentifiers[id] = struct{}{}
		}
	}

	var released []releasedPoolAssignment
	for i, wb := range assignedWorkBeads {
		isDeferred := wb.Status == "deferred"
		if wb.Status != "open" && wb.Status != "in_progress" && !isDeferred {
			continue
		}
		assignee := strings.TrimSpace(wb.Assignee)
		template := routedToOrLegacyWorkflowTarget(wb)
		if template == "" {
			continue
		}
		agentCfg := findAgentByTemplate(cfg, template)
		if agentCfg == nil || !agentCfg.SupportsGenericEphemeralSessions() {
			continue
		}
		if assignee == "" {
			// Deferred pool-routed beads with no assignee should always be
			// undeferred — no session owns them, so there is nothing to check.
			if !isDeferred {
				if wb.Status != "in_progress" {
					continue
				}
			}
		} else {
			workStoreRef := ""
			if storeRefAware {
				workStoreRef = assignedWorkStoreRefs[i]
			}
			if openSessionOwnsWork(legacyOpenIdentifiers, openIdentifiers, assignee, workStoreRef, storeRefAware) {
				continue
			}
			if assigneePreservesNamedSessionRoute(cfg, cityPath, template, assignee, workStoreRef, storeRefAware) {
				continue
			}
			if liveOpenSessionAssignmentExists(store, assignee) {
				continue
			}
		}

		var ownerStore beads.Store
		if storeAware {
			if i >= len(assignedWorkStores) || assignedWorkStores[i] == nil {
				log.Printf("releaseOrphanedPoolAssignments: missing owner store for assigned work %q at index %d", wb.ID, i)
				continue
			}
			ownerStore = assignedWorkStores[i]
		} else {
			ownerStore = storeForPoolAssignment(cfg, store, rigStores, wb)
			if ownerStore == nil {
				continue
			}
		}
		if !isDeferred {
			if !liveWorkAssignmentStillReleasable(ownerStore, wb.ID, wb.Status, assignee) {
				continue
			}
		}
		allowsRelease, clearDetached := detachedProbeAllowsOrphanRelease(wb)
		if !allowsRelease {
			continue
		}
		if !releaseOrphanedPoolAssignmentFull(ownerStore, wb.ID, clearDetached, isDeferred) {
			continue
		}
		released = append(released, releasedPoolAssignment{ID: wb.ID, Index: i})
	}

	// Second pass: recover stranded ephemeral wisps assigned to dead sessions
	// that have no gc.routed_to (e.g. patrol formulas that used --assignee=$GC_AGENT
	// instead of setting gc.routed_to). Without gc.routed_to these beads are
	// invisible to both the main pass above (which requires a route target) and
	// the pool demand check (which skips assigned beads). Recover the route target
	// from the dead session's template metadata so the next reconcile tick can
	// count them as demand and spawn a replacement session.
	released = releaseStrandedEphemeralWisps(store, cfg, cityPath, openIdentifiers, legacyOpenIdentifiers, storeRefAware, assignedWorkBeads, assignedWorkStores, assignedWorkStoreRefs, storeAware, rigStores, released)

	return released
}

// releaseStrandedEphemeralWisps recovers ephemeral work beads that are assigned
// to a dead pool session but have no gc.routed_to metadata. Such beads are
// invisible to the main orphan-release pass (which requires gc.routed_to to
// identify the target template) and to the pool demand check (which skips
// assigned beads entirely). This happens when a patrol formula assigns the
// next iteration wisp by session name ($GC_AGENT) instead of setting
// gc.routed_to to the stable template alias.
//
// For each stranded ephemeral bead, we look up the dead session's bead to
// recover the template, then set gc.routed_to and release the bead so the
// next pool demand check can find it and spawn a replacement session.
func releaseStrandedEphemeralWisps(
	store beads.Store,
	cfg *config.City,
	cityPath string,
	openIdentifiers map[string]map[string]struct{},
	legacyOpenIdentifiers map[string]struct{},
	storeRefAware bool,
	assignedWorkBeads []beads.Bead,
	assignedWorkStores []beads.Store,
	assignedWorkStoreRefs []string,
	storeAware bool,
	rigStores map[string]beads.Store,
	alreadyReleased []releasedPoolAssignment,
) []releasedPoolAssignment {
	// Build a set of already-released IDs to avoid double-release.
	alreadyReleasedIDs := make(map[string]struct{}, len(alreadyReleased))
	for _, r := range alreadyReleased {
		alreadyReleasedIDs[r.ID] = struct{}{}
	}

	// Lazy cache: dead-assignee → template, populated on first stranded bead.
	// Nil means not yet built; empty string means "not found" for a given assignee.
	var deadSessionTemplateCache map[string]string

	storeRefAwareForStranded := storeRefAware && len(assignedWorkStoreRefs) == len(assignedWorkBeads)

	released := alreadyReleased
	for i, wb := range assignedWorkBeads {
		if !wb.Ephemeral {
			continue
		}
		if strings.TrimSpace(wb.Metadata["gc.routed_to"]) != "" {
			continue // handled by main pass
		}
		assignee := strings.TrimSpace(wb.Assignee)
		if assignee == "" {
			continue // unassigned — not our concern here
		}
		if wb.Status != "open" && wb.Status != "in_progress" {
			continue
		}
		if _, done := alreadyReleasedIDs[wb.ID]; done {
			continue
		}
		// Check if the assignee is still live.
		workStoreRef := ""
		if storeRefAwareForStranded {
			workStoreRef = assignedWorkStoreRefs[i]
		}
		if openSessionOwnsWork(legacyOpenIdentifiers, openIdentifiers, assignee, workStoreRef, storeRefAwareForStranded) {
			continue
		}
		if liveOpenSessionAssignmentExists(store, assignee) {
			continue
		}

		// Build the dead-session template cache lazily.
		if deadSessionTemplateCache == nil {
			deadSessionTemplateCache = buildDeadSessionTemplateCache(store)
		}
		template, ok := deadSessionTemplateCache[assignee]
		if !ok || template == "" {
			continue
		}
		agentCfg := findAgentByTemplate(cfg, template)
		if agentCfg == nil || !agentCfg.SupportsGenericEphemeralSessions() {
			continue
		}

		var ownerStore beads.Store
		if storeAware {
			if i >= len(assignedWorkStores) || assignedWorkStores[i] == nil {
				log.Printf("releaseStrandedEphemeralWisps: missing owner store for %q at index %d", wb.ID, i)
				continue
			}
			ownerStore = assignedWorkStores[i]
		} else {
			// We don't have gc.routed_to yet, so storeForPoolAssignment won't find
			// a rig store by route. Use the city store as the fallback; the bead
			// likely lives in the city store if it was created by a city agent.
			ownerStore = store
		}

		// Set gc.routed_to so pool demand check can discover the released bead.
		if err := ownerStore.SetMetadata(wb.ID, "gc.routed_to", template); err != nil {
			log.Printf("releaseStrandedEphemeralWisps: setting gc.routed_to on %s: %v", wb.ID, err)
			continue
		}
		if !releaseOrphanedPoolAssignment(ownerStore, wb.ID, false) {
			continue
		}
		log.Printf("releaseStrandedEphemeralWisps: recovered stranded ephemeral wisp %s (assignee=%s template=%s)", wb.ID, assignee, template)
		released = append(released, releasedPoolAssignment{ID: wb.ID, Index: i})
		alreadyReleasedIDs[wb.ID] = struct{}{}
	}
	return released
}

// buildDeadSessionTemplateCache queries all session beads (including closed)
// and returns a map from each session identity to its template name. Used to
// recover the route target for stranded ephemeral wisps assigned by session
// name to a now-dead session.
func buildDeadSessionTemplateCache(store beads.Store) map[string]string {
	if store == nil {
		return nil
	}
	sessionBeadsAll, err := store.List(beads.ListQuery{
		Label:         sessionBeadLabel,
		IncludeClosed: true,
		Limit:         500,
	})
	if err != nil {
		log.Printf("releaseStrandedEphemeralWisps: listing session beads: %v", err)
		return nil
	}
	cache := make(map[string]string, len(sessionBeadsAll)*4)
	for _, sb := range sessionBeadsAll {
		template := retiredSessionFallbackRoute(sb)
		if template == "" {
			continue
		}
		for _, id := range sessionBeadAssigneeIdentities(sb) {
			if id != "" {
				cache[id] = template
			}
		}
	}
	return cache
}

func detachedProbeAllowsOrphanRelease(wb beads.Bead) (bool, bool) {
	spec := strings.TrimSpace(wb.Metadata[detachedProbeMetadataKey])
	if spec == "" {
		clearDetachedProbeErrorCount(wb.ID)
		return true, false
	}

	result := probeDetachedWork(context.Background(), spec)
	switch result.Status {
	case detachedProbeAlive:
		clearDetachedProbeErrorCount(wb.ID)
		log.Printf("releaseOrphanedPoolAssignments: skipping release: detached probe alive for %s: %s", wb.ID, spec)
		return false, false
	case detachedProbeDead:
		clearDetachedProbeErrorCount(wb.ID)
		log.Printf("releaseOrphanedPoolAssignments: releasing %s: detached probe dead: %s", wb.ID, spec)
		return true, true
	case detachedProbeError, detachedProbeTimeout:
		count := incrementDetachedProbeErrorCount(wb.ID)
		if count < detachedProbeErrorThreshold {
			log.Printf("releaseOrphanedPoolAssignments: detached probe %s for %s: %v (error %d/%d)", result.Status, wb.ID, result.Err, count, detachedProbeErrorThreshold)
			return false, false
		}
		clearDetachedProbeErrorCount(wb.ID)
		log.Printf("releaseOrphanedPoolAssignments: releasing %s: detached probe %s after %d errors: %v", wb.ID, result.Status, count, result.Err)
		return true, true
	default:
		count := incrementDetachedProbeErrorCount(wb.ID)
		if count < detachedProbeErrorThreshold {
			log.Printf("releaseOrphanedPoolAssignments: detached probe unknown result for %s: %q (error %d/%d)", wb.ID, result.Status, count, detachedProbeErrorThreshold)
			return false, false
		}
		clearDetachedProbeErrorCount(wb.ID)
		return true, true
	}
}

func clearDetachedProbeMetadata(store beads.Store, id string) {
	if store == nil || id == "" {
		return
	}
	if err := store.SetMetadata(id, detachedProbeMetadataKey, ""); err != nil {
		log.Printf("clearing detached probe metadata for %s: %v", id, err)
	}
}

const unresolvedOpenSessionStoreRef = "\x00unresolved"

func makeOpenSessionStoreRefIndex(cityPath string, cfg *config.City, openSessionBeads []beads.Bead, storeRefAware bool) map[string]map[string]struct{} {
	index := make(map[string]map[string]struct{}, len(openSessionBeads)*5)
	if !storeRefAware {
		return index
	}
	for _, sb := range openSessionBeads {
		if sb.Status == "closed" {
			continue
		}
		storeRef, ok := assignedWorkStoreRefForSession(cityPath, cfg, sb)
		if !ok {
			storeRef = unresolvedOpenSessionStoreRef
		}
		for _, id := range sessionBeadAssigneeIdentities(sb) {
			addOpenSessionStoreRef(index, id, storeRef)
		}
	}
	return index
}

func addOpenSessionStoreRef(index map[string]map[string]struct{}, identifier, storeRef string) {
	identifier = strings.TrimSpace(identifier)
	if identifier == "" {
		return
	}
	refs := index[identifier]
	if refs == nil {
		refs = make(map[string]struct{}, 1)
		index[identifier] = refs
	}
	refs[storeRef] = struct{}{}
}

func openSessionOwnsWork(legacyIdentifiers map[string]struct{}, scopedIdentifiers map[string]map[string]struct{}, assignee, workStoreRef string, storeRefAware bool) bool {
	if !storeRefAware {
		_, ok := legacyIdentifiers[assignee]
		return ok
	}
	refs := scopedIdentifiers[assignee]
	if refs == nil {
		return false
	}
	if _, ok := refs[unresolvedOpenSessionStoreRef]; ok {
		return true
	}
	_, ok := refs[workStoreRef]
	return ok
}

func storeForPoolAssignment(cfg *config.City, cityStore beads.Store, rigStores map[string]beads.Store, wb beads.Bead) beads.Store {
	if cfg == nil || len(rigStores) == 0 {
		return cityStore
	}
	routed := routedToOrLegacyWorkflowTarget(wb)
	if routed != "" {
		if slash := strings.IndexByte(routed, '/'); slash > 0 {
			if store := rigStores[routed[:slash]]; store != nil {
				return store
			}
		}
	}
	idPrefix := sling.BeadPrefixForCity(cfg, wb.ID)
	for _, rig := range cfg.Rigs {
		if strings.EqualFold(idPrefix, rig.EffectivePrefix()) {
			if store := rigStores[rig.Name]; store != nil {
				return store
			}
		}
	}
	return cityStore
}

func isRecoverableUnassignedInProgressPoolWork(cfg *config.City, wb beads.Bead) bool {
	if wb.Status != "in_progress" || strings.TrimSpace(wb.Assignee) != "" {
		return false
	}
	template := routedToOrLegacyWorkflowTarget(wb)
	if template == "" {
		return false
	}
	agentCfg := findAgentByTemplate(cfg, template)
	return agentCfg != nil && agentCfg.SupportsGenericEphemeralSessions()
}

func releaseOrphanedPoolAssignment(store beads.Store, id string, clearDetached bool) bool {
	return releaseOrphanedPoolAssignmentFull(store, id, clearDetached, false)
}

func releaseOrphanedPoolAssignmentFull(store beads.Store, id string, clearDetached, clearDefer bool) bool {
	if store == nil || id == "" {
		return false
	}
	opts := beads.UpdateOpts{
		Assignee:   stringPtr(""),
		Status:     stringPtr("open"),
		ClearDefer: clearDefer,
	}
	if clearDetached {
		opts.Metadata = map[string]string{detachedProbeMetadataKey: ""}
	}
	if err := store.Update(id, opts); err != nil {
		log.Printf("releaseOrphanedPoolAssignments: releasing orphaned pool assignment %s: %v", id, err)
		return false
	}
	return true
}

func liveOpenSessionAssignmentExists(store beads.Store, assignee string) bool {
	assignee = strings.TrimSpace(assignee)
	if store == nil || assignee == "" {
		return false
	}
	if liveSessionBeadExistsByIdentity(store, assignee) {
		return true
	}
	// NOTE: this call site intentionally keeps a label-only query — not
	// the Type+Label union from session.ListAllSessionBeads. The
	// orphan-release tests (TestReleaseOrphanedPoolAssignments_*) set up
	// city session beads with Type=session but no gc:session label and
	// assert that rig work pointing at a session_name only reachable via
	// the typed bead IS released. Switching this query to the union
	// would surface those typed beads as "live" and cause the work to
	// be skipped instead of released, regressing
	// ReopensRigStoreMissingPoolAssignee and
	// ReleasesRigWorkAssignedToUnreachableOpenSession. The label-loss
	// bug this PR is fixing manifests in the snapshot/list/reconciler
	// paths; orphan release continues to treat the label as the
	// authoritative liveness signal.
	sessions, err := store.List(beads.ListQuery{
		Label: sessionBeadLabel,
		Live:  true,
	})
	if err != nil {
		log.Printf("releaseOrphanedPoolAssignments: live session validation failed for assignee %q: %v", assignee, err)
		return true
	}
	for _, sb := range sessions {
		if sb.Status == "closed" || !isSessionBead(sb) {
			continue
		}
		for _, id := range sessionBeadAssigneeIdentities(sb) {
			if assignee == id {
				return true
			}
		}
	}
	return false
}

func liveSessionBeadExistsByIdentity(store beads.Store, assignee string) bool {
	for _, id := range directSessionBeadIDCandidates(assignee) {
		sb, err := store.Get(id)
		if err != nil {
			continue
		}
		if sb.Status == "closed" || !isSessionBead(sb) {
			continue
		}
		for _, candidate := range sessionBeadAssigneeIdentities(sb) {
			if assignee == candidate {
				return true
			}
		}
	}
	return false
}

func directSessionBeadIDCandidates(assignee string) []string {
	assignee = strings.TrimSpace(assignee)
	if assignee == "" {
		return nil
	}
	candidates := []string{assignee}
	if idx := strings.LastIndex(assignee, "-mc-"); idx >= 0 {
		candidates = append(candidates, assignee[idx+1:])
	}
	return candidates
}

// liveWorkAssignmentStillReleasable confirms the snapshot is not stale
// before clearing assignee. The expectedStatus must match the snapshot
// status the caller observed: if the bead has since transitioned (e.g. a
// concurrent claim moved open→in_progress, or another release moved
// in_progress→open) the snapshot's release decision is no longer safe.
// Open status is required for the issue #2793 path — graph.v2 step
// beads stuck on a dead session's long-form assignee are status=open,
// not in_progress.
func liveWorkAssignmentStillReleasable(store beads.Store, id, expectedStatus, assignee string) bool {
	id = strings.TrimSpace(id)
	expectedStatus = strings.TrimSpace(expectedStatus)
	if store == nil || id == "" || expectedStatus == "" {
		return false
	}
	work, err := store.List(beads.ListQuery{
		Status:   expectedStatus,
		Live:     true,
		TierMode: beads.TierBoth,
	})
	if err != nil {
		log.Printf("releaseOrphanedPoolAssignments: live work validation failed for %q: %v", id, err)
		return false
	}
	for _, wb := range work {
		if wb.ID != id {
			continue
		}
		return strings.TrimSpace(wb.Assignee) == strings.TrimSpace(assignee)
	}
	return false
}

func assigneePreservesNamedSessionRoute(cfg *config.City, cityPath, template, assignee, workStoreRef string, storeRefAware bool) bool {
	if cfg == nil {
		return false
	}
	spec, ok := findNamedSessionSpec(cfg, cfg.EffectiveCityName(), assignee)
	if !ok {
		return false
	}
	if namedSessionBackingTemplate(spec) != template {
		return false
	}
	if !storeRefAware {
		return true
	}
	return assignedWorkStoreRefForAgent(cityPath, cfg, spec.Agent) == workStoreRef
}

func stringPtr(s string) *string { return &s }
