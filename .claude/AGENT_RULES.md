# Agent Rules — Non-Negotiable

These rules apply to **every session, without exception**. Violation of any rule is a blocker.

---

## Session Start — Mandatory Before Any Action

1. Read `HOW_TO_RESUME.md` — follow the numbered protocol exactly
2. Read `state/CURRENT_STATUS.md` — understand the exact current state
3. Read `state/TASK_QUEUE.md` — identify the next task
4. Read `CODING_STANDARDS.md` — internalize conventions before writing any code
5. Read `SECURITY_STANDARDS.md` — internalize all security requirements
6. Identify the active environment before running any command — consult `ENVIRONMENT_GUIDE.md` if in doubt
7. Confirm the working environment is functional before writing any code

---

## During Implementation

- **Scope lock:** Never make changes outside the scope of the current task
- **No silent overwrites:** Never delete or overwrite existing files without explicit instruction
- **Dependency guard:** Never introduce a new dependency, change a schema, or make an architectural decision without surfacing the proposal to the user and receiving explicit confirmation first
- **Zero-regression rule:** Existing passing tests must remain passing after any change — run the full suite before and after
- **Pattern consistency:** Always follow the patterns in `CODING_STANDARDS.md` — do not introduce new patterns without logging them in `state/DECISIONS_LOG.md`

---

## Security Rules — Non-Negotiable

- **No secrets in code:** Never store, log, or expose secrets, tokens, or credentials in any form — not in source code, not in test fixtures, not in log output
- **Input validation:** Always validate and sanitize all external input at the boundary layer before it reaches any business logic
- **No auth bypasses:** Never implement an auth bypass "to be fixed later" — incomplete auth is a blocker, not a deferrable item
- **Dependency audit:** Before adding any dependency, check for known vulnerabilities using the appropriate tool for this stack and document the check in `state/DECISIONS_LOG.md`
- **Security-first consultation:** Consult `SECURITY_STANDARDS.md` before implementing any feature involving input handling, authentication, external services, or data storage

---

## Environment Awareness Rules

- **Always identify the active environment** before running any command
- **Development:** Proceed with standard workflow
- **Staging or production:** Present a written plan and receive explicit confirmation before executing any change, migration, or destructive operation
- **No debug in prod:** Never expose debug ports, seed scripts, or development tooling in production configuration
- **Gitignore check:** Verify `.env` is properly gitignored before the first commit of any session
- **When in doubt:** Consult `ENVIRONMENT_GUIDE.md`

---

## Session End — Mandatory Before Closing

Every session must conclude with these updates:

- [ ] Update `state/CURRENT_STATUS.md` with accurate current state and a session summary
- [ ] Update `state/TASK_QUEUE.md` — mark completed tasks, add newly discovered tasks
- [ ] Log any significant decision in `state/DECISIONS_LOG.md`
- [ ] Update `CODING_STANDARDS.md` if new patterns or conventions were established
- [ ] Update `SECURITY_STANDARDS.md` if new security patterns were established or stack-specific guidance was extended
- [ ] Update `ENVIRONMENT_GUIDE.md` if environment configuration changed
- [ ] Update `README.md` if project-level context changed

---

## Self-Maintenance Directive

As the project evolves, the agent must proactively update all `.claude/` files:

- **Tech stack determined:** Update `CODING_STANDARDS.md` and `SECURITY_STANDARDS.md` immediately with stack-specific guidance
- **Architecture decided:** Update `README.md` and log in `DECISIONS_LOG.md`
- **Docker setup established:** Update `ENVIRONMENT_GUIDE.md` with real commands
- **New patterns observed:** Update `CODING_STANDARDS.md` to reflect actual codebase conventions

This is not optional — keeping `.claude/` accurate is part of every task.

---

## Escalation Rule

When blocked, uncertain about scope, or facing a decision with significant architectural, security, or UX impact:

1. Document the blocker in `state/CURRENT_STATUS.md`
2. Ask the user directly rather than assuming
3. Do not proceed until the blocker is resolved

**Assumption is the enemy of correctness. When in doubt, stop and ask.**
