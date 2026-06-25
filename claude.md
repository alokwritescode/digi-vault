# digi-vault — Claude Code Instructions

## Project Overview
[Brief description of digi-vault]

## Tech Stack
[Golang? Node? List here]

## Daily Automation Task
When triggered, do the following in order:
1. Read all TODO comments in the codebase (`grep -r "TODO" .`)
2. Read failing tests if any (`go test ./...` or equivalent)
3. Propose a numbered plan of what you'll work on today — WAIT FOR APPROVAL
4. Implement the plan one task at a time
5. Run tests after each change
6. Update README/docs if any public API changed
7. Stage all changes and propose a git commit message — WAIT FOR APPROVAL before committing

## Code Conventions
- [Your naming conventions]
- [Branch strategy — commit to main? or feature branch?]
- [Test coverage expectations]

## Do NOT
- Push to remote without explicit approval
- Delete any files without confirmation
- Modify .env or secrets