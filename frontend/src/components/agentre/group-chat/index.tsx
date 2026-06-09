import * as React from "react";
import { ArrowDown, Pause, Play, Square } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { useChatTabsStore } from "@/stores/chat-tabs-store";

import { useChatAgents } from "../../../hooks/use-chat-agents";
import { useGroup } from "../../../hooks/use-group";
import { useStickToBottom } from "../../../hooks/use-stick-to-bottom";

import { GroupComposer, type GroupComposerSend } from "./group-composer";
import { GroupRoster } from "./group-roster";
import { GroupTranscript } from "./group-transcript";
import { MentionText } from "./mention-text";

import {
  GroupArchive,
  GroupPause,
  GroupResume,
  GroupSend,
  GroupStop,
} from "../../../../wailsjs/go/app/App";
import { app } from "../../../../wailsjs/go/models";

type GroupMemberItem = app.GroupMemberItem;

// run_status -> i18n key suffix. The backend may send values like
// "waiting_user"; normalize them to existing locale keys.
// 这里收敛到已存在的 key；未知值兜底成 idle，绝不让 t() 落空。
const RUN_STATUS_KEY: Record<string, string> = {
  running: "running",
  waiting_user: "waitingUser",
  waitingUser: "waitingUser",
  paused: "paused",
  idle: "idle",
  error: "error",
};

