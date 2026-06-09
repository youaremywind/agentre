import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { GroupRoster } from "./group-roster";

import type { app } from "../../../../wailsjs/go/models";

// 两个都设成 member 角色,保证渲染顺序 = 数组顺序(host 会被分到上面一组)。
function members(): app.GroupMemberItem[] {
  return [
    {
      id: 1,
      agentID: 2,
      role: "member",
      status: "active",
      backingSessionID: 11,
      runState: "running",
    },
    {
      id: 2,
      agentID: 3,
      role: "member",
      status: "active",
      backingSessionID: 12,
      runState: "idle",
    },
  ] as unknown as app.GroupMemberItem[];
}

describe("GroupRoster status dot", () => {
  it("colors the dot by runState, not membership status", () => {
    const { container } = render(
      <GroupRoster
        members={members()}
        memberName={(id) => `M${id}`}
        onOpenMember={() => {}}
        onInvite={() => {}}
        onArchive={() => {}}
      />,
    );
    // 状态点是 size-1.5 的圆点(头像是更大的圆,不会命中)。
    const dots = container.querySelectorAll(".size-1\\.5");
    expect(dots).toHaveLength(2);
    // 在跑成员 → running 色;空转成员 → idle 色(即便 membership 都是 active)。
    expect(dots[0].classList.contains("bg-status-running")).toBe(true);
    expect(dots[1].classList.contains("bg-status-idle")).toBe(true);
  });
});
