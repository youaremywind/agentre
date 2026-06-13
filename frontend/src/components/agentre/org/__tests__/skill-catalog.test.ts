import { describe, expect, it } from "vitest";
import i18n from "@/i18n";

import { skillPacksToCatalog } from "../skill-catalog";

const t = i18n.getFixedT("en");
const pack = (over: Record<string, unknown> = {}) => ({
  id: "x@m",
  name: "x",
  description: "d",
  skills: ["a", "b"],
  source: "installed",
  recommended: false,
  installed: true,
  enabled: false,
  ...over,
});

describe("skillPacksToCatalog", () => {
  it("maps a recommended+installed pack into the recommended group with both badges", () => {
    const [it0] = skillPacksToCatalog(
      [pack({ recommended: true, installed: true, enabled: true })],
      t,
    );
    expect(it0.group).toBe(t("org.agent.skillCatalog.group.recommended"));
    expect(it0.enabled).toBe(true);
    expect(it0.contents).toEqual(["a", "b"]);
    expect(it0.badges?.map((b) => b.tone).sort()).toEqual([
      "installed",
      "recommended",
    ]);
    expect(it0.disabledReason).toBeUndefined();
  });

  it("maps an installed-only pack into the installed group", () => {
    const [it0] = skillPacksToCatalog([pack()], t);
    expect(it0.group).toBe(t("org.agent.skillCatalog.group.installed"));
    expect(it0.badges?.map((b) => b.tone)).toEqual(["installed"]);
  });

  it("maps an available (not installed) pack as disabled with needInstall badge", () => {
    const [it0] = skillPacksToCatalog(
      [pack({ source: "available", installed: false })],
      t,
    );
    expect(it0.group).toBe(t("org.agent.skillCatalog.group.available"));
    expect(it0.disabledReason).toBe(t("org.agent.skillCatalog.needInstall"));
    expect(it0.badges?.map((b) => b.tone)).toEqual(["needInstall"]);
  });
});
