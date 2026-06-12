import * as React from "react";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { useWorkflows, type WorkflowItem } from "@/hooks/use-workflows";
import { cn } from "@/lib/utils";

import { MarkdownText } from "../markdown-text";
import { WorkflowDeleteDialog } from "./workflow-delete-dialog";
import { WorkflowEditDialog } from "./workflow-edit-dialog";

// 摘要首行:跳过空行与 markdown 标题行,取第一行正文(列表行副标题用)。
function firstSummaryLine(content: string): string {
  for (const raw of content.split("\n")) {
    const line = raw.trim();
    if (!line || line.startsWith("#")) continue;
    return line;
  }
  return "";
}

export function WorkflowsPage() {
  const { t } = useTranslation();
  const { workflows, loading, error, create, update, remove } = useWorkflows();
  const [selectedID, setSelectedID] = React.useState(0);
  const [editorOpen, setEditorOpen] = React.useState(false);
  const [editing, setEditing] = React.useState<WorkflowItem | null>(null);
  // open 与 item 分开:关闭只翻 open,item 留到下次打开再换,
  // 否则关闭即置 null 会在 Radix 退出动画完成前 unmount 内容。
  const [deleting, setDeleting] = React.useState<WorkflowItem | null>(null);
  const [deleteOpen, setDeleteOpen] = React.useState(false);

  // 删除成功后 reload 使条目消失,selected 派生为 null 自动回空态;
  // 失败(remove 落 error)时保留选中,所以不预先清 selectedID。
  const confirmDelete = () => {
    if (deleting) void remove(deleting.id);
  };

  const selected = workflows.find((w) => w.id === selectedID) ?? null;

  const openCreate = () => {
    setEditing(null);
    setEditorOpen(true);
  };
  const openEdit = (w: WorkflowItem) => {
    setEditing(w);
    setEditorOpen(true);
  };
  const handleSubmit = async (name: string, content: string) => {
    if (editing) {
      await update(editing.id, name, content);
    } else {
      await create(name, content);
    }
  };

  return (
    <div className="flex h-full min-h-0">
      <aside className="flex w-[340px] shrink-0 flex-col border-r border-border">
        <header className="flex items-center justify-between gap-2 border-b border-border px-4 py-3">
          <div className="flex min-w-0 flex-col gap-0.5">
            <h1 className="text-sm font-semibold text-foreground">
              {t("workflows.title")}
            </h1>
            <p className="truncate text-2xs text-muted-foreground">
              {t("workflows.subtitle")}
            </p>
          </div>
          <Button type="button" size="sm" onClick={openCreate}>
            <Plus className="size-3.5" aria-hidden="true" />
            {t("workflows.new")}
          </Button>
        </header>
        <div className="flex-1 overflow-y-auto">
          {error ? (
            <div className="px-4 py-2 text-2xs text-destructive">{error}</div>
          ) : null}
          {workflows.length === 0 && !loading && !error ? (
            <div className="px-4 py-8 text-center text-xs text-muted-foreground">
              {t("workflows.empty")}
            </div>
          ) : null}
          {workflows.map((w) => {
            const summary = firstSummaryLine(w.content);
            return (
              <div
                key={w.id}
                className={cn(
                  "group relative border-b border-border",
                  selectedID === w.id && "bg-accent",
                )}
              >
                <button
                  type="button"
                  onClick={() => setSelectedID(w.id)}
                  aria-current={selectedID === w.id ? "true" : undefined}
                  className="flex w-full cursor-pointer flex-col gap-1 px-4 py-3 text-left hover:bg-accent/50"
                >
                  <div className="flex w-full items-center gap-2">
                    <span className="min-w-0 flex-1 truncate text-xs font-medium text-foreground">
                      {w.name}
                    </span>
                    {w.groupCount > 0 ? (
                      <span className="shrink-0 rounded-full bg-accent px-1.5 py-0.5 text-2xs text-muted-foreground">
                        {t("workflows.groupCount", { count: w.groupCount })}
                      </span>
                    ) : null}
                  </div>
                  {summary ? (
                    <p className="w-full truncate text-2xs text-muted-foreground">
                      {summary}
                    </p>
                  ) : null}
                  <div className="flex w-full items-center pr-14">
                    <span className="text-2xs text-muted-foreground">
                      {t("workflows.updatedAt", {
                        time: new Date(w.updatetime).toLocaleDateString(),
                      })}
                    </span>
                  </div>
                </button>
                <span className="absolute bottom-2.5 right-4 flex gap-1 opacity-0 transition-opacity group-focus-within:opacity-100 group-hover:opacity-100">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={t("workflows.edit")}
                    onClick={() => openEdit(w)}
                  >
                    <Pencil className="size-3" aria-hidden="true" />
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-xs"
                    aria-label={t("workflows.delete")}
                    onClick={() => {
                      setDeleting(w);
                      setDeleteOpen(true);
                    }}
                  >
                    <Trash2 className="size-3" aria-hidden="true" />
                  </Button>
                </span>
              </div>
            );
          })}
        </div>
      </aside>
      <section className="flex min-w-0 flex-1 flex-col">
        {selected ? (
          <>
            <header className="flex flex-col gap-0.5 border-b border-border px-5 py-3">
              <h2 className="text-sm font-semibold text-foreground">
                {selected.name}
              </h2>
              <p className="text-2xs text-muted-foreground">
                {t("workflows.preview.liveHint")}
              </p>
            </header>
            <div className="flex-1 overflow-y-auto px-5 py-4">
              <MarkdownText text={selected.content} />
            </div>
            <footer className="border-t border-border px-5 py-3">
              <Button
                type="button"
                size="sm"
                onClick={() => selected && openEdit(selected)}
              >
                {t("workflows.edit")}
              </Button>
            </footer>
          </>
        ) : (
          <div className="flex flex-1 items-center justify-center text-xs text-muted-foreground">
            {t("workflows.preview.empty")}
          </div>
        )}
      </section>
      <WorkflowEditDialog
        open={editorOpen}
        onOpenChange={setEditorOpen}
        editing={editing}
        onSubmit={handleSubmit}
      />
      <WorkflowDeleteDialog
        open={deleteOpen}
        onOpenChange={setDeleteOpen}
        workflow={deleting}
        onConfirm={confirmDelete}
      />
    </div>
  );
}
