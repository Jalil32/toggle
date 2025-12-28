"use server";

import { revalidatePath } from "next/cache";
import { headers } from "next/headers";
import type { Project } from "@/lib/api-server";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

export async function createProjectAction(
  slug: string,
  tenantId: string,
  name: string
): Promise<{ success: boolean; error?: string; project?: Project }> {
  try {
    const requestHeaders = await headers();

    // Get JWT token from Better Auth
    const { auth } = await import("@/lib/auth");
    let token: string | null = null;
    try {
      const { headers: responseHeaders } = await auth.api.getSession({
        headers: requestHeaders,
        returnHeaders: true,
      });
      token = responseHeaders.get("set-auth-jwt");
    } catch (error) {
      // Continue without token
    }

    const response = await fetch(`${API_BASE_URL}/projects`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Tenant-ID": tenantId,
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
      body: JSON.stringify({ name }),
    });

    if (!response.ok) {
      const error = await response.json();
      return {
        success: false,
        error: error.error || "Failed to create project",
      };
    }

    const project = await response.json();

    // Revalidate the projects page to show the new project
    revalidatePath(`/${slug}/projects`);

    return { success: true, project };
  } catch (error) {
    console.error("Failed to create project:", error);
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to create project",
    };
  }
}
