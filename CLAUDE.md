# CLAUDE.md

This file provides guidance to Claude Code when working with this repository.

## SubAgents
- before beginning a new session, consult the claude-code-guide subagent to 
  understand where to make changes
- when making material changes to the codebase consult the 
  codebase-architect to determine the correct patterns to use.  Have them 
  review the code as well.

## GitHub Issue Workflow

When working on a GitHub issue, follow these steps in order:

1. **Read** the issue and all existing documentation (`docs/*`)
2. **Ask clarifying questions** before proceeding
3. **Plan** — add a section to `enhancements.md` using the template below
4. **Update docs** — revise `docs/design.md`, `docs/architecture.md`, and
   `docs/ux.md` to reflect the planned change
5. **Implement** the change
6. **Verify** — build the project and run tests
7. **Document** — update `README.md` with any new user-facing functionality

### enhancements.md Template

```markdown
## <Feature Name> (`<shortcut or keyword>`)

[issue #N](https://github.com/nick-orton/ghac/issues/N)

### Summary

<One-paragraph description of what the feature does and why.>

---

### Scope

| Screen             | In scope |
| ------------------ | -------- |
| {screen name}      | Yes/No   |

---

### Behaviour Specification

1. <Step-by-step description of the exact behaviour.>
2. <Include edge cases (no match, boundary conditions, etc.).>

#### <Sub-section if needed (e.g. display name used for matching)>

| Screen | Detail |
| ------ | ------ |
| ...    | ...    |

---

### Design Decisions (confirmed)

1. **<Decision topic>** — <rationale>.
2. ...

---
```

## Commands

## Documentation

Documentation on the design and architecture of the app is found in docs/
- These are living documents.  If there are additional features added to the
  code, then the design and architecture should be updated accordingly.


```
docs/
  ├── architecture.md   (Describes the system design and patterns)
  ├── requirements.md   (initial requirements, since superceded by github 
  |                      issues for new features)
  ├── design.md         (Describes how the system should behave)
  ├── plan.md           (Describes the initial implementation plan)
  ├── enhancements.md   (extensions beyond plan.md)
  └── ux.md             (UX standards: colors, layout, typography,
                         component conventions)
README.md               (user-facing documentation)
```
