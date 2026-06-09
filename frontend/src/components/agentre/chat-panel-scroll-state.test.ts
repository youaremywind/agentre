import { beforeEach, describe, expect, it } from "vitest";

import {
  __resetChatPanelScrollStateForTesting,
  loadTranscriptDraftState,
  pruneChatPanelScrollState,
  saveTranscriptDraftState,
} from "./chat-panel-scroll-state";

describe("chat panel tab UI state", () => {
  beforeEach(() => {
    __resetChatPanelScrollStateForTesting();
  });

  it("Given draft state for multiple tabs, When closed tabs are pruned, Then only active tab drafts remain", () => {
    saveTranscriptDraftState("tab-a", "userAsk:req-1", { text: "keep" });
    saveTranscriptDraftState("tab-b", "userAsk:req-1", { text: "drop" });

    pruneChatPanelScrollState(new Set(["tab-a"]));

    expect(
      loadTranscriptDraftState<{ text: string }>("tab-a", "userAsk:req-1"),
    ).toEqual({ text: "keep" });
    expect(
      loadTranscriptDraftState<{ text: string }>("tab-b", "userAsk:req-1"),
    ).toBeNull();
  });
});
