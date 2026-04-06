package mcp

// ToolGuideMarkdown is returned by mem_tool_guide — kept in one place for agents
// that do not load IDE rules.
const ToolGuideMarkdown = "## Mio — which tool?\n\n" +
	"**Start of turn (resume context)** → `mem_context` (omit `project` to use repo folder name from cwd) or `mem_surface` with the user’s prompt.\n\n" +
	"**Find something specific** → `mem_search` (add `include_full: true` if you need full text in one shot for a small result set). Then `mem_get_observation(id)` for authoritative content when previews are not enough.\n\n" +
	"**Richer ranking / agent mix** → `mem_enhanced_search` (same `include_full` pattern).\n\n" +
	"**Knowledge spread across repos** → `mem_cross_project`.\n\n" +
	"**Save durable intent** → `mem_save` (use `topic_key`; `mem_suggest_topic_key` helps).\n\n" +
	"**Link decisions** → `mem_relate` / `mem_relations` / `mem_graph`.\n\n" +
	"**Maintain** → `mem_consolidate`, `mem_summarize`, `mem_gc` (occasional housekeeping).\n\n" +
	"**Sessions** → `mem_session_start` / `mem_session_end` / `mem_session_summary` (top-level agent only). If `MIO_SUBAGENT` is set, these are blocked unless `force: true`.\n\n" +
	"**Project names** → Hyphens, case, and underscores are ignored for filters (`my-app` matches `MyApp`).\n"
