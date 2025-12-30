# Design Decisions

## Architecture Choices

### Single Binary Approach
- **Decision**: Single `main.go` file for MVP
- **Rationale**: Simplicity over modularity for <500 LOC
- **Future**: Can split into packages if it grows

### Stdlib Only
- **Decision**: No external dependencies
- **Rationale**: Easier deployment, no dependency management
- **Trade-off**: More verbose HTTP code, but acceptable

### Manual Token Setup
- **Decision**: User extracts tokens manually
- **Rationale**: Avoid chromedp complexity, one-time 5min task
- **Alternative Considered**: Headless browser automation (rejected as over-engineered)

### State Management
- **Decision**: JSON file with atomic writes
- **Rationale**: Simple, human-readable, reliable
- **Alternative Considered**: SQLite (rejected as overkill)

### Polling Strategy
- **Decision**: 60 second default interval
- **Rationale**: Respects rate limits, fast enough for notifications
- **Configurable**: Users can adjust in config

---

## Implementation Notes

[To be filled during development]
