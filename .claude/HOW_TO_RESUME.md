# How to Resume — Session Startup Protocol

Every agent must execute this protocol at the start of every session, without exception. Do not skip steps. Do not reorder steps.

---

## Step 1: Read `.claude/README.md`

Orient yourself — understand the project, stack, and directory structure.

## Step 2: Read `.claude/state/CURRENT_STATUS.md`

Know exactly what is done, in progress, and blocked. This is the ground truth of project state.

## Step 3: Read `.claude/state/TASK_QUEUE.md`

Identify the next task and confirm its dependencies are met. Do not start a task whose dependencies are incomplete.

## Step 4: Read `.claude/AGENT_RULES.md`

Re-internalize all behavioral rules before touching anything. These are non-negotiable.

## Step 5: Read `.claude/CODING_STANDARDS.md`

Re-internalize all conventions before writing any code. Follow established patterns, not personal preferences.

## Step 6: Read `.claude/SECURITY_STANDARDS.md`

Re-internalize all security requirements before writing any code. Security review is mandatory for every change.

## Step 7: Identify the Active Environment

Check `APP_ENV` or equivalent environment variable. Consult `ENVIRONMENT_GUIDE.md` if the environment is unclear or if behavior differs by environment.

## Step 8: Read Task-Relevant Docs

Read any PRD sections, architecture documents, API contracts, or design docs directly relevant to the current task. If no such docs exist yet, proceed to the next step.

## Step 9: Verify the Environment is Functional

Run the project's health-check or startup command.

*(Update this step with the real command once the project stack is established.)*

If the environment is not functional, stop and fix the environment before proceeding.

## Step 10: Confirm No Regressions

Run the existing test suite before writing any new code. All tests must pass before you begin.

*(Update this step with the real test command once the test framework is configured.)*

If tests fail, fix the failures before starting new work. Do not layer new code on top of a broken foundation.

## Step 11: Begin the Task

Implement → test → security review → report → update all `.claude/` state files.

Follow the completion checklists in `templates/` for the task type:
- New feature → `templates/new_feature.md`
- New endpoint → `templates/new_endpoint.md`
- New test → `templates/new_test.md`
- Bug fix → `templates/bug_fix.md`
