import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, it, expect, vi } from "vitest";

import { OrgApprovalCard } from "./card";
import type { OrgApprovalData } from "@/stores/chat-streams-store";
import { AnswerOrgApproval } from "../../../../wailsjs/go/app/App";

vi.mock("../../../../wailsjs/go/app/App", () => ({
  AnswerOrgApproval: vi.fn().mockResolvedValue(undefined),
}));

describe("OrgApprovalCard", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  const pending = (
    overrides: Partial<OrgApprovalData> = {},
  ): OrgApprovalData => ({
    requestId: "org-1",
    toolName: "org_create_department",
    toolInput: { name: "研发部", parentId: 1 },
    status: "pending",
    ...overrides,
  });

  it("renders the tool label, the input payload and approve/reject buttons when pending", () => {
    render(<OrgApprovalCard approval={pending()} sessionId={42} />);
    // tools.org_create_department → "Create department" (setup forces en locale)
    expect(screen.getByText("Create department")).toBeDefined();
    // 入参 JSON 原样渲染(动态内容不翻译)
    expect(screen.getByText(/研发部/)).toBeDefined();
    expect(screen.getByText("Approve")).toBeDefined();
    expect(screen.getByText("Reject")).toBeDefined();
  });

  it("calls AnswerOrgApproval with allow:true when approve is clicked", async () => {
    const user = userEvent.setup();
    render(<OrgApprovalCard approval={pending()} sessionId={42} />);
    await user.click(screen.getByText("Approve"));
    await waitFor(() => {
      expect(AnswerOrgApproval).toHaveBeenCalledTimes(1);
    });
    expect(AnswerOrgApproval).toHaveBeenCalledWith(
      expect.objectContaining({
        sessionId: 42,
        requestId: "org-1",
        allow: true,
      }),
    );
  });

  it("calls AnswerOrgApproval with allow:false when reject is clicked", async () => {
    const user = userEvent.setup();
    render(<OrgApprovalCard approval={pending()} sessionId={42} />);
    await user.click(screen.getByText("Reject"));
    await waitFor(() => {
      expect(AnswerOrgApproval).toHaveBeenCalledTimes(1);
    });
    expect(AnswerOrgApproval).toHaveBeenCalledWith(
      expect.objectContaining({
        sessionId: 42,
        requestId: "org-1",
        allow: false,
      }),
    );
  });

  it("renders a read-only status badge with no buttons once denied", () => {
    render(
      <OrgApprovalCard
        approval={pending({
          status: "denied",
          result: "用户拒绝了删除操作",
        })}
        sessionId={42}
      />,
    );
    expect(screen.getByText("Rejected")).toBeDefined();
    expect(screen.getByText("用户拒绝了删除操作")).toBeDefined();
    expect(screen.queryByText("Approve")).toBeNull();
    expect(screen.queryByText("Reject")).toBeNull();
  });

  it("renders an approved badge with the result text", () => {
    render(
      <OrgApprovalCard
        approval={pending({
          status: "approved",
          result: "已创建部门 研发部",
        })}
        sessionId={42}
      />,
    );
    expect(screen.getByText("Approved")).toBeDefined();
    expect(screen.getByText("已创建部门 研发部")).toBeDefined();
    expect(screen.queryByText("Approve")).toBeNull();
  });

  it("renders an expired badge for status=expired", () => {
    render(
      <OrgApprovalCard
        approval={pending({ status: "expired" })}
        sessionId={42}
      />,
    );
    expect(screen.getByText("Expired")).toBeDefined();
    expect(screen.queryByText("Approve")).toBeNull();
  });
});
