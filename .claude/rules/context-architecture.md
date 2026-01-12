# Context Architecture

## Loading Behavior

| Location | Loading | Purpose |
|----------|---------|---------|
| `.claude/CLAUDE.md` | Always | Role, navigation, project overview |
| `.claude/rules/*.md` | Always | Core conventions, quick references |
| `.claude/skills/*/SKILL.md` | On-demand | Detailed patterns, loaded by relevance |
| `_context/milestones/*.md` | Manual | Milestone-specific architecture |
| `_context/sessions/*.md` | Manual | Session summaries for pattern reference |

## Directory Structure

```
.claude/
├── CLAUDE.md              # Lean navigation (~150-200 lines)
├── rules/                 # Always-loaded conventions
└── skills/                # On-demand domain knowledge
    └── {skill-name}/
        └── SKILL.md

_context/
├── milestones/            # Active milestone architecture docs
│   └── .archive/          # Completed milestone docs
├── sessions/              # Session summaries
│   └── .archive/          # Archived implementation guides
└── prompts/               # Session initialization prompts
```

## Skill Triggering

Skills load based on their `description` field matching:
- **Keywords** in the conversation context
- **File patterns** being edited
- **Use cases** mentioned in the task

## When to Update What

| Change Type | Location |
|-------------|----------|
| New Go pattern | Appropriate skill (go-core, go-http, etc.) |
| New workflow convention | development-methodology skill |
| Command reference | rules/go-commands.md |
| Session workflow change | rules/session-workflow.md |
| Navigation/role change | .claude/CLAUDE.md |
| New domain emerges | Create new skill |

## Adding New Skills

1. Create directory: `.claude/skills/{skill-name}/`
2. Create `SKILL.md` with YAML frontmatter:
   ```yaml
   ---
   name: skill-name
   description: >
     When to use this skill. Include keywords, file patterns, use cases.
   ---
   ```
3. Add sections: When This Skill Applies, Principles, Patterns, Anti-Patterns
