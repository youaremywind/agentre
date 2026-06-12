import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { WorkflowEditDialog } from "./workflow-edit-dialog";

describe("WorkflowEditDialog", () => {
  it("新建模式:填名称正文 → 保存回调收到 trim 后的值", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    const onSubmit = vi.fn().mockResolvedValue(undefined);
    render(
      <WorkflowEditDialog
        open
        onOpenChange={() => {}}
        editing={null}
        onSubmit={onSubmit}
      />,
    );
    expect(screen.getByText("New workflow")).toBeTruthy();
    fireEvent.change(screen.getByRole("textbox", { name: "Name" }), {
      target: { value: "  产品开发流程 " },
    });
    fireEvent.change(
      screen.getByRole("textbox", { name: "Workflow content (Markdown)" }),
      { target: { value: "# 产品开发流程" } },
    );
    await user.click(screen.getByRole("button", { name: "Save" }));
    await waitFor(() =>
      expect(onSubmit).toHaveBeenCalledWith("产品开发流程", "# 产品开发流程"),
    );
  });

  it("名称为空时保存禁用", () => {
    render(
      <WorkflowEditDialog
        open
        onOpenChange={() => {}}
        editing={null}
        onSubmit={vi.fn()}
      />,
    );
    expect(
      (screen.getByRole("button", { name: "Save" }) as HTMLButtonElement)
        .disabled,
    ).toBe(true);
  });

  it("编辑模式:表单预填 + 标题为编辑", () => {
    render(
      <WorkflowEditDialog
        open
        onOpenChange={() => {}}
        editing={{
          id: 3,
          name: "旧名",
          content: "## 旧正文",
          groupCount: 1,
          createtime: 1,
          updatetime: 2,
        }}
        onSubmit={vi.fn()}
      />,
    );
    expect(screen.getByText("Edit workflow")).toBeTruthy();
    expect(
      (screen.getByRole("textbox", { name: "Name" }) as HTMLInputElement).value,
    ).toBe("旧名");
    expect(
      (
        screen.getByRole("textbox", {
          name: "Workflow content (Markdown)",
        }) as HTMLTextAreaElement
      ).value,
    ).toBe("## 旧正文");
  });

  it("插入骨架模板:空正文直接填入,非空追加到末尾", async () => {
    const user = userEvent.setup({ pointerEventsCheck: 0 });
    render(
      <WorkflowEditDialog
        open
        onOpenChange={() => {}}
        editing={null}
        onSubmit={vi.fn()}
      />,
    );
    const textarea = screen.getByRole("textbox", {
      name: "Workflow content (Markdown)",
    }) as HTMLTextAreaElement;
    await user.click(screen.getByRole("button", { name: "Insert template" }));
    // 骨架四段(spec §6.3):适用/角色/步骤/纪律
    expect(textarea.value).toContain("## Roles");
    expect(textarea.value).toContain("## Steps");
    expect(textarea.value).toContain("## Discipline");
    const first = textarea.value;
    await user.click(screen.getByRole("button", { name: "Insert template" }));
    expect(textarea.value.length).toBeGreaterThan(first.length);
    expect(textarea.value.startsWith(first)).toBe(true);
  });
});
