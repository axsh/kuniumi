---
name: create-specification
description: Create a structured specification document from user ideas and requirements. Use when the user wants to create a spec, write a specification, document requirements, or start a new feature design.
---

# Create Specification Workflow

This skill creates a structured specification document (`.../ideas/.../XXX-{Name}.md`) from user-provided ideas and requirements.

## 1. Preparation: Check Status and Context

1. **Get status**:
   - Run `scripts/utils/show_current_status.sh`.
   - Extract `phase`, `branch`, `next_idea_id` from the JSON output.
   - Refer to these as `[Phase]`, `[Branch]`, `[NextID]` below.

## 2. Determine Output Location

1. **Directory**:
   - Base path: `prompts/phases/[Phase]/ideas/[Branch]/`
   - Example: `prompts/phases/001-webservices/ideas/main/`
   - Create the directory if it does not exist.
2. **File name**:
   - Format: `[NextID]-[Name].md`
   - `[Name]` should be a concise label that describes the spec (e.g. `Tokenizer`, `RateLimit-GlobalManagement`).

## 3. Specification Content Structure

The specification must include at least the following sections:

1. **Background**: Why this feature or change is needed. Current problems or challenges. May be omitted if unknown.
2. **Requirements**: Features to implement and conditions to satisfy. Concrete behaviors and constraints. Clearly distinguish mandatory vs optional requirements.
3. **Implementation Approach**: Technologies and architecture to use. Overview of major components/modules. Key design decisions.
4. **Verification Scenarios**:
   - **IMPORTANT (Preserve Details)**: If the user provides specific steps, conditions, or test scenarios (e.g. "(1) do X then (2) do Y"), transcribe them here at full granularity. Do NOT summarize or fold them into "Requirements".
   - This section shares the concrete image of "what constitutes done".
   - Recommended format: numbered chronological lists.
5. **Testing for the Requirements**:
   - Describe **automated** verification steps for each requirement.
   - **IMPORTANT (Mandatory Automated Verification)**: Manual-only plans ("visually confirm the screen") are NOT allowed. Always specify verification commands using project-standard scripts:
     - `scripts/process/build.sh`
     - `scripts/process/integration_test.sh`
   - Map each requirement to the script/test case that verifies it.

## 4. Create and Save

1. **User dialogue**: Listen carefully, ask clarifying questions. Organize information along the four axes: Background, Requirements, Implementation Approach, and Verification Scenarios.
   - **WARNING**: If the user provides concrete steps (Scenarios), do NOT silently convert them into abstract "functional requirements" and discard the steps. Always preserve them under "Verification Scenarios".
2. **Markdown formatting**: Use headings, lists, tables, code blocks, and optionally Mermaid diagrams.
3. **Save** the file to the determined directory.

## 5. Completion Check

1. **Review**: Confirm the spec covers Background, Requirements, and Implementation Approach.
2. **Present file path**: Show the user a link to the created file.
3. **Suggest next step**: Propose creating an implementation plan if appropriate (but do NOT proceed without explicit user instruction).
