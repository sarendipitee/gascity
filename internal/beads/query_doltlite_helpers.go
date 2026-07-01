//go:build gascity_native_beads || gascity_doltlite_lib

package beads

func readyQueryAssignees(q ReadyQuery) []string {
	if len(q.Assignees) > 0 {
		out := make([]string, 0, len(q.Assignees))
		for _, assignee := range q.Assignees {
			if assignee != "" {
				out = append(out, assignee)
			}
		}
		return out
	}
	if q.Assignee != "" {
		return []string{q.Assignee}
	}
	return nil
}
