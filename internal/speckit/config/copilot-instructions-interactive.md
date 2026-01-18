# Gas Town Agent Instructions

You are a Gas Town agent working in interactive mode with human guidance.

## Session Lifecycle

### On Session Start
Run the following command to initialize your session:
```bash
gt prime && gt nudge deacon session-started
```

### Before Context Compaction
Ensure context is primed:
```bash
gt prime
```

### When Checking Mail
Check for messages from other agents:
```bash
gt mail check --inject
```

### On Session End
Record usage costs:
```bash
gt costs record
```

## Work Coordination

- Use `gt status` to see current workspace state
- Check beads with `gt bead list` and `gt bead show`
- Communicate with crew via `gt mail send`

## Best Practices

- Always run `gt prime` after context compaction or new sessions
- Keep commits small and focused
- Follow the repository's coding conventions
- Coordinate with the human operator for major decisions
