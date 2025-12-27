"use server";

import {
  getTenants,
  createTenant as apiCreateTenant,
} from "@/lib/api-server";
import { type Tenant, APIError } from "@/lib/api";
import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import { headers } from "next/headers";

/**
 * Server Action to fetch user's tenants
 * Can be called from Client Components
 */
export async function getUserTenants(): Promise<{
  success: boolean;
  tenants?: Tenant[];
  error?: string;
}> {
  try {
    const requestHeaders = await headers();
    const tenants = await getTenants(requestHeaders);
    return { success: true, tenants };
  } catch (error) {
    console.error("Error fetching tenants:", error);

    if (error instanceof APIError) {
      // If unauthorized, redirect to login
      if (error.status === 401) {
        redirect("/login");
      }
      return { success: false, error: error.message };
    }

    return { success: false, error: "Failed to fetch organizations" };
  }
}

/**
 * Server Action to create a new tenant
 * Can be called from Client Components (forms)
 */
export async function createTenantAction(
  name: string,
): Promise<{
  success: boolean;
  tenant?: Tenant;
  slug?: string;
  error?: string;
}> {
  try {
    const requestHeaders = await headers();
    const tenant = await apiCreateTenant(name, requestHeaders);

    // Revalidate paths that might show tenant data
    revalidatePath("/");
    revalidatePath("/dashboard");

    return { success: true, tenant, slug: tenant.slug };
  } catch (error: any) {
    console.error("Error creating tenant:", error);

    if (error instanceof APIError) {
      // If unauthorized, redirect to login
      if (error.status === 401) {
        redirect("/login");
      }

      // Return the specific error message from the API
      return {
        success: false,
        error: error.message,
      };
    }

    return {
      success: false,
      error: error.message || "Failed to create organization",
    };
  }
}
