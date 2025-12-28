"use server";

import { updateFlag as apiUpdateFlag } from "@/lib/api-server";
import { type Flag } from "@/types/flags";
import { APIError } from "@/lib/api";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { headers } from "next/headers";

/**
 * Server Action to update a flag
 * Can be called from Client Components
 */
export async function updateFlagAction(
  tenantId: string,
  flagId: string,
  data: Partial<Flag>
): Promise<{
  success: boolean;
  flag?: Flag;
  error?: string;
}> {
  try {
    const requestHeaders = await headers();
    const flag = await apiUpdateFlag(tenantId, flagId, data, requestHeaders);

    // Revalidate the flag detail page
    revalidatePath(`/[slug]/flags/${flagId}`);

    return { success: true, flag };
  } catch (error: any) {
    console.error("Error updating flag:", error);

    if (error instanceof APIError) {
      // If unauthorized, redirect to login
      if (error.status === 401) {
        redirect("/login");
      }

      return {
        success: false,
        error: error.message,
      };
    }

    return {
      success: false,
      error: error.message || "Failed to update flag",
    };
  }
}
