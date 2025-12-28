"use client";

import * as React from "react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/status-badge";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { IconCopy, IconCheck, IconPlus, IconX, IconTrash, IconDots } from "@tabler/icons-react";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import type { Flag } from "@/types/flags";
import { toast } from "sonner";
import { updateFlagAction } from "@/app/actions/flags";

interface FlagDetailViewProps {
  flag: Flag;
  tenantId: string;
}

export function FlagDetailView({ flag, tenantId }: FlagDetailViewProps) {
  const [copied, setCopied] = useState(false);
  const [description, setDescription] = useState(flag.description || "");
  const [enabled, setEnabled] = useState(flag.enabled);
  const [ruleLogic, setRuleLogic] = useState(flag.rule_logic);
  const [isAddingRule, setIsAddingRule] = useState(false);
  const [newRule, setNewRule] = useState({
    attribute: "",
    operator: "equals" as const,
    value: "",
  });

  // Auto-save description changes with debouncing
  React.useEffect(() => {
    // Don't save if description hasn't changed
    if (description === flag.description) {
      return;
    }

    // Debounce: wait 500ms after user stops typing
    const timeoutId = setTimeout(async () => {
      try {
        const result = await updateFlagAction(tenantId, flag.id, {
          description: description || undefined,
        });

        if (result.success) {
          toast.success("Description updated");
        } else {
          toast.error(result.error || "Failed to update description");
          setDescription(flag.description || "");
        }
      } catch (error) {
        toast.error("Failed to update description");
        setDescription(flag.description || "");
      }
    }, 500);

    return () => clearTimeout(timeoutId);
  }, [description, flag.description, flag.id, tenantId]);

  const handleToggleEnabled = async (checked: boolean) => {
    setEnabled(checked);
    try {
      const result = await updateFlagAction(tenantId, flag.id, {
        enabled: checked,
      });

      if (result.success) {
        toast.success(`Flag ${checked ? "enabled" : "disabled"}`);
      } else {
        toast.error(result.error || "Failed to update flag");
        setEnabled(!checked);
      }
    } catch (error) {
      toast.error("Failed to update flag");
      setEnabled(!checked);
    }
  };

  const handleCopyId = async () => {
    await navigator.clipboard.writeText(flag.id);
    setCopied(true);
    toast.success("Flag ID copied to clipboard");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleAddRule = async () => {
    if (!newRule.attribute || !newRule.value) {
      toast.error("Please fill in all fields");
      return;
    }

    const updatedRules = [
      ...flag.rules,
      {
        id: crypto.randomUUID(),
        attribute: newRule.attribute,
        operator: newRule.operator,
        value: newRule.value,
      },
    ];

    try {
      const result = await updateFlagAction(tenantId, flag.id, {
        rules: updatedRules,
      });

      if (result.success) {
        toast.success("Rule added");
        flag.rules = updatedRules;
        setNewRule({ attribute: "", operator: "equals", value: "" });
        setIsAddingRule(false);
      } else {
        toast.error(result.error || "Failed to add rule");
      }
    } catch (error) {
      toast.error("Failed to add rule");
    }
  };

  const handleCancelAddRule = () => {
    setNewRule({ attribute: "", operator: "equals", value: "" });
    setIsAddingRule(false);
  };

  const handleRuleLogicChange = async (newLogic: "AND" | "OR") => {
    setRuleLogic(newLogic);
    try {
      const result = await updateFlagAction(tenantId, flag.id, {
        rule_logic: newLogic,
      });

      if (result.success) {
        toast.success(`Rule logic changed to ${newLogic}`);
        flag.rule_logic = newLogic;
      } else {
        toast.error(result.error || "Failed to update rule logic");
        setRuleLogic(flag.rule_logic);
      }
    } catch (error) {
      toast.error("Failed to update rule logic");
      setRuleLogic(flag.rule_logic);
    }
  };

  const handleDeleteRule = async (ruleId: string) => {
    const updatedRules = flag.rules.filter((r) => r.id !== ruleId);

    try {
      const result = await updateFlagAction(tenantId, flag.id, {
        rules: updatedRules,
      });

      if (result.success) {
        toast.success("Rule deleted");
        flag.rules = updatedRules;
      } else {
        toast.error(result.error || "Failed to delete rule");
      }
    } catch (error) {
      toast.error("Failed to delete rule");
    }
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleString("en-US", {
      year: "numeric",
      month: "long",
      day: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  };

  const operatorLabels = {
    equals: "Equals",
    not_equals: "Not Equals",
    contains: "Contains",
    greater_than: "Greater Than",
    less_than: "Less Than",
  };

  return (
    <div className="flex flex-col gap-12 animate-in fade-in slide-in-from-bottom-4 duration-200">
      {/* Header Section */}
      <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex-1 min-w-0">
          <h1 className="text-2xl sm:text-3xl font-bold">{flag.name}</h1>
        </div>

        {/* Toggle & Actions */}
        <div className="flex items-center gap-3 select-none">
          <label htmlFor="flag-enabled" className="text-sm font-medium">
            Enabled
          </label>
          <Switch
            id="flag-enabled"
            checked={enabled}
            onCheckedChange={handleToggleEnabled}
            className="data-[state=checked]:bg-status-enabled"
          />
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button variant="ghost" size="icon" className="h-8 w-8">
                <IconDots className="size-4" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end">
              <DropdownMenuLabel>Rule Logic</DropdownMenuLabel>
              <DropdownMenuItem onClick={() => handleRuleLogicChange("AND")}>
                {ruleLogic === "AND" && "✓ "}AND
              </DropdownMenuItem>
              <DropdownMenuItem onClick={() => handleRuleLogicChange("OR")}>
                {ruleLogic === "OR" && "✓ "}OR
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      {/* Metadata Section */}
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-4 gap-4 rounded-lg bg-muted p-4">
        <div>
          <div className="text-muted-foreground text-sm font-medium select-none">Status</div>
          <div className="mt-1 select-none">
            <StatusBadge enabled={flag.enabled} />
          </div>
        </div>
        <div>
          <div className="text-muted-foreground text-sm font-medium select-none">Rule Logic</div>
          <div className="mt-1 select-none">
            <Badge variant="outline" className="font-medium">{ruleLogic}</Badge>
          </div>
        </div>
        <div>
          <div className="text-muted-foreground text-sm font-medium select-none">Created</div>
          <div className="mt-1 font-semibold text-foreground select-none">
            {formatDate(flag.created_at)}
          </div>
        </div>
        <div>
          <div className="text-muted-foreground text-sm font-medium select-none">Last Updated</div>
          <div className="mt-1 font-semibold text-foreground select-none">
            {formatDate(flag.updated_at)}
          </div>
        </div>
      </div>

      {/* Description Section */}
      <div>
        <div className="text-muted-foreground text-sm font-medium select-none">Description</div>
        <textarea
          value={description}
          onChange={(e) => setDescription(e.target.value)}
          placeholder="Add description..."
          className="w-full text-sm sm:text-base text-foreground bg-transparent border-none outline-none resize-none placeholder:text-muted-foreground/50 cursor-text min-h-[24px] mt-1"
          rows={description ? Math.max(1, description.split('\n').length) : 1}
        />
      </div>

      {/* Rules Section */}
      <div className="space-y-4">
        <div className="select-none">
          <h2 className="text-xl font-semibold">Rules ({flag.rules.length})</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            {flag.rules.length === 0
              ? "No rules configured. This flag applies to all users."
              : `Rules are combined using ${flag.rule_logic} logic.`}
          </p>
        </div>
        {flag.rules.length > 0 ? (
          <div className="overflow-hidden rounded-lg border border-gradient-start/10">
            <Table>
              <TableHeader className="bg-muted/50 dark:bg-muted border-b border-border select-none">
                <TableRow className="hover:bg-transparent">
                  <TableHead className="font-semibold">Attribute</TableHead>
                  <TableHead className="font-semibold">Operator</TableHead>
                  <TableHead className="font-semibold">Value</TableHead>
                  <TableHead className="w-12"></TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {flag.rules.map((rule, index) => (
                  <TableRow
                    key={rule.id}
                    className="transition-all duration-200 hover:bg-gradient-subtle animate-in fade-in"
                    style={{
                      animationDelay: `${index * 30}ms`
                    }}
                  >
                    <TableCell className="font-medium py-4">
                      {rule.attribute}
                    </TableCell>
                    <TableCell className="py-4 select-none">
                      <Badge variant="outline" className="font-medium">
                        {operatorLabels[rule.operator]}
                      </Badge>
                    </TableCell>
                    <TableCell className="py-4">
                      <code className="bg-gradient-start/5 border border-gradient-start/10 rounded px-2 py-1 text-sm font-mono">
                        {String(rule.value)}
                      </code>
                    </TableCell>
                    <TableCell className="py-4 select-none">
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-8 w-8"
                        onClick={() => handleDeleteRule(rule.id)}
                      >
                        <IconTrash className="size-4" />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        ) : (
          <div className="text-muted-foreground flex h-32 items-center justify-center rounded-lg border-2 border-dashed border-border text-sm select-none">
            No rules configured
          </div>
        )}

        {/* Add Rule Form */}
        {isAddingRule ? (
          <div className="rounded-lg border border-gradient-start/10 p-4 space-y-4 bg-gradient-subtle">
            <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
              <div>
                <label className="text-sm font-medium mb-2 block select-none">Attribute</label>
                <Input
                  placeholder="e.g., userId, region"
                  value={newRule.attribute}
                  onChange={(e) => setNewRule({ ...newRule, attribute: e.target.value })}
                />
              </div>
              <div>
                <label className="text-sm font-medium mb-2 block select-none">Operator</label>
                <Select
                  value={newRule.operator}
                  onValueChange={(value: any) => setNewRule({ ...newRule, operator: value })}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="equals">Equals</SelectItem>
                    <SelectItem value="not_equals">Not Equals</SelectItem>
                    <SelectItem value="contains">Contains</SelectItem>
                    <SelectItem value="greater_than">Greater Than</SelectItem>
                    <SelectItem value="less_than">Less Than</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div>
                <label className="text-sm font-medium mb-2 block select-none">Value</label>
                <Input
                  placeholder="e.g., admin, US"
                  value={newRule.value}
                  onChange={(e) => setNewRule({ ...newRule, value: e.target.value })}
                />
              </div>
            </div>
            <div className="flex gap-2 select-none">
              <Button onClick={handleAddRule} size="sm">
                <IconCheck className="size-4 mr-1" />
                Add Rule
              </Button>
              <Button onClick={handleCancelAddRule} variant="outline" size="sm">
                <IconX className="size-4 mr-1" />
                Cancel
              </Button>
            </div>
          </div>
        ) : (
          <div className="select-none">
            <Button onClick={() => setIsAddingRule(true)} variant="outline" size="sm">
              <IconPlus className="size-4 mr-1" />
              Add Rule
            </Button>
          </div>
        )}
      </div>

      {/* Flag ID Section */}
      <div className="space-y-4">
        <div className="select-none">
          <h2 className="text-xl font-semibold">Flag ID</h2>
          <p className="mt-1 text-sm text-muted-foreground">
            Use this ID when integrating with the SDK
          </p>
        </div>
        <div className="flex items-center gap-2">
          <code className="bg-muted block rounded-lg p-4 text-sm flex-1 font-mono">
            {flag.id}
          </code>
          <div className="select-none">
            <Button
              variant="outline"
              size="icon"
              onClick={handleCopyId}
              className="shrink-0 transition-all hover:bg-gradient-start/10"
            >
              {copied ? (
                <IconCheck className="size-4 text-status-enabled" />
              ) : (
                <IconCopy className="size-4" />
              )}
            </Button>
          </div>
        </div>
      </div>
    </div>
  );
}
