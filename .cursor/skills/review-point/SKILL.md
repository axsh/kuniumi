---
name: review-point
description: Pause the workflow for user review and confirmation before proceeding to the next phase. Use when a specification or implementation plan is complete and needs user approval, or when the user asks for a review checkpoint.
---

# Review Point (Workflow Pause)

This skill inserts a checkpoint between phases (specification → implementation plan → implementation execution) to prevent unintended automatic progression and ensure the user has time for review.

## Procedure

1. **Confirm current state**:
   - Review the artifacts generated or updated by the previous workflow (specifications, implementation plans, code, etc.).
   - Address any questions or change requests from the user.

2. **Maintain waiting state**:
   - **Do NOT automatically start the next workflow** until the user gives an explicit instruction such as "proceed to the next phase" or "run the next workflow".
   - If discussion or revisions are needed, remain in this review state and continue the dialogue.

3. **Guide next steps**:
   - If the user approves the current artifacts, suggest the next workflow:
     - After a specification is complete: "If this looks good, I can create an implementation plan next."
     - After an implementation plan is complete: "If this looks good, I can start the implementation next."

## Prohibited Actions

Do NOT start any of the following workflows without the user's explicit permission:

- Creating a specification
- Creating an implementation plan
- Executing an implementation plan
- Running the build pipeline