function GroupChat({ groupId }: { groupId: number }) {
  const { t } = useTranslation();
  const { detail } = useGroup(groupId);
  const { agents } = useChatAgents();
  const openGroupMemberSession = useChatTabsStore(
    (s) => s.openGroupMemberSession,
  );

  const group = detail?.group;
  const members = React.useMemo(() => detail?.members ?? [], [detail?.members]);
  const messages = React.useMemo(
    () => detail?.messages ?? [],
    [detail?.messages],
  );

  const {
    ref: scrollRef,
    atBottom,
    scrollToBottom,
    onScroll,
  } = useStickToBottom(messages.length);

  const memberById = React.useMemo(() => {
    const map = new Map<number, GroupMemberItem>();
    for (const m of members) map.set(m.id, m);
    return map;
  }, [members]);

  // nameOf: agentID → 真实 agent 显示名,数据来自 chat-agents-store(useChatAgents)。
  // 找不到时回退 "Agent #<id>"(动态内容,绝不进 t())。
  const nameOf = React.useCallback(
    (agentID: number) =>
      agents.find((a) => a.id === agentID)?.name ?? `Agent #${agentID}`,
    [agents],
  );

  // memberName: roster member id → 显示名。先取成员的 agentID,再走 nameOf 拿真实名字。
  const memberName = React.useCallback(
    (memberId: number) => {
      const m = memberById.get(memberId);
      if (!m) return `#${memberId}`;
      return nameOf(m.agentID);
    },
    [memberById, nameOf],
  );

  const openMemberById = React.useCallback(
    (memberId: number) => {
      const member = memberById.get(memberId);
      if (!member || member.backingSessionID <= 0) return;
      openGroupMemberSession(
        groupId,
        member.backingSessionID,
        memberName(memberId),
      );
    },
    [groupId, memberById, memberName, openGroupMemberSession],
  );

  function openMember(member: GroupMemberItem) {
    openMemberById(member.id);
  }

  // mentionRoster: 把成员映射成 MentionText 需要的 { memberId, name } 形态,
  // 供 transcript 里 @mention chip 按名字命中 + 点击跳转成员会话视图。
  const mentionRoster = React.useMemo(
    () => members.map((m) => ({ memberId: m.id, name: nameOf(m.agentID) })),
    [members, nameOf],
  );

  // renderMessageBody: transcript 的 body 渲染接缝。把正文交给 MentionText 高亮
  // @mention,点击 chip 复用 openMemberById —— 与点击 roster 行的 "›" 跳到同一视图。
  const renderMessageBody = React.useCallback(
    (content: string) => (
      <MentionText
        text={content}
        roster={mentionRoster}
        onJump={openMemberById}
      />
    ),
    [mentionRoster, openMemberById],
  );

  async function handleSend(payload: GroupComposerSend) {
    if (!group) return;
    try {
      await GroupSend(
        app.GroupSendRequest.createFrom({
          groupID: group.id,
          text: payload.text,
          recipientMemberIDs: payload.recipientMemberIDs,
          toUser: false,
        }),
      );
    } catch (e: unknown) {
      console.error("[group] send failed", e);
    }
  }

  function runControl(fn: (id: number) => Promise<void>) {
    if (!group) return;
    void fn(group.id).catch((e: unknown) => {
      console.error("[group] control failed", e);
    });
  }

  if (!group) {
    return <main className="flex flex-1 bg-background" />;
  }

  const runStatusKey = RUN_STATUS_KEY[group.runStatus] ?? "idle";
  const isRunning = group.runStatus === "running";
  const isPaused = group.runStatus === "paused";

  return (
    <div className="flex min-h-0 min-w-0 flex-1">
      <main className="flex min-h-0 min-w-0 flex-1 flex-col bg-background">
        <div className="flex min-h-0 flex-1 flex-col">
          <div className="flex min-h-[44px] shrink-0 items-center gap-3 border-b border-border px-5 py-1.5">
            {/* 群标题是动态内容，不进 t()。 */}
            <div
              className="min-w-0 flex-1 truncate text-sm font-semibold"
              title={group.title}
            >
              {group.title}
            </div>
            <Badge variant="secondary">
              {t(`group.runStatus.${runStatusKey}`)}
            </Badge>
            <span className="text-2xs text-muted-foreground">
              {t("group.rounds", { count: group.roundCount })}
            </span>
            {isRunning ? (
              <>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => runControl(GroupPause)}
                >
                  <Pause data-icon="inline-start" aria-hidden="true" />
                  {t("group.controls.pause")}
                </Button>
                <Button
                  type="button"
                  variant="outline"
                  size="sm"
                  onClick={() => runControl(GroupStop)}
                >
                  <Square data-icon="inline-start" aria-hidden="true" />
                  {t("group.controls.stop")}
                </Button>
              </>
            ) : null}
            {isPaused ? (
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => runControl(GroupResume)}
              >
                <Play data-icon="inline-start" aria-hidden="true" />
                {t("group.controls.resume")}
              </Button>
            ) : null}
          </div>

          <section
            ref={scrollRef as React.Ref<HTMLElement>}
            onScroll={onScroll}
            data-testid="group-scroll"
            className="relative min-h-0 flex-1 overflow-auto px-7 py-5"
          >
            <GroupTranscript
              messages={messages}
              roster={members}
              memberName={memberName}
              renderBody={renderMessageBody}
            />
            {!atBottom ? (
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                aria-label={t("chatPanel.scroll.backToBottom")}
                title={t("chatPanel.scroll.backToBottom")}
                onClick={scrollToBottom}
                className="sticky bottom-4 z-20 ml-auto flex rounded-full bg-background shadow-md hover:shadow-lg dark:bg-background animate-in fade-in slide-in-from-bottom-1 duration-200 ease-out motion-reduce:animate-none"
              >
                <ArrowDown data-icon="only" aria-hidden="true" />
              </Button>
            ) : null}
          </section>

          <GroupComposer
            members={members}
            memberName={memberName}
            onSend={(payload) => void handleSend(payload)}
          />
        </div>
      </main>

      <GroupRoster
        members={members}
        memberName={memberName}
        onOpenMember={openMember}
        onInvite={() => {
          // MVP：邀请 picker 留待后续接入真实 agent 选择器。
          console.warn("[group] invite picker not wired");
        }}
        onArchive={() => void GroupArchive(group.id)}
      />
    </div>
  );
}

export { GroupChat };
