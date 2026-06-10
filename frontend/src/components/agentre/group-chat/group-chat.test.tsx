import { render, screen, fireEvent, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, it, expect, vi } from "vitest";

import { useChatTabsStore } from "@/stores/chat-tabs-store";

// mock useGroup to return a fixed detail (so the panel renders deterministically)
vi.mock("../../../hooks/use-group", () => ({
  useGroup: () => ({
    detail: {
      group: {
        id: 5,
        title: "队",
        runStatus: "running",
        roundCount: 3,
        projectId: 2,
      },
      members: [
        {
          id: 1,
          agentID: 2,
          role: "host",
          status: "active",
          backingSessionID: 11,
        },
        {
          id: 2,
          agentID: 3,
          role: "member",
          status: "active",
          backingSessionID: 12,
        },
        {
          id: 3,
          agentID: 4,
          role: "member",
          status: "active",
          backingSessionID: 0,
        },
      ],
      messages: [
        {
          id: 1,
          seq: 1,
          senderKind: "user",
          senderMemberID: 0,
          recipientMemberIDs: [1],
          toUser: false,
          content: "开工",
          createtime: 0,
        },
        {
          id: 2,
          seq: 2,
          senderKind: "agent",
          senderMemberID: 1,
          recipientMemberIDs: [],
          toUser: false,
          content: "请看 **重点**,交给 <mention>前端</mention>",
          createtime: 0,
        },
        {
          id: 3,
          seq: 3,
          senderKind: "system",
          senderMemberID: 0,
          recipientMemberIDs: [],
          toUser: false,
          content: "**前端** 加入了群聊",
          createtime: 0,
        },
      ],
    },
    loading: false,
    reload: vi.fn(),
  }),
}));
// mock the Wails bindings + ChatPanel embed (ChatPanel is heavy; stub it)
vi.mock("../../../../wailsjs/go/app/App", () => ({
  GroupSend: vi.fn(),
  GroupStop: vi.fn(),
  GroupPause: vi.fn(),
  GroupResume: vi.fn(),
  GroupAddMember: vi.fn(),
  GroupRemoveMember: vi.fn(),
  GroupRename: vi.fn(),
  GroupDelete: vi.fn(),
}));
vi.mock("../chat-panel", () => ({ ChatPanel: () => null }));
// mock useChatAgents so the panel resolves real agent names deterministically
// (agentID 2 → "后端", 3 → "前端", 4 → "产品") without hitting the ListChatAgents binding.
vi.mock("../../../hooks/use-chat-agents", () => ({
  useChatAgents: () => ({
    agents: [
      { id: 2, name: "后端" },
      { id: 3, name: "前端" },
      { id: 4, name: "产品" },
    ],
    loading: false,
    error: null,
    reload: vi.fn(),
  }),
}));
// mock useProjectList so the roster resolves projectId=2 → "Agentre-desktop"
// without hitting the ProjectListTree binding.
vi.mock("../../../hooks/use-project-list", () => ({
  useProjectList: () => ({
    projects: [{ id: 2, name: "Agentre-desktop" }],
    loading: false,
    error: null,
    reload: vi.fn(),
  }),
}));

import { GroupChat } from "./index";

function renderGroupChat(groupId = 5) {
  return render(
    <MemoryRouter>
      <GroupChat groupId={groupId} />
    </MemoryRouter>,
  );
}

describe("GroupChat", () => {
  beforeEach(() => {
    useChatTabsStore.setState({ tabs: [], activeTabId: null });
  });

  it("renders room title, run status pill and member roster", () => {
    renderGroupChat();
    expect(screen.getByText("队")).toBeInTheDocument(); // dynamic title
    expect(screen.getByText(/Running|运行中/)).toBeInTheDocument(); // run_status pill (en default in tests)
    expect(screen.getByText("Host")).toBeInTheDocument(); // members tab default, host section
  });

  it("renders the group identifier (group-{id}) in the header", () => {
    renderGroupChat();
    expect(screen.getByText("group-5")).toBeInTheDocument(); // mirrors sess-{id}
  });

  it("Given a group chat tab, When it renders, Then it does not create a nested chat tab strip", () => {
    renderGroupChat();
    expect(screen.queryByRole("tab", { name: /Group|群聊/ })).toBeNull();
  });

  it("Given a group member row, When it is opened, Then the member session opens in the top-level tab store", () => {
    renderGroupChat();
    fireEvent.click(screen.getByText("前端"));

    const state = useChatTabsStore.getState();
    const active = state.tabs.find((tab) => tab.id === state.activeTabId);
    expect(active?.meta).toEqual({
      kind: "groupSession",
      groupId: 5,
      sessionId: 12,
      title: "前端",
    });
  });

  it("Given a member has no backing session yet, When it is clicked, Then no empty session tab opens", () => {
    renderGroupChat();
    fireEvent.click(screen.getByText("产品"));

    const state = useChatTabsStore.getState();
    expect(state.tabs).toHaveLength(0);
  });

  it("agent 消息正文渲染 markdown(与单聊一致)", () => {
    const { container } = renderGroupChat();
    expect(container.querySelector("strong")?.textContent).toBe("重点");
  });

  it("markdown 正文里的 <mention> 渲染成 chip,点击跳转成员会话", () => {
    renderGroupChat();
    fireEvent.click(screen.getByRole("button", { name: "@前端" }));

    const state = useChatTabsStore.getState();
    const active = state.tabs.find((tab) => tab.id === state.activeTabId);
    expect(active?.meta).toEqual({
      kind: "groupSession",
      groupId: 5,
      sessionId: 12,
      title: "前端",
    });
  });

  it("system 行不走 markdown,内容原样展示", () => {
    renderGroupChat();
    expect(screen.getByText("**前端** 加入了群聊")).toBeInTheDocument();
  });

  it("switches right panel to settings tab showing the bound project", () => {
    renderGroupChat();
    fireEvent.click(screen.getByText(/Settings|设置/));
    // settings tab 现在展示群绑定项目(可点击跳转),取代了原先未接线的「工作目录」。
    expect(
      screen.getByRole("button", { name: "Agentre-desktop" }),
    ).toBeInTheDocument();
  });

  it("非贴底时显示「回到底部」按钮，点击拉回底部", () => {
    renderGroupChat();
    const scroller = screen.getByTestId("group-scroll");
    Object.defineProperty(scroller, "scrollHeight", {
      configurable: true,
      value: 1000,
    });
    Object.defineProperty(scroller, "clientHeight", {
      configurable: true,
      value: 500,
    });
    scroller.scrollTop = 0;

    // 初始贴底，无按钮
    expect(
      screen.queryByRole("button", { name: /Back to bottom|回到底部/ }),
    ).toBeNull();

    act(() => {
      scroller.dispatchEvent(new Event("scroll"));
    });

    const btn = screen.getByRole("button", { name: /Back to bottom|回到底部/ });
    fireEvent.click(btn);
    expect(scroller.scrollTop).toBe(1000);
  });
});
