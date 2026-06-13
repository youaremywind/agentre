import type { TFunction } from "i18next";

import type { CatalogBadge, CatalogItem } from "../capability/catalog";
import type { skill_svc } from "../../../../wailsjs/go/models";

// skillPacksToCatalog 把后端 SkillPackDTO[] 适配为 CapabilityPicker 的 CatalogItem[]。
// 分组按 recommended/installed flag（不是 raw source）—— 既推荐又安装的包
// 后端标 source=installed，但设计稿要它进「推荐」组。
export function skillPacksToCatalog(
  packs: skill_svc.SkillPackDTO[],
  t: TFunction,
): CatalogItem[] {
  return packs.map((p) => {
    const badges: CatalogBadge[] = [];
    if (p.recommended) {
      badges.push({
        label: t("org.agent.skillCatalog.badge.recommended"),
        tone: "recommended",
      });
    }
    if (p.installed) {
      badges.push({
        label: t("org.agent.skillCatalog.badge.installed"),
        tone: "installed",
      });
    } else {
      badges.push({
        label: t("org.agent.skillCatalog.badge.needInstall"),
        tone: "needInstall",
      });
    }

    const group = p.recommended
      ? t("org.agent.skillCatalog.group.recommended")
      : p.installed
        ? t("org.agent.skillCatalog.group.installed")
        : t("org.agent.skillCatalog.group.available");

    return {
      id: p.id,
      name: p.name,
      description: p.description,
      contents: p.skills ?? [],
      group,
      badges,
      enabled: p.enabled,
      disabledReason: p.installed
        ? undefined
        : t("org.agent.skillCatalog.needInstall"),
    };
  });
}
