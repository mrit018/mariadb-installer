---
name: remote-ssh-installer-porting
description: Adapt or recreate the mariadb-installer workflow in another Go project that manages remote Linux hosts over SSH. Use when an AI needs to port the architecture to a different app, including CLI flags, Windows GUI, dry-run/apply execution, single-host and multi-host config, root/sudo handling, remote file writes, and vendor-based build/validation rules.
---

# Remote SSH Installer Porting

Use this skill when you need to recreate the behavior of this project in another codebase.

## Goal

Build a local tool that:

- Runs on Windows, Linux, or macOS
- Connects to remote Linux hosts over SSH
- Performs all real work on the remote host only
- Supports `dry-run` and `apply`
- Supports single-host and multi-host config
- Supports root login or non-root login plus `sudo`
- Can expose the same workflow through CLI and GUI

## Core Rules

- Keep remote execution isolated from local execution.
- Keep CLI and GUI backed by the same execution path.
- Preserve `dry-run`/`apply` parity. Anything that can run in apply should be visible in dry-run.
- Never assume the local machine is the target host.
- Do not store SSH passwords or sudo passwords in saved profiles.
- Prefer vendor-based Go builds when the repo vendors dependencies.
- Do not commit build artifacts.

## Target Architecture

Map the implementation into these layers:

- `main.go`: parse flags and dispatch to the app layer
- `app.go`: own the reusable workflow and host resolution
- `internal/runner`: handle dry-run logging, prompts, and remote execution abstraction
- `internal/sshclient`: handle SSH auth, command execution, file writes, and root/sudo execution
- `internal/steps`: keep remote actions as atomic steps
- `internal/examples`: keep runnable example commands and troubleshooting snippets
- GUI files: keep local form handling, profile management, and process launch logic separate from the core workflow

## Porting Workflow

1. Identify the remote workflow.
   - What must be checked first?
   - What files must be written?
   - What packages or services must be installed?
   - What must happen in single-host mode versus cluster/multi-host mode?

2. Preserve the execution model.
   - Resolve targets from flags or config
   - Build a pipeline of named steps
   - Run steps sequentially per host
   - Bootstrap the first cluster node separately when needed

3. Keep SSH handling generic.
   - Support key auth and password auth
   - Allow non-root SSH login
   - If the remote user is not root, elevate commands through `sudo`
   - Allow `sudo` password input only when needed
   - Keep host key checking configurable

4. Keep file writes remote.
   - Write config files on the remote host
   - Avoid local-side shortcuts that bypass SSH

5. Keep configuration reusable.
   - Support single-host flags
   - Support multi-host JSON/YAML if the project needs it
   - Support profile save/load in GUI if that helps repeated use

6. Keep validation explicit.
   - Build with the repo’s expected Go flags
   - Run tests after changes
   - Verify the CLI and GUI use the same backend path

## What to Change When Porting

Replace the MariaDB-specific parts with the new project’s domain logic:

- Cleanup step
- Repository/package sources
- Config file contents
- Service names and bootstrap behavior
- OS family checks
- Example commands and troubleshooting text

Keep the surrounding control flow unless the new project clearly needs a different model.

## Implementation Checklist

- Keep a reusable `runApp(...)` entrypoint for both CLI and GUI
- Keep a target-resolution function for single-host and config-based runs
- Keep a step pipeline with named steps for error reporting
- Keep SSH auth, command execution, and file writing in one client layer
- Keep GUI handlers thin; they should collect input and call the same backend
- Keep saved profiles sanitized
- Keep `dry-run` usable without credentials when possible

## Good Porting Questions

Ask these before coding if the target project is unclear:

- What is the remote software or service being installed?
- Is the target OS family fixed or variable?
- Does non-root SSH login need to be supported?
- Are multi-host or cluster semantics required?
- Should GUI profiles persist, and if so, what fields must be excluded?
- What build constraints apply, such as vendoring or platform-specific packaging?

## Expected Output

When using this skill, produce:

- The code changes needed to recreate the remote installer pattern
- Updated CLI flags or config fields
- GUI entrypoints if requested
- Example commands for the new project
- Build and test confirmation

