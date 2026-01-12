# Session Workflow Quick Reference

## Development Session Phases

1. **Planning** - Collaborative exploration, reach alignment
2. **Plan Presentation** - Outline for approval (or use plan mode)
3. **Implementation Guide** - Create at `_context/[session-id]-[title].md`
4. **OpenAPI Maintenance** - Prepare openapi.go files before implementation
5. **Developer Execution** - Follow guide
6. **Validation** - Add tests, execute until passing
7. **Documentation** - Add godoc comments
8. **Closeout** - Summary, archive guide, update docs

## Maintenance Session Phases

1. **Planning** - Review scope and dependencies
2. **Execution** - Follow plan, coordinate releases if needed
3. **Validation** - Validate changes, adjust tests
4. **Closeout** - Archive, summary, update PROJECT.md

## Implementation Guide Conventions

- Session ID format: `01a`, `01b`, `02a` (milestone + letter)
- Maintenance ID format: `mt01`, `mt02`
- Code blocks: NO comments, NO tests, NO OpenAPI file contents
- Existing files: Show incremental changes only
- New files: Provide complete implementation
- Structure steps from lowest to highest dependency

## Session Closeout Checklist

1. Generate summary at `_context/sessions/[session-id]-[title].md`
2. Archive guide to `_context/sessions/.archive/`
3. Update context architecture if patterns changed (rules, skills)
4. Update PROJECT.md session status
5. Update README.md only for critical user-facing changes

## Testing Success Criteria

- Happy paths covered
- Security paths covered (validation, injection protection)
- Error types distinguishable
- Integration points verified
