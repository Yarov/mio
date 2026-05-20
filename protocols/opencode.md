## Mio — Your memory across sessions

You have no memory between conversations. When this one ends, everything vanishes. The only bridge is what you save to Mio.

**Not saving is forgetting.**

### What to save

Save what the code doesn't say — decisions (and why), discoveries, user preferences, progress, surprises.

Structure: `What: / Why: / Where: / Learned:`

### When to save — recognize these moments

**The user is leaving.** "me voy", "bye", "regreso", "luego sigo" — save a summary before responding. This is the most common failure: not saving when the conversation ends.

**You discovered something.** A root cause, unexpected behavior, workaround — save now, not later. Conversations end without warning.

**A decision was made.** Save what you chose AND what you rejected and why.

**The user shared a preference.** Save it. People don't like repeating themselves.

**You've been working a while without saving.** Meaningful progress with no saves = risk.

### When to search

Call `mio.mem_context` with the project name at session start. Search when the user references past work or you're about to make a decision that could conflict with a prior one.

### Types

`bugfix`, `decision`, `architecture`, `discovery`, `pattern`, `config`, `preference`, `learning`, `summary`

### Topic keys

Use `topic_key` for evolving topics so updates replace instead of duplicating.

### Relations

Use `mio.mem_relate` with `supersedes` or `caused_by` to link related memories.
