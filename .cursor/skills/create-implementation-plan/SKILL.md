---
name: create-implementation-plan
description: Create a detailed implementation plan from a specification document. Use when the user wants to create an implementation plan, plan coding tasks, or convert a spec into actionable development steps.
---

# Create Implementation Plan Workflow

This skill creates an implementation plan (`.../plans/.../YYY-{Name}.md`) from a specification (`.../ideas/.../XXX-{Name}.md`), following project planning rules.

## 1. Input and Rule Verification

1. **Identify input file**: Use the file specified by the user or the currently open file as the "specification".
2. **Load rules**: Read `prompts/rules/planning-rules.md`.
3. **Get status**: Run `scripts/utils/show_current_status.sh`. Extract `phase`, `branch`, `next_plan_id` from the JSON output → `[Phase]`, `[Branch]`, `[NextID]`.

## 2. Determine Output Location

1. **Directory**: `prompts/phases/[Phase]/plans/[Branch]/` (create if absent).
2. **File name**: `[NextID]-[Name].md`

## 3. Fill the Template

> **Technical Inheritance Rule**:
> Concrete logic, formulas, constants, algorithms, code snippets, **and data structure definitions (Struct/Interface)** from the source specification must NEVER be summarized. Include them verbatim or in more detail.
> - All code blocks in the specification are "implementation targets". Never ignore them as "reference only".
> - Writing "implement as per the spec" is **FORBIDDEN**. Always restate the logic explicitly.

Use this template — fill every `[...]` placeholder:

```markdown
# [File name without extension]

> **Source Specification**: [relative path to the spec]

## Goal Description
[Brief overview of the feature or change]

## User Review Required
[Items needing user confirmation. Write "None." if none]

## Requirement Traceability

> **Traceability Check**:
> List every requirement/decision from the specification and map it to the corresponding section in this plan.
> If a requirement is deferred, state the reason explicitly.

| Requirement (from Spec) | Implementation Point (Section/File) |
| :--- | :--- |
| [Requirement text] | [e.g. "Proposed Changes > File A"] |

## Proposed Changes

[Per-file changes ordered by dependency: Interface → Struct → Logic]

### [Component name]

#### [MODIFY/NEW] [File path](file:///absolute-path)
- **Description**: [Summary of change]
- **Technical Design**:
  - [Function signatures, interface changes]
  - ```go
    // Pseudo-code or function signature
    func Example(arg Type) ReturnType { ... }
    ```
- **Logic**:
  - [Logic inherited from the spec, stated concretely]

## Step-by-Step Implementation Guide

[Chronological work steps, NOT per-file grouping]

1. **[Step Name]**:
   - Edit `[File Path]` to [specific action].
   - [Code-level instruction, e.g. "Add 'count' field to State struct"]
2. **[Step Name]**:
   - ...

## Verification Plan

### Automated Verification

1. **Build & Unit Tests**:
   ```bash
   ./scripts/process/build.sh
   ```
2. **Integration Tests**:
   ```bash
   ./scripts/process/integration_test.sh --categories [gui|llm|taskengine|general] --specify "[Test Case Name]"
   ```
   - **Log Verification**: [What to check in logs]

## Documentation

Update existing specs/docs under `prompts/specifications` affected by this plan.

#### [MODIFY] [File name](file:///absolute-path)
- **Update content**: [Changes]
```

## 4. Save

Save the file. If it needs splitting across multiple files, add a "Continuation Plan" section at the end.

## 5. Self-Review Checklist

Before finalizing, verify:

1. **Requirement coverage**: The `Requirement Traceability` table covers ALL spec requirements (including minor logic changes).
2. **Reproducibility**: The plan alone is specific enough for unambiguous implementation.
3. **Data structures**: Struct/model definitions from the spec are included without omission.
4. **Test coverage**:
   - (Go) Unit tests and integration tests are planned with proper categorization.
   - TDD approach is planned.
