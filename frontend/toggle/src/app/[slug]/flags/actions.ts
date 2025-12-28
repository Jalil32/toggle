"use server";

import { revalidatePath } from "next/cache";
import { headers } from "next/headers";
import { createFlag } from "@/lib/api-server";
import type { Flag } from "@/types/flags";

export async function createFlagAction(
  slug: string,
  tenantId: string,
  data: {
    name: string;
    description: string;
    enabled: boolean;
  }
): Promise<{ success: boolean; error?: string; flag?: Flag }> {
  try {
    const requestHeaders = await headers();

    const flag = await createFlag(
      tenantId,
      {
        name: data.name,
        description: data.description,
        enabled: data.enabled,
        rules: [],
        rule_logic: "AND",
      },
      requestHeaders
    );

    // Revalidate the flags page to show the new flag
    revalidatePath(`/${slug}/flags`);

    return { success: true, flag };
  } catch (error) {
    console.error("Failed to create flag:", error);
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to create flag",
    };
  }
}
