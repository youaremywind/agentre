import { render, screen, fireEvent } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { GroupTranscript } from "./group-transcript";

import type { app } from "../../../../wailsjs/go/models";

const copySpy = vi.fn();
vi.mock("@/lib/clipboard-toast", () => ({
  copyTextWithToast: (text: string, opts: unknown) => {
    copySpy(text, opts);
    return Promise.resolve(true);
  },
}));

const roster = [
  { id: 1, agentID: 2, role: "host", status: "active", backingSessionID: 11 },
] as unknown as app.GroupMemberItem[];

function msg(over: Partial<app.GroupMessageItem>): app.GroupMessageItem {
  return {
    id: 1,
    seq: 1,
    senderKind: "agent",
    senderMemberID: 1,
    recipientMemberIDs: [],
    toUser: false,
    content: "hello from agent",
    createtime: 0,
    ...over,
  } as app.GroupMessageItem;
}

const memberName = (id: number) => (id === 1 ? "后端" : `#${id}`);

describe("GroupTranscript", () => {
  it("agent 消息渲染规范尺寸头像(size-7)", () => {
    render(
      <GroupTranscript
        messages={[msg({})]}
        roster={roster}
        memberName={memberName}
      />,
    );
    const avatar = screen.getByLabelText("后端");
    expect(avatar.className).toContain("size-7");
  });

  it("agent 消息有复制按钮，点击以正文调用 copyTextWithToast", () => {
    copySpy.mockClear();
    render(
      <GroupTranscript
        messages={[msg({ content: "hello from agent" })]}
        roster={roster}
        memberName={memberName}
      />,
    );
    fireEvent.click(screen.getByRole("button", { name: /Copy|复制/ }));
    expect(copySpy).toHaveBeenCalledWith(
      "hello from agent",
      expect.any(Object),
    );
  });

  it("默认正文渲染保持 pre-wrap 纯文本", () => {
    const { container } = render(
      <GroupTranscript
        messages={[msg({ content: "line1\nline2" })]}
        roster={roster}
        memberName={memberName}
      />,
    );
    expect(
      container.querySelector(".whitespace-pre-wrap")?.textContent,
    ).toContain("line1\nline2");
  });

  it("renderBody 拥有整块正文：transcript 不再包 pre-wrap 外壳", () => {
    const { container } = render(
      <GroupTranscript
        messages={[msg({ content: "**bold**" })]}
        roster={roster}
        memberName={memberName}
        renderBody={(content) => <div data-testid="custom-body">{content}</div>}
      />,
    );
    expect(screen.getByTestId("custom-body")).toHaveTextContent("**bold**");
    expect(container.querySelector(".whitespace-pre-wrap")).toBeNull();
  });

  it("system 行走 renderSystemBody,不进 renderBody", () => {
    const renderBody = vi.fn((content: string) => <div>{content}</div>);
    render(
      <GroupTranscript
        messages={[msg({ senderKind: "system", content: "X 加入了群聊" })]}
        roster={roster}
        memberName={memberName}
        renderBody={renderBody}
        renderSystemBody={(content) => (
          <span data-testid="system-body">{content}</span>
        )}
      />,
    );
    expect(screen.getByTestId("system-body")).toHaveTextContent("X 加入了群聊");
    expect(renderBody).not.toHaveBeenCalled();
  });

  it("system 行不渲染复制按钮", () => {
    render(
      <GroupTranscript
        messages={[msg({ senderKind: "system", content: "X 加入了群聊" })]}
        roster={roster}
        memberName={memberName}
      />,
    );
    expect(screen.queryByRole("button", { name: /Copy|复制/ })).toBeNull();
  });
});
