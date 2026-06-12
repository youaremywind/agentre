import { useCallback, useEffect, useState } from "react";

import {
  WorkflowCreate,
  WorkflowDelete,
  WorkflowList,
  WorkflowUpdate,
} from "../../wailsjs/go/app/App";

// 平铺投影:页面/弹窗只需要这些字段,避免直接耦合 wails models 类。
export type WorkflowItem = {
  id: number;
  name: string;
  content: string;
  groupCount: number;
  createtime: number;
  updatetime: number;
};

export function useWorkflows() {
  const [workflows, setWorkflows] = useState<WorkflowItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await WorkflowList();
      setWorkflows(
        (resp?.items ?? []).map((i) => ({
          id: i.id,
          name: i.name,
          content: i.content,
          groupCount: i.groupCount,
          createtime: i.createtime,
          updatetime: i.updatetime,
        })),
      );
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setLoading(false);
    }
  }, []);

  // load-on-mount pattern: reload() drives setLoading/setWorkflows/setError.
  // The React 19 lint rule discourages setState in effects, but there is no
  // Suspense-driven loader here; firing reload from an effect is the simplest
  // way to kick off the initial fetch.

  useEffect(() => {
    void reload();
  }, [reload]);

  const create = useCallback(
    async (name: string, content: string) => {
      await WorkflowCreate({ name, content });
      await reload();
    },
    [reload],
  );

  const update = useCallback(
    async (id: number, name: string, content: string) => {
      await WorkflowUpdate({ id, name, content });
      await reload();
    },
    [reload],
  );

  // remove 与 create/update 不同:create/update 被编辑弹窗表单 catch 后内联展示,
  // 所以保持抛出;remove 没有表单消费方,失败直接落 error 由页面列表区展示。
  const remove = useCallback(
    async (id: number) => {
      try {
        await WorkflowDelete({ id });
      } catch (e: unknown) {
        setError(e instanceof Error ? e.message : String(e));
        return;
      }
      await reload();
    },
    [reload],
  );

  return { workflows, loading, error, reload, create, update, remove };
}
