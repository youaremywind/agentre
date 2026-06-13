import * as React from "react";
import { useTranslation } from "react-i18next";

import { ListAgentSkillPacks } from "@/../wailsjs/go/app/App";

import type { CatalogItem } from "../capability/catalog";
import { skillPacksToCatalog } from "./skill-catalog";

// useSkillCatalog 懒加载某 agent 的技能目录。组件应在「打开添加弹窗」时
// 调一次 load(false)，「重新扫描」时 load(true)。不在 mount 时自动拉，
// 避免每次打开面板都跑后端 CLI(plugin list)。
export function useSkillCatalog(agentId: number) {
  const { t } = useTranslation();
  const [items, setItems] = React.useState<CatalogItem[]>([]);
  const [loading, setLoading] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);
  const [fetched, setFetched] = React.useState(false);

  const load = React.useCallback(
    async (refresh: boolean) => {
      setLoading(true);
      setError(null);
      try {
        const resp = await ListAgentSkillPacks(agentId, refresh);
        setItems(skillPacksToCatalog(resp?.packs ?? [], t));
        setFetched(true);
      } catch (e) {
        setError(e instanceof Error ? e.message : String(e));
      } finally {
        setLoading(false);
      }
    },
    [agentId, t],
  );

  return { items, loading, error, fetched, load, reload: () => load(false) };
}
