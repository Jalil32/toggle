"use client";

import * as React from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
  const handleToggleEnabled = () => {
    // Dummy implementation - just show toast
    const newStatus = !flag.enabled;
    toast.success(
      `Flag ${newStatus ? "enabled" : "disabled"} (dummy implementation)`
    );
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
    <div className="flex flex-col gap-6">
      <Card>
        <CardHeader>
          <div className="flex items-start justify-between">
            <div className="flex-1">
              <CardTitle className="text-2xl">{flag.name}</CardTitle>
              {flag.description && (
                <CardDescription className="mt-2 text-base">
                  {flag.description}
                </CardDescription>
              )}
            </div>
            <Badge
              variant={flag.enabled ? "default" : "secondary"}
              className="ml-4"
            >
              {flag.enabled ? "Enabled" : "Disabled"}
            </Badge>
          </div>
        </CardHeader>
        <CardContent className="space-y-6">
          <div className="flex flex-col gap-4 sm:flex-row sm:gap-6">
            <div className="flex-1">
              <div className="text-muted-foreground text-sm">Created</div>
              <div className="mt-1 font-medium">
                {formatDate(flag.created_at)}
              </div>
            </div>
            <div className="flex-1">
              <div className="text-muted-foreground text-sm">Last Updated</div>
              <div className="mt-1 font-medium">
                {formatDate(flag.updated_at)}
              </div>
            </div>
            <div className="flex-1">
              <div className="text-muted-foreground text-sm">Rule Logic</div>
              <div className="mt-1">
                <Badge variant="outline">{flag.rule_logic}</Badge>
              </div>
            </div>
          </div>

          <Separator />

          <div className="flex gap-2">
            <Button onClick={handleToggleEnabled} variant="default">
              {flag.enabled ? "Disable Flag" : "Enable Flag"}
            </Button>
            <Button onClick={handleEdit} variant="outline">
              Edit
            </Button>
            <Button onClick={handleDelete} variant="destructive">
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
            <div className="overflow-hidden rounded-lg border">
              <Table>
                <TableHeader className="bg-muted">
                  <TableRow>
                    <TableHead>Attribute</TableHead>
                    <TableHead>Operator</TableHead>
                    <TableHead>Value</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {flag.rules.map((rule, index) => (
                    <TableRow key={rule.id}>
                      <TableCell className="font-medium">
                        {rule.attribute}
                      </TableCell>
                      <TableCell>
                        <Badge variant="outline">
                          {operatorLabels[rule.operator]}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        <code className="bg-muted rounded px-2 py-1 text-sm">
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

      <Card>
        <CardHeader>
          <CardTitle>Flag ID</CardTitle>
          <CardDescription>
            Use this ID when integrating with the SDK
          </CardDescription>
        </CardHeader>
        <CardContent>
          <code className="bg-muted block rounded-lg p-4 text-sm">
            {flag.id}
          </code>
        </CardContent>
      </Card>
    </div>
  );
}
