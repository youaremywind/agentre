import * as React from "react";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import type { WorkflowItem } from "@/hooks/use-workflows";

import { AgentreDialog } from "../app-dialog";

export type WorkflowEditDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  /** null = 新建;非 null = 编辑该流程。 */
  editing: WorkflowItem | null;
  onSubmit: (name: string, content: string) => Promise<void>;
};

function WorkflowEditDialog({
  open,
  onOpenChange,
  editing,
  onSubmit,
}: WorkflowEditDialogProps) {
  const { t } = useTranslation();
  const [name, setName] = React.useState("");
  const [content, setContent] = React.useState("");
  const [submitting, setSubmitting] = React.useState(false);
  const [error, setError] = React.useState<string | null>(null);

  // 每次打开按 editing 重置表单,避免上次输入残留。
  React.useEffect(() => {
    if (open) {
      setName(editing?.name ?? "");
      setContent(editing?.content ?? "");
      setError(null);
    }
  }, [open, editing]);

  // 骨架模板(spec §6.3 四段:适用/角色/步骤/纪律);正文非空时追加到末尾不覆盖。
  const insertTemplate = () => {
    const tpl = t("workflows.editor.template");
    setContent((prev) => (prev.trim() ? `${prev.trimEnd()}\n\n${tpl}` : tpl));
  };

  const canSubmit = name.trim().length > 0 && !submitting;

  const submit = async () => {
    setError(null);
    setSubmitting(true);
    try {
      await onSubmit(name.trim(), content);
      onOpenChange(false);
    } catch (err) {
      setError(String(err));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <AgentreDialog
      open={open}
      onOpenChange={onOpenChange}
      title={
        editing
          ? t("workflows.editor.editTitle")
          : t("workflows.editor.createTitle")
      }
      contentClassName="max-w-[640px]"
      footer={
        <>
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
            disabled={submitting}
          >
            {t("common.cancel")}
          </Button>
          <Button
            type="button"
            disabled={!canSubmit}
            onClick={() => void submit()}
          >
            {submitting ? (
              <Loader2 className="size-3.5 animate-spin" aria-hidden="true" />
            ) : null}
            {t("workflows.editor.save")}
          </Button>
        </>
      }
    >
      <div className="flex flex-col gap-3.5">
        <label className="flex flex-col gap-1.5 text-xs">
          <span className="font-medium text-foreground">
            {t("workflows.editor.name")}
            <span className="ml-0.5 text-destructive">*</span>
          </span>
          <Input
            aria-label={t("workflows.editor.name")}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t("workflows.editor.namePlaceholder")}
            className="h-9 text-xs"
          />
        </label>
        <div className="flex flex-col gap-1.5 text-xs">
          <span className="flex items-center justify-between font-medium text-foreground">
            <span>{t("workflows.editor.content")}</span>
            <Button
              type="button"
              variant="link"
              size="sm"
              className="h-auto p-0 text-2xs"
              onClick={insertTemplate}
            >
              {t("workflows.editor.insertTemplate")}
            </Button>
          </span>
          <Textarea
            aria-label={t("workflows.editor.content")}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            rows={16}
            className="font-mono text-xs"
          />
        </div>
        {error ? (
          <div className="rounded-md border border-destructive bg-destructive-soft px-3 py-2 text-2xs text-destructive">
            {error}
          </div>
        ) : null}
      </div>
    </AgentreDialog>
  );
}

export { WorkflowEditDialog };
