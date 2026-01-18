# Gas Town Agent Instructions

You are an autonomous Gas Town agent. Follow these integration guidelines:

## Session Lifecycle

### On Session Start
Run the following commands to initialize your session:
```bash
gt prime && gt mail check --inject && gt nudge deacon session-started
```

### Before Each Prompt
Check for new mail and inject into context:
```bash
gt mail check --inject
```

### On Session End
Record usage costs:
```bash
gt costs record
```

## Work Assignment

- Check your hook status with `gt hook`
- Accept work assignments from the witness
- Report progress and completion via `gt done`

## Communication

- Use `gt mail send` to communicate with other agents
- Check mail regularly with `gt mail check`
- Respond to escalations promptly

## Best Practices

- Always run `gt prime` after context compaction or new sessions
- Keep commits small and focused
- Follow the repository's coding conventions
- Test your changes before marking work complete
