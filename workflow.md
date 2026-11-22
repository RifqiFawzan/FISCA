## 10. Quick Reference

### New Project
Brainstorm → Create `plan.md` → Choose Tech Stack → Create Backlog → Break Down into Issues → Generate Project Kit → Start Build Loop

### Existing Project
Brainstorm → Create `plan-<feature>.md` → Review Technical Approach → Update Backlog → Create Issues → Update Project Kit → Start Build Loop

### Per Issue
1. **PM:** Groom issue, clarify requirements, define acceptance criteria, and identify dependencies.
2. **Engineer:** Analyze the codebase, implement the solution, and perform initial testing.
3. **QA:** Independently verify implementation against acceptance criteria.
4. **Decision:**
   - PASS → Approve and close the issue.
   - FAIL → Return to Engineer with reproducible findings.
   - Repeated or complex FAIL → Escalate for investigation or architectural review.
5. **Completion:** Update documentation, project status, and relevant technical notes.

### Investigation
Define Question → Search & Trace → Analyze Root Cause → Assess Impact → Produce Investigation Report → Make Decision → Feed Findings into Build Workflow (if needed)

### Small Task
Define Clear Scope → Write Acceptance Criteria → Implement → Run Relevant Tests → Verify Changes → Done

### General Rules
- Never implement without understanding the requirements and existing codebase.
- Every issue must have clear acceptance criteria.
- QA verification is required before closing implementation issues.
- Do not silently ignore failed tests or unresolved errors.
- Escalate complex or repeated failures instead of endlessly retrying.
- Update project documentation when implementation changes affect existing specifications.