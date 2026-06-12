import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

const workflowList = vi.fn();
const workflowCreate = vi.fn();
const workflowUpdate = vi.fn();
const workflowDelete = vi.fn();

vi.mock("../../../../wailsjs/go/app/App", () => ({
  WorkflowList: (...a: unknown[]) => workflowList(...a),
  WorkflowCreate: (...a: unknown[]) => workflowCreate(...a),
  WorkflowUpdate: (...a: unknown[]) => workflowUpdate(...a),
  WorkflowDelete: (...a: unknown[]) => workflowDelete(...a),
}));

import { WorkflowsPage } from "./workflows-page";

const items = [
  {
    id: 1,
    name: "产品开发流程",
    content: "# 产品开发流程\n\n适用:新功能完整交付。\n\n## 角色",
    groupCount: 2,
    createtime: 1700000000000,
    updatetime: 1700000000000,
  },
  {
    id: 2,
    name: "紧急修复流程",
    content: "",
    groupCount: 0,
    createtime: 1700000000000,
    updatetime: 1700000000000,
  },
];

describe("WorkflowsPage", () => {
  beforeEach(() => {
    workflowList.mockReset().mockResolvedValue({ items });
    workflowCreate.mockReset().mockResolvedValue({ item: { id: 9 } });
    workflowUpdate.mockReset().mockResolvedValue({ item: { id: 1 } });
    workflowDelete.mockReset().mockResolvedValue({});
  });

  it("列表行:名称 + 摘要首行 + 使用中群数", async () => {
    render(<WorkflowsPage />);
    await waitFor(() => expect(screen.getByText("产品开发流程")).toBeTruthy());
    // 摘要首行跳过 markdown 标题与空行
    expect(screen.getByText("适用:新功能完整交付。")).toBeTruthy();
    // 使用中群数 badge(en:Used by 2 groups);0 个群不显示 badge
    expect(screen.getByText("Used by 2 groups")).toBeTruthy();
    expect(screen.queryByText("Used by 0 groups")).toBeNull();
  });

  it("点列表行 → 右侧预览正文 + 「修改即时生效」标注", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(<WorkflowsPage />);
    await waitFor(() => expect(screen.getByText("产品开发流程")).toBeTruthy());
    // 未选中时显示预览空态
    expect(screen.getByText("Select a workflow to preview")).toBeTruthy();
    await user.click(screen.getByText("产品开发流程"));
    // 预览面板:markdown 正文渲染 + live hint + 底部编辑按钮
    await waitFor(() =>
      expect(
        screen.getByRole("heading", { level: 1, name: "产品开发流程" }),
      ).toBeTruthy(),
    );
    expect(
      screen.getByText(
        "Changes take effect immediately: running groups get the latest content next round",
      ),
    ).toBeTruthy();
    // 选中后预览底部多出一颗「编辑流程」按钮(行内 2 颗 + 底部 1 颗)
    expect(
      screen.getAllByRole("button", { name: "Edit workflow" }),
    ).toHaveLength(3);
  });

  it("空列表显示空态", async () => {
    workflowList.mockResolvedValue({ items: [] });
    render(<WorkflowsPage />);
    await waitFor(() =>
      expect(
        screen.getByText('No workflows yet — click "New workflow" to start'),
      ).toBeTruthy(),
    );
  });

  it("新建按钮开弹窗 → 保存调 WorkflowCreate 并 reload", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(<WorkflowsPage />);
    await waitFor(() => expect(screen.getByText("产品开发流程")).toBeTruthy());
    await user.click(screen.getByRole("button", { name: "New workflow" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Name" }), {
      target: { value: "评审流程" },
    });
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(workflowCreate).toHaveBeenCalledWith({
        name: "评审流程",
        content: "",
      }),
    );
    // 写后重载列表
    expect(workflowList.mock.calls.length).toBeGreaterThanOrEqual(2);
  });

  it("行内铅笔开编辑弹窗 → 保存调 WorkflowUpdate", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(<WorkflowsPage />);
    await waitFor(() => expect(screen.getByText("产品开发流程")).toBeTruthy());
    await user.click(
      screen.getAllByRole("button", { name: "Edit workflow" })[0],
    );
    const nameInput = screen.getByRole("textbox", {
      name: "Name",
    }) as HTMLInputElement;
    expect(nameInput.value).toBe("产品开发流程");
    fireEvent.change(nameInput, { target: { value: "产品开发流程 v2" } });
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(workflowUpdate).toHaveBeenCalledWith(
        expect.objectContaining({ id: 1, name: "产品开发流程 v2" }),
      ),
    );
  });

  it("删除:确认弹窗提示使用中群数 → 确认调 WorkflowDelete", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(<WorkflowsPage />);
    await waitFor(() => expect(screen.getByText("产品开发流程")).toBeTruthy());
    await user.click(screen.getAllByRole("button", { name: "Delete" })[0]);
    // 使用中群数警示(items[0].groupCount = 2)
    expect(
      screen.getByText(
        '"产品开发流程" is used by 2 groups; after deletion they fall back to "no workflow". This cannot be undone.',
      ),
    ).toBeTruthy();
    await user.click(screen.getByRole("button", { name: "Delete workflow" }));
    await waitFor(() => expect(workflowDelete).toHaveBeenCalledWith({ id: 1 }));
  });

  it("删除选中流程后预览回空态", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(<WorkflowsPage />);
    await waitFor(() => expect(screen.getByText("产品开发流程")).toBeTruthy());
    await user.click(screen.getByText("产品开发流程"));
    await waitFor(() =>
      expect(
        screen.getAllByRole("button", { name: "Edit workflow" }).length,
      ).toBeGreaterThanOrEqual(3),
    );
    // 删除后列表只剩 id=2
    workflowList.mockResolvedValue({ items: [items[1]] });
    await user.click(screen.getAllByRole("button", { name: "Delete" })[0]);
    await user.click(screen.getByRole("button", { name: "Delete workflow" }));
    await waitFor(() =>
      expect(screen.getByText("Select a workflow to preview")).toBeTruthy(),
    );
  });
});
