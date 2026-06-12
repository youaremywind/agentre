import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import type { WorkflowItem } from "@/hooks/use-workflows";

import { AgentreDialog } from "../app-dialog";

export type WorkflowDeleteDialogProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  workflow: WorkflowItem | null;
  onConfirm: () => void;
};

// 删除确认:groupCount > 0 时强提示使用中群数(spec §8:删除前确认并提示使用中的群数)。
function WorkflowDeleteDialog({
  open,
  onOpenChange,
  workflow,
  onConfirm,
}: WorkflowDeleteDialogProps) {
  const { t } = useTranslation();
  if (!workflow) return null;
  return (
    <AgentreDialog
      open={open}
      onOpenChange={onOpenChange}
      title={t("workflows.deleteConfirm.title")}
      description={
        workflow.groupCount > 0
          ? t("workflows.deleteConfirm.desc", {
              name: workflow.name,
              count: workflow.groupCount,
            })
          : t("workflows.deleteConfirm.descUnused", { name: workflow.name })
      }
      footer={
        <>
          <Button
            type="button"
            variant="ghost"
            onClick={() => onOpenChange(false)}
          >
            {t("common.cancel")}
          </Button>
          <Button
            type="button"
            variant="destructive"
            onClick={() => {
              onConfirm();
              onOpenChange(false);
            }}
          >
            {t("workflows.deleteConfirm.confirm")}
          </Button>
        </>
      }
    />
  );
}

export { WorkflowDeleteDialog };
