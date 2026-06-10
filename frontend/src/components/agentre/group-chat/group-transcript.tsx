import * as React from "react";
import { useTranslation } from "react-i18next";

import { MessageRow, MessageCopyButton } from "../message-row";
import type { AgentColor } from "../types";

import type { app } from "../../../../wailsjs/go/models";

type GroupMessageItem = app.GroupMessageItem;
type GroupMemberItem = app.GroupMemberItem;

// agentColorForMember 给每个成员稳定地分配一个色板色：用 member.id 取模色板长度，
// 让同一个成员在 transcript 里始终保持同一颜色（不依赖后端下发 color 字段）。
const PALETTE: AgentColor[] = [
  "agent-1",
  "agent-2",
  "agent-3",
  "agent-4",
  "agent-5",
  "agent-6",
  "agent-7",
  "agent-8",
  "agent-9",
  "agent-10",
];

function agentColorForMember(memberId: number): AgentColor {
  if (memberId <= 0) return "neutral";
  return PALETTE[memberId % PALETTE.length];
}

export type RenderBody = (content: string) => React.ReactNode;

// 默认 body 渲染：pre-wrap 纯文本块。renderBody 拥有整块正文(块级元素自带样式外壳),
// 父层可注入 markdown / mention chip 等富文本渲染,transcript 不再叠加自己的文本样式。
const defaultRenderBody: RenderBody = (content) => (
  <div className="whitespace-pre-wrap break-words text-sm leading-relaxed text-foreground">
    {content}
  </div>
);

// 默认 system 行渲染：原样内联文本(在居中胶囊 span 里,必须是 inline 内容)。
const defaultRenderSystemBody: RenderBody = (content) => content;

export type GroupTranscriptProps = {
  messages: GroupMessageItem[];
  roster: GroupMemberItem[];
  /** roster member id → 显示名（成员名是动态内容，由父层解析后传入，绝不进 t()）。 */
  memberName: (memberId: number) => string;
  /** user/agent 消息正文渲染接缝(块级,拥有整块正文),默认 pre-wrap 纯文本。 */
  renderBody?: RenderBody;
  /** system 行正文渲染接缝(内联,在居中胶囊里),默认原样文本。 */
  renderSystemBody?: RenderBody;
};

function GroupTranscript({
  messages,
  roster,
  memberName,
  renderBody = defaultRenderBody,
  renderSystemBody = defaultRenderSystemBody,
}: GroupTranscriptProps) {
  const { t } = useTranslation();
  const totalMembers = roster.filter((m) => m.status === "active").length;

  return (
    <div className="flex flex-col gap-4">
      {messages.map((msg) => {
        if (msg.senderKind === "system") {
          // system 行居中：content 是动态文案（"X 加入了群聊"），原样渲染不进 t()。
          return (
            <div
              key={msg.id}
              data-testid="group-message-system"
              className="flex justify-center"
            >
              <span className="rounded-full bg-secondary px-3 py-1 text-2xs text-muted-foreground">
                {renderSystemBody(msg.content)}
              </span>
            </div>
          );
        }

        const isUser = msg.senderKind === "user";
        // user 的「名字」是静态 chrome（"You"），走 t()；agent 名字是动态内容，原样取。
        const displayName = isUser
          ? t("group.you")
          : memberName(msg.senderMemberID);
        const color = isUser
          ? "neutral"
          : agentColorForMember(msg.senderMemberID);

        // 定向消息提示：recipientMemberIDs 非空且不是「发给所有成员」时，挂一条灰色小字。
        const directed =
          msg.recipientMemberIDs.length > 0 &&
          msg.recipientMemberIDs.length < totalMembers;
        const firstRecipientName =
          msg.recipientMemberIDs.length > 0
            ? memberName(msg.recipientMemberIDs[0])
            : "";

        return (
          <MessageRow
            key={msg.id}
            avatarName={displayName}
            avatarColor={color}
            name={displayName}
            headerExtra={
              directed ? (
                <span className="text-2xs text-muted-foreground">
                  {t("group.onlyXReceived", { name: firstRecipientName })}
                </span>
              ) : null
            }
            footer={<MessageCopyButton text={msg.content} />}
          >
            {renderBody(msg.content)}
          </MessageRow>
        );
      })}
    </div>
  );
}

export { GroupTranscript, agentColorForMember };
