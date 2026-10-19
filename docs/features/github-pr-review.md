# Pull Request Review & Agent Collaboration

px0 can check out a pull request's full source tree and review it like any local workspace: full codebase navigation, symbol outline, LSP intelligence, and search, alongside a diff scoped to the PR's merge-base, real-time coding agent synchronization, inline comment drafting, AI batch application, and the same git panel a plain workspace has — so you can commit and push fixes back to the PR's own branch, and pull in new commits someone else pushed while you were reviewing.

---

## Overview & Core Purpose

GitHub's web PR view shows you the isolated diff, but not the codebase around it: jumping to a caller three files away, or checking how a changed function is used elsewhere, means either trusting memory or manually cloning and switching branches.

With px0, you simply pass the pull request URL:
```bash
px0 https://github.com/owner/repo/pull/123
```

px0 automatically prepares a temporary worktree or clone, computes the merge-base diff against the target branch, and opens a lightweight, zero-latency code viewer in your browser.

---

## Opening a Pull Request

Pull request review is triggered by passing the full URL directly:

```bash
px0 https://github.com/owner/repo/pull/123
```

> [!NOTE]
> Bare PR numbers (e.g. `px0 123`) and the `px0 pr` subcommand have been deprecated in favor of explicit URL routing (`px0 <url>`). Running `px0 pr` provides a helpful reminder to pass the URL directly.

### Interactive Preparation Spinner
Because fetching metadata and checking out remote references takes a few moments, px0 displays an animated CLI spinner:
```text
⠋ Fetching PR #123 metadata from github...
⠙ Fetching PR #123 head and preparing worktree...
⠸ Computing merge base with main...
✔ PR #123 checked out (Refactor auth token resolution)
```

### Merged PR Handling
If the pull request is already merged:
- px0 detects its merged status from the API and opens it directly without blocking.
- In both the CLI checkout message and the browser review header, a prominent purple **`Merged`** pill badge is displayed.

### Multi-Session Isolation
Each PR review runs as its own isolated process on its own port. Running `px0 https://github.com/owner/repo/pull/456` while another PR review or local workspace is open will not disturb existing sessions.

From inside a running px0 browser session, the Command Palette (`Cmd/Ctrl+K` → **Git: Open Pull Request…**) launches a new review process in a fresh browser tab. In a workspace opened from the [workbench](workbench.md), the PR opens as another workbench workspace instead.

---

## Reviewing & Real-Time Agent Collaboration

### 1. Merge-Base Diff View
- The **Changes** toggle in the sidebar defaults to all files modified by the PR.
- Press **`Cmd/Ctrl+D`** on any file to open side-by-side or unified diffs.
- The diff is computed against the merge-base between the PR head and its target branch, exactly mirroring the diff shown on GitHub.

### 2. Live Agent Synchronization
When you or your background AI coding agents (Claude Code, Gemini CLI, Cursor Agent, Antigravity, Aider, etc.) make edits to the PR checkout from other terminals:
- px0's real-time file watcher immediately picks up modifications without full re-indexing.
- Gutter diffs, status badges, and open tabs reload live in the browser.

### 3. Line Comments & Hover Actions
When hovering over code lines or diff lines:
- A thread icon appears next to the line number.
- Clicking it opens a context menu for that line. On a diff line GitHub knows about it includes:
  - **Add Review Comment**: Drafts an inline review comment.
  - **Edit Inline**: Prompts your local AI coding agent to edit those lines directly.
  - **Start Thread**: Opens a conversation with the agent about the line.
- Alternatively, select any range of lines and press **`Alt+R`** (or click **Comment** on the selection bar) to open the review comment composer.

### 4. Batch Applying Comments Locally (`⚡ Batch Apply`)
Drafted review comments appear in the PR top bar and inline across files:
- **⚡ Batch Apply**: Lets you apply all drafted review comments across the entire pull request in one go using your configured coding agent harness. The agent reads your comments as instructions and modifies the code directly in the PR worktree.
- Comments remain drafts until either batch-applied or formally submitted. Drafts are saved outside the repository, per pull request, so reopening the same PR, with `px0 <pr-url>` or from the [workbench](workbench.md), brings them back.

