import * as React from "react";
import { ChevronRight, Trash2, UserPlus } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { AgentAvatar } from "../primitives";

import { GroupDeleteDialog } from "./group-delete-dialog";
import { agentColorForMember } from "./group-transcript";

import type { app } from "../../../../wailsjs/go/models";

type GroupMemberItem = app.GroupMemberItem;

type RosterTab = "members" | "settings";

export type GroupRosterProps = {
  members: GroupMemberItem[];
  memberName: (memberId: number) => string;
  /** 群绑定的项目 id（0 = 未绑定）。取代了原先未接线的「工作目录」展示。 */
  projectID?: number;
  /** 项目显示名是动态值，原样展示，不进 t()。projectID>0 但缺名时按未绑定兜底。 */
  projectName?: string;
  /** 点击项目名跳转到该项目（navigate(`/projects?focus=<id>`)）。 */
  onOpenProject?: (projectID: number) => void;
  onOpenMember: (member: GroupMemberItem) => void;
  onInvite: () => void;
  onDelete: (deleteSessions: boolean) => void;
};

// 状态点按运行态(running/idle)着色,而不是成员身份(active/left)。在跑→绿,
// 否则→灰(空串 / idle / 已离群都算不在跑)。这样 roster 才与该成员的实际状态一致。
const runStateDotClass: Record<string, string> = {
  running: "bg-status-running",
  idle: "bg-status-idle",
};

function MemberRow({
  member,
  name,
  onOpen,
}: {
  member: GroupMemberItem;
  name: string;
  onOpen: (member: GroupMemberItem) => void;
}) {
  const { t } = useTranslation();
  const canOpen = member.backingSessionID > 0;
  return (
    <button
      type="button"
      onClick={() => onOpen(member)}
      disabled={!canOpen}
      className={cn(
        "flex w-full items-center gap-2.5 rounded-md px-2 py-1.5 text-left hover:bg-accent",
        !canOpen && "cursor-default opacity-70 hover:bg-transparent",
      )}
    >
      <AgentAvatar
        name={name}
        color={agentColorForMember(member.id)}
        size="sm"
      />
      {/* 成员名是动态内容，原样渲染。 */}
      <span className="min-w-0 flex-1 truncate text-sm text-foreground">
        {name}
      </span>
      <span
        aria-hidden="true"
        className={cn(
          "size-1.5 rounded-full",
          runStateDotClass[member.runState] ?? "bg-status-idle",
        )}
      />
      {canOpen ? (
        <ChevronRight
          className="size-4 text-muted-foreground"
          aria-label={t("group.roster.openMember")}
        />
      ) : null}
    </button>
  );
}

function GroupRoster({
  members,
  memberName,
  projectID,
  projectName,
  onOpenProject,
  onOpenMember,
  onInvite,
  onDelete,
}: GroupRosterProps) {
  const { t } = useTranslation();
  const [tab, setTab] = React.useState<RosterTab>("members");
  const [deleteOpen, setDeleteOpen] = React.useState(false);

  const hosts = members.filter((m) => m.role === "host");
  const regulars = members.filter((m) => m.role !== "host");

  return (
    <aside className="flex w-64 shrink-0 flex-col border-l border-border bg-card">
      <div className="flex shrink-0 gap-1 border-b border-border p-2">
        <Button
          type="button"
          variant={tab === "members" ? "secondary" : "ghost"}
          size="sm"
          className="flex-1"
          onClick={() => setTab("members")}
        >
          {t("group.tabs.members")}
        </Button>
        <Button
          type="button"
          variant={tab === "settings" ? "secondary" : "ghost"}
          size="sm"
          className="flex-1"
          onClick={() => setTab("settings")}
        >
          {t("group.tabs.settings")}
        </Button>
      </div>

      {tab === "members" ? (
        <div className="flex min-h-0 flex-1 flex-col overflow-auto p-2">
          {hosts.length > 0 ? (
            <>
              <div className="px-2 pb-1 pt-2 text-2xs font-semibold uppercase tracking-wide text-muted-foreground">
                {t("group.roster.host")}
              </div>
              {hosts.map((m) => (
                <MemberRow
                  key={m.id}
                  member={m}
                  name={memberName(m.id)}
                  onOpen={onOpenMember}
                />
              ))}
            </>
          ) : null}
          {regulars.length > 0 ? (
            <>
              <div className="px-2 pb-1 pt-3 text-2xs font-semibold uppercase tracking-wide text-muted-foreground">
                {t("group.roster.members")}
              </div>
              {regulars.map((m) => (
                <MemberRow
                  key={m.id}
                  member={m}
                  name={memberName(m.id)}
                  onOpen={onOpenMember}
                />
              ))}
            </>
          ) : null}
          <Button
            type="button"
            variant="outline"
            size="sm"
            className="mt-3"
            onClick={onInvite}
          >
            <UserPlus data-icon="inline-start" aria-hidden="true" />
            {t("group.roster.invite")}
          </Button>
        </div>
      ) : (
        <div className="flex min-h-0 flex-1 flex-col gap-4 overflow-auto p-4">
          <div>
            <div className="text-2xs font-semibold uppercase tracking-wide text-muted-foreground">
              {t("group.settings.project")}
            </div>
            {projectID && projectID > 0 && projectName ? (
              // 项目名是动态值，原样渲染；点击跳转到该项目。
              <button
                type="button"
                onClick={() => onOpenProject?.(projectID)}
                className="mt-1 block max-w-full truncate text-left text-xs text-primary hover:underline"
                title={projectName}
              >
                {projectName}
              </button>
            ) : (
              <div className="mt-1 text-xs text-muted-foreground">
                {t("group.settings.noProject")}
              </div>
            )}
          </div>
          <Button
            type="button"
            variant="destructive"
            size="sm"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 data-icon="inline-start" aria-hidden="true" />
            {t("group.delete.button")}
          </Button>
          <GroupDeleteDialog
            open={deleteOpen}
            onOpenChange={setDeleteOpen}
            onConfirm={onDelete}
          />
        </div>
      )}
    </aside>
  );
}

export { GroupRoster };
