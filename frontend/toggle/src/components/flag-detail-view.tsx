"use client";

import * as React from "react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { StatusBadge } from "@/components/ui/status-badge";
import { Button } from "@/components/ui/button";
import { IconCopy, IconCheck } from "@tabler/icons-react";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Separator } from "@/components/ui/separator";
import type { Flag } from "@/types/flags";
import { toast } from "sonner";

interface FlagDetailViewProps {
  flag: Flag;
}

export function FlagDetailView({ flag }: FlagDetailViewProps) {
  const [copied, setCopied] = useState(false);

  const handleToggleEnabled = () => {
    // Dummy implementation - just show toast
    const newStatus = !flag.enabled;
    toast.success(
      `Flag ${newStatus ? "enabled" : "disabled"} (dummy implementation)`
    );
  };

  const handleCopyId = async () => {
    await navigator.clipboard.writeText(flag.id);
    setCopied(true);
    toast.success("Flag ID copied to clipboard");
    setTimeout(() => setCopied(false), 2000);
  };

  const handleEdit = () => {
    toast.info("Edit functionality coming soon (backend integration needed)");
  };

  const handleDelete = () => {
    toast.info("Delete functionality coming soon (backend integration needed)");
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
    <div className="flex flex-col gap-6 animate-in fade-in slide-in-from-bottom-4 duration-500">
      <Card className="border-gradient-start/10 shadow-lg hover:shadow-xl transition-all duration-300">
        <CardHeader>
          <div className="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div className="flex-1">
              <CardTitle className="text-xl sm:text-2xl">{flag.name}</CardTitle>
              {flag.description && (
                <CardDescription className="mt-2 text-sm sm:text-base">
                  {flag.description}
                </CardDescription>
              )}
            </div>
            <StatusBadge
              enabled={flag.enabled}
              className="self-start sm:ml-4"
            />
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:gap-6 rounded-lg bg-gradient-subtle p-4 border border-gradient-start/10">
            <div className="flex-1">
              <div className="text-muted-foreground text-sm font-medium">Created</div>
              <div className="mt-1 font-semibold text-foreground">
                {formatDate(flag.created_at)}
              </div>
            </div>
            <div className="flex-1">
              <div className="text-muted-foreground text-sm font-medium">Last Updated</div>
              <div className="mt-1 font-semibold text-foreground">
                {formatDate(flag.updated_at)}
              </div>
            </div>
            <div className="flex-1">
              <div className="text-muted-foreground text-sm font-medium">Rule Logic</div>
              <div className="mt-1">
                <Badge variant="outline" className="font-medium">{flag.rule_logic}</Badge>
              </div>
            </div>
          </div>

          <Separator />

          <div className="flex flex-col sm:flex-row gap-2">
            <Button onClick={handleToggleEnabled} variant="default" className="w-full sm:w-auto">
              {flag.enabled ? "Disable Flag" : "Enable Flag"}
            </Button>
            <Button onClick={handleEdit} variant="outline" className="w-full sm:w-auto">
              Edit
            </Button>
            <Button onClick={handleDelete} variant="destructive" className="w-full sm:w-auto">
              Delete
            </Button>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Rules ({flag.rules.length})</CardTitle>
          <CardDescription>
            {flag.rules.length === 0
              ? "No rules configured. This flag applies to all users."
              : `Rules are combined using ${flag.rule_logic} logic.`}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {flag.rules.length > 0 ? (
            <div className="overflow-hidden rounded-lg border border-gradient-start/10 shadow-sm">
              <Table>
                <TableHeader className="bg-gradient-subtle border-b border-gradient-start/10">
                  <TableRow className="hover:bg-transparent">
                    <TableHead className="font-semibold">Attribute</TableHead>
                    <TableHead className="font-semibold">Operator</TableHead>
                    <TableHead className="font-semibold">Value</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {flag.rules.map((rule, index) => (
                    <TableRow
                      key={rule.id}
                      className="transition-all duration-200 hover:bg-gradient-subtle animate-in fade-in"
                      style={{
                        animationDelay: `${index * 50}ms`
                      }}
                    >
                      <TableCell className="font-medium">
                        {rule.attribute}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline" className="font-medium">
                          {operatorLabels[rule.operator]}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <code className="bg-gradient-start/5 border border-gradient-start/10 rounded px-2 py-1 text-sm font-mono">
                          {String(rule.value)}
                        </code>
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          ) : (
            <div className="text-muted-foreground flex h-32 items-center justify-center rounded-lg border border-dashed text-sm">
              No rules configured
            </div>
          )}
        </CardContent>
      </Card>

      <Card className="relative overflow-hidden">
        <div className="absolute inset-0 bg-gradient-to-br from-gradient-start/5 to-gradient-end/5 pointer-events-none" />
        <CardHeader className="relative">
          <CardTitle>Flag ID</CardTitle>
          <CardDescription>
            Use this ID when integrating with the SDK
          </CardDescription>
        </CardHeader>
        <CardContent className="relative">
          <div className="flex items-center gap-2">
            <code className="bg-muted block rounded-lg p-4 text-sm flex-1 font-mono">
              {flag.id}
            </code>
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
        </CardContent>
      </Card>
    </div>
  );
}