### 5. Submitting Formal Reviews
The PR review bar above the editor tabs hosts the overall review summary and verdict actions:
- **Comment**: Submit feedback without an approval status (available to all reviewers).
- **Approve** / **Request Changes**: Available when your authenticated token has repository push access.
- Submitting posts a single review payload containing all draft line comments and the review body.
- The verdict follows the branch. While a rebase is running or stopped halfway in the checkout, every verdict button is disabled. Once the checkout is rebased but not yet force-pushed, **Approve** stays disabled: the forge still holds the old commits, and approving now would approve those. When the forge reports conflicts, **Approve** still works but shows a dashed border and a tooltip saying the PR can't merge until the conflicts are resolved.

### 6. Merge Status, Resolving Conflicts, and Merging
The first row of the PR bar shows a **status pill** with the forge's answer to "can this merge?": *Ready to merge*, *Checks failing*, *Conflicts with main*, *Behind main*, *Merge blocked*, *Draft*, or *Checking mergeability...* (px0 asks again a few times while the forge computes it). The checkout's own state takes priority over the forge's: *Rebasing...*, *Rebase unfinished*, or *Rebased, not pushed*. Clicking the pill takes the next step (resolve or force-push). When there is no next step, the click refreshes the status.

**Resolving conflicts / updating the branch.** When the PR conflicts with its base, or is behind it, click **Resolve conflicts** (or **Update branch**). This is the git panel's [agent rebase flow](git-integration.md#catching-up-with-a-moving-remote), aimed at the PR's base:
1. px0 fetches the base branch itself, with your forge token, so the harness never needs credentials and never fetches.
2. The git panel's notice card asks for permission: **Rebase with AI** (replays the PR's commits on top of the base) or **Merge with AI** (brings the base in with a merge commit). Pick the harness and model right there. Nothing runs until you click.
3. The harness resolves conflicts and finishes the operation. It never pushes. **Cancel** stops it, and a rebase stopped halfway can be finished by the harness or aborted.
4. After a merge, **Push to PR** is an ordinary push. After a rebase, the PR's commits have been rewritten, so the only way to update the PR is **Force-push**, and px0 asks for confirmation first. The push uses `--force-with-lease` against the head the checkout last saw. If anyone else pushed to the branch in the meantime, the push is refused and their commits survive.

