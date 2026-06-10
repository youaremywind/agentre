import { DatabaseSync } from "node:sqlite";
import { tmpdir } from "node:os";
import { join } from "node:path";

// The e2e temp DB. AGENTRE_DATA_DIR is set by playwright.config.ts (in every process, incl.
// workers); fall back to the same deterministic path for safety.
const dbPath = () =>
  join(process.env.AGENTRE_DATA_DIR ?? join(tmpdir(), "agentre-e2e-data"), "agentre.db");

// Count chat_sessions stuck in agent_status='running'. Read-only, independent of the Go service
// layer — proves the real status write hit disk. After a finished turn this must be 0.
export function runningSessionCount(): number {
  const db = new DatabaseSync(dbPath(), { readOnly: true });
  try {
    db.exec("PRAGMA busy_timeout = 5000");
    const row = db
      .prepare("SELECT COUNT(*) AS n FROM chat_sessions WHERE agent_status = 'running'")
      .get() as { n: number };
    return row.n;
  } finally {
    db.close();
  }
}

// Count user-authored group_messages (sender_kind='user'). Read-only — proves a user post landed
// in the real group transcript at the source of truth, independent of the rendered UI.
export function groupUserMessageCount(): number {
  const db = new DatabaseSync(dbPath(), { readOnly: true });
  try {
    db.exec("PRAGMA busy_timeout = 5000");
    const row = db
      .prepare("SELECT COUNT(*) AS n FROM group_messages WHERE sender_kind = 'user'")
      .get() as { n: number };
    return row.n;
  } finally {
    db.close();
  }
}

// Count agent-authored group_messages whose text echoes the fake reply prefix (sender_kind='agent').
// Read-only — proves a member turn's reply actually bubbled into the group transcript via the real
// group_send MCP round-trip (fake → gateway /mcp/group/ → IngestAgentMessage), at the source of
// truth and independent of the rendered UI. The fake's reply text lands in the `content` column.
export function agentGroupMessageCount(): number {
  const db = new DatabaseSync(dbPath(), { readOnly: true });
  try {
    db.exec("PRAGMA busy_timeout = 5000");
    const row = db
      .prepare(
        "SELECT COUNT(*) AS n FROM group_messages WHERE sender_kind = 'agent' AND content LIKE '%e2e-fake-reply:%'",
      )
      .get() as { n: number };
    return row.n;
  } finally {
    db.close();
  }
}

// Count persisted assistant chat_messages whose text echoes the fake reply prefix. Read-only,
// independent of the UI — proves an agent turn's reply actually hit disk (used to corroborate
// rehydration after a reload). The fake's text lands in blocks_json, so match the raw column.
export function fakeAssistantMessageCount(): number {
  const db = new DatabaseSync(dbPath(), { readOnly: true });
  try {
    db.exec("PRAGMA busy_timeout = 5000");
    const row = db
      .prepare(
        "SELECT COUNT(*) AS n FROM chat_messages WHERE role = 'assistant' AND blocks_json LIKE '%e2e-fake-reply:%'",
      )
      .get() as { n: number };
    return row.n;
  } finally {
    db.close();
  }
}
