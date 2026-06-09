import { beforeEach, describe, expect, it } from "vitest";

import { useGroupStore, type GroupDetail } from "../group-store";

function detailWith(members: unknown[]): GroupDetail {
  return {
    group: { id: 5, title: "队", runStatus: "running", roundCount: 0 },
    members,
    messages: [],
  } as unknown as GroupDetail;
}

describe("group-store patchMemberRunState", () => {
  beforeEach(() => {
    useGroupStore.setState({ details: new Map() });
  });

  it("updates only the target member's runState", () => {
    useGroupStore.getState().setDetail(
      5,
      detailWith([
        { id: 1, runState: "idle" },
        { id: 2, runState: "idle" },
      ]),
    );

    useGroupStore.getState().patchMemberRunState(5, 2, "running");

    const members = useGroupStore.getState().details.get(5)?.members;
    expect(members?.[0].runState).toBe("idle");
    expect(members?.[1].runState).toBe("running");
  });

  it("is a no-op when the member is absent", () => {
    useGroupStore
      .getState()
      .setDetail(5, detailWith([{ id: 1, runState: "running" }]));
    useGroupStore.getState().patchMemberRunState(5, 999, "idle");
    expect(useGroupStore.getState().details.get(5)?.members[0].runState).toBe(
      "running",
    );
  });
});
