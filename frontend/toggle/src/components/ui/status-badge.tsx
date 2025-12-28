"use client";

import * as React from "react";
import { IconCircleCheckFilled, IconCircleXFilled } from "@tabler/icons-react";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";

interface StatusBadgeProps {
  enabled: boolean;
  className?: string;
  showIcon?: boolean;
}

export function StatusBadge({
  enabled,
  className,
  showIcon = true
}: StatusBadgeProps) {
  return (
    <Badge
      variant={enabled ? "enabled" : "disabled"}
      className={cn(
        "transition-all duration-200 hover:scale-105 active:scale-100",
        className
      )}
    >
      {showIcon && (
        enabled ? (
          <IconCircleCheckFilled className="size-3" />
        ) : (
          <IconCircleXFilled className="size-3" />
        )
      )}
      {enabled ? "Enabled" : "Disabled"}
    </Badge>
  );
}
