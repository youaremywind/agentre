type TranscriptScrollState = {
  atBottom: boolean;
  scrollTop: number;
  // 非贴底时额外保存"视口顶部那条消息"的锚点:anchorId=消息 id,
  // anchorOffset=该消息顶边在视口顶上方的 px。路由重挂载后虚拟器无测量(整列 estimate),
  // 仅凭 scrollTop 像素会落到错的消息;有锚点时改用 scrollToAnchor 钉到该消息并随测量收敛。
  anchorId?: number;
  anchorOffset?: number;
};

const transcriptScrollStates = new Map<string, TranscriptScrollState>();
const transcriptDraftStates = new Map<string, unknown>();

export function saveTranscriptScrollState(
  key: string | undefined,
  state: TranscriptScrollState,
): void {
  if (!key) return;
  transcriptScrollStates.set(key, state);
}

export function loadTranscriptScrollState(
  key: string | undefined,
): TranscriptScrollState | null {
  if (!key) return null;
  return transcriptScrollStates.get(key) ?? null;
}

export function pruneChatPanelScrollState(activeKeys: Set<string>): void {
  for (const key of transcriptScrollStates.keys()) {
    if (!activeKeys.has(key)) transcriptScrollStates.delete(key);
  }
  for (const key of transcriptDraftStates.keys()) {
    const tabKey = key.slice(0, key.indexOf(":"));
    if (!activeKeys.has(tabKey)) transcriptDraftStates.delete(key);
  }
}

export function saveTranscriptDraftState<T>(
  tabKey: string | undefined,
  draftKey: string | undefined,
  state: T,
): void {
  if (!tabKey || !draftKey) return;
  transcriptDraftStates.set(`${tabKey}:${draftKey}`, state);
}

export function loadTranscriptDraftState<T>(
  tabKey: string | undefined,
  draftKey: string | undefined,
): T | null {
  if (!tabKey || !draftKey) return null;
  return (transcriptDraftStates.get(`${tabKey}:${draftKey}`) as T) ?? null;
}

export function clearTranscriptDraftState(
  tabKey: string | undefined,
  draftKey: string | undefined,
): void {
  if (!tabKey || !draftKey) return;
  transcriptDraftStates.delete(`${tabKey}:${draftKey}`);
}

export function __resetChatPanelScrollStateForTesting(): void {
  transcriptScrollStates.clear();
  transcriptDraftStates.clear();
}
