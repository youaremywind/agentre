# Session Lifecycle

This doc owns the rules for creating and reusing `chat_sessions`. Read it before adding a feature that starts agent work from outside the normal chat composer, such as group chat, issues, hooks, or remote dispatch.

## Creation Boundary

`chat_svc` is the only service boundary that creates or reuses `chat_sessions`.

- Use `chat_svc.EnsureSession(ctx, req)` for domain-driven session creation.
- Keep the Wails binding thin: parse request -> call the owning service -> return.
- Other domains such as `group_svc`, `issue_svc`, and `hook_svc` must not call `chat_repo.Session().Create` directly.
- Repositories stay persistence-only; they do not decide whether a session should exist.

`EnsureGroupMemberSession` is a legacy compatibility wrapper. New domain-driven creation paths should prefer `EnsureSession`.

## Known Session Purposes

### Normal Chat

Normal chat creation still happens through `chat_svc.Send` with `SessionID=0`. The first user message creates the session, persists the user and assistant rows, and starts the runtime turn.

### Group Member Backing Session

Group members are first-class `group_members` rows. Their `backing_session_id` may be `0`, meaning the member is in the group but has never received a turn.

Group chat creates the backing session lazily:

- Creating a group does not create member sessions.
- Adding or inviting a member does not create a member session.
- The scheduler creates/reuses the backing session only when a user or agent message is actually delivered to that member.
- After creation, `group_members.backing_session_id` is updated and a `member_updated` group event is emitted so the frontend can enable member-session navigation.

Group backing sessions are ordinary `chat_sessions` rows with `group_id > 0`. They reuse chat history, runtime selection, steering, tool approval, permission mode, and remote execution behavior.

### Issue And Hook Dispatch

Issue and hook features that need to start agent work should call `chat_svc.EnsureSession` instead of writing `chat_sessions` themselves. Add a new `SessionPurpose` only when the identity and reuse key are different from an existing purpose.

For example, a future issue dispatch can define a purpose whose reuse key is `(issue_id, agent_id)` if redispatch should continue the same agent thread, or create a fresh normal chat if each dispatch must be isolated. That decision belongs in `chat_svc`, with the issue service only passing intent.

## Remote Execution

Remote execution does not move session creation to `agentred`.

The desktop app owns the local database and creates/reuses the `chat_sessions` row through `chat_svc`. When a turn starts, runtime selection decides whether execution is local or proxied through `remote.Runtime` to an `agentred` daemon. The remote daemon executes the turn and reports runtime state; it does not own the desktop session lifecycle.

This keeps session identity, sidebar state, read state, group membership, issue linkage, and notifications in one local source of truth.

## Adding A New Session Purpose

When adding a new feature that creates sessions:

1. Add a failing service test for the intended reuse key and error path.
2. Add the smallest `SessionPurpose` and request fields needed by `chat_svc.EnsureSession`.
3. Keep the feature service dependent on a narrow gateway/interface rather than on `chat_repo`.
4. Emit a domain event if the creating service stores the returned `SessionID` and the frontend needs to update live state.
5. Document the new purpose in this file.