**Merging.** With push access, a split button sits next to **Approve**. The main half runs the method you used last. The caret lists every method the repository allows, with a one-line description of each: **Create a merge commit**, **Squash and merge**, **Rebase and merge**. While the PR can't merge, the main half becomes the next step instead (*Resolve conflicts*, *Update branch*, *Force-push first*, *Pull new commits*). Every merge opens a dialog first:
- A method switcher. Edits made under one method are kept when you switch to another.
- For a merge commit or a squash: an editable **commit title** and **message**, prefilled with the usual defaults (`Title (#N)` and a list of the PR's commits for a squash; `Merge pull request #N from branch` for a merge commit). **Rewrite with AI** has the harness rewrite them from the PR's commits and diff. It follows your [Commit Message Instructions](settings-and-configuration.md) from Settings plus any one-off instruction you type next to the button, and it rewrites the current text rather than starting from scratch. The result only fills the fields. Nothing merges until you confirm.
- For a rebase merge: the list of commits that will be replayed.
- Anything that makes merging now wrong blocks **Confirm**: a rebase in progress, a rebase not yet pushed, commits on the PR this checkout hasn't seen, or conflicts. Warnings that don't block are listed too: failing checks, and local commits that aren't pushed and won't be part of the merge.
- The merge is sent with the head SHA you reviewed. If the PR moved since then, the forge refuses the merge.

### 7. Editing, Committing, and Pushing Back
A PR checkout is a real git worktree, and the git panel works inside it exactly as it does in a plain workspace:
- **Edit and commit**: Make a change (by hand, or by dispatching a coding agent on the checkout), stage it, and either write a commit message yourself or click **Commit with AI**.
- **Push**: Sends the checkout's `HEAD` to the pull request's *actual* head branch on its actual repository — a fork included — not wherever the checkout happens to live locally.
- **Pull**: Re-fetches the PR's current head. If new commits landed on it since you opened the review and your checkout can fast-forward onto them cleanly, px0 updates the worktree and refreshes the diff, the PR bar, and the comments panel automatically. If your checkout has diverged — you made local commits that aren't on the PR head yet, or the head was force-pushed — px0 refuses with a clear message rather than merging; commit and push first, or resolve it in a terminal. (Bringing the PR up to date with its *base* branch is the separate **Resolve conflicts** flow above.)

> [!IMPORTANT]
> A PR checkout lives in a temporary directory for the life of the px0 process (see [Multi-Session Isolation](#multi-session-isolation) below). Any edits you make there — committed or not — are gone once the session closes, unless you've pushed them back to the PR's branch first.

---

## Authentication & Read-Only Review

px0 discovers forge credentials in the following order:

1. **`github.token`** in px0 Settings (`Cmd/Ctrl+,` → GitHub, or `~/.px0/settings.json`).
2. **`GITHUB_TOKEN`** environment variable.
3. **`GH_TOKEN`** environment variable.
4. **`gh auth token`** via the GitHub CLI if installed and authenticated.

### Unauthenticated & Read-Only Access
If no token is configured:
- Public repositories still check out and diff seamlessly.
- You can draft review comments locally and use **⚡ Batch Apply** with local AI agents.
- Formal review submission back to the remote forge requires an auth token. The CLI banner displays:
  ```text
  access: read-only (no github token: set GITHUB_TOKEN or gh auth login to submit reviews)
  ```

---

## Extensible Forge Architecture (`GitProvider`)

px0 abstracts forge interactions through a clean, minimal `GitProvider` interface in [`provider.go`](file:///home/arpit/workspace/px0/px0/provider.go):
- **Provider Detection**: Matches input URLs against registered providers (GitHub, and in the future GitLab, Bitbucket, etc.).
- **Normalized Metadata**: Maps forge-specific PR/MR objects to standard `PRMeta` structures.
- **Push Access & Token Discovery**: Isolates provider-specific authentication mechanisms.
- **Merging**: `MergeStatus` and `Merge` speak forge-neutral merge states (`clean`, `conflicts`, `behind`, `blocked`, ...) and methods (`merge`, `squash`, `rebase`). Each provider maps them onto its own API, so the pill, the split button, and the dialog work unchanged for any forge.

---

## Keyboard Shortcuts & Controls

| Shortcut / Control | Context | Action |
| :--- | :--- | :--- |
| `Alt+R` | Selection in editor or diff view | Open review comment composer |
| Line Hover (`✏`) | Hovering on editor line number | Choose between GitHub review comment or inline agent edit |
| `Cmd/Ctrl+D` | Active tab | Toggle side-by-side / unified diff against merge-base |
| Command Palette (`Cmd/Ctrl+K`) | Command Palette → **Git: Open Pull Request…** | Launch a new PR review tab |
| **`⚡ Batch Apply`** | PR header bar | Dispatch all drafted comments to local AI coding harness |
| **Submit Review** | PR header bar | Submit Approve / Request Changes / Comment to remote forge |
| Status pill | PR header bar | Show whether the PR can merge; click for the next step (resolve, force-push) or to refresh |
| **Squash and merge ▾** | PR header bar | Merge with the last-used method (caret: pick another); opens the merge dialog |
| `Cmd/Ctrl+Enter` / `Esc` | Merge dialog | Confirm the merge / cancel |
| **Resolve conflicts** | PR header bar | Rebase onto, or merge, the base branch with the coding harness (asks first) |
| **Force-push** | Sidebar git panel | After a rebase: replace the PR branch's commits, with a lease (asks first) |
| **Pull** | Sidebar git panel | Fast-forward the checkout onto the PR's current head; refuses on divergence |
| **Push** | Sidebar git panel | Push the checkout's `HEAD` to the PR's actual head branch |
| **Commit with AI** | Sidebar git panel | Write and commit a message for the staged diff with your coding harness |
