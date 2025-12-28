import { auth } from "./auth";
import { APIError } from "./api";
import type { Tenant, CreateTenantRequest } from "./api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

/**
 * Server-side API request function with JWT authentication
 * This should only be used in Server Components and Server Actions
 */
async function apiRequestServer<T>(
  endpoint: string,
  options: RequestInit = {},
  headers: Headers,
): Promise<T> {
  try {
    // Get JWT token from Better Auth server-side
    // According to Better Auth docs, when calling getSession with returnHeaders: true,
    // a JWT is returned in the 'set-auth-jwt' header
    let token: string | null = null;
    try {
      // Call getSession with returnHeaders option to get the JWT from response headers
      const { headers: responseHeaders } = await auth.api.getSession({
        headers: headers,
        returnHeaders: true,
      });

      // Extract JWT from set-auth-jwt header
      token = responseHeaders.get("set-auth-jwt");
    } catch (error) {
      // Continue without token - backend will return 401 if auth is required
    }

    const url = `${API_BASE_URL}${endpoint}`;
    const requestHeaders: HeadersInit = {
      "Content-Type": "application/json",
      ...options.headers,
    };

    // Add authorization header if token exists
    if (token) {
      requestHeaders.Authorization = `Bearer ${token}`;
    }

    const response = await fetch(url, {
      ...options,
      headers: requestHeaders,
    });

    if (!response.ok) {
      let errorBody;
      try {
        errorBody = await response.json();
      } catch {
        errorBody = await response.text();
      }

      const errorMessage = errorBody?.error || getErrorMessage(response.status, response.statusText);

      throw new APIError(
        errorMessage,
        response.status,
        errorBody
      );
    }

    // Handle 204 No Content
    if (response.status === 204) {
      return {} as T;
    }

    return await response.json();
  } catch (error) {
    if (error instanceof APIError) {
      throw error;
    }

    throw new APIError(
      error instanceof Error ? error.message : "An unexpected error occurred",
      500
    );
  }
}

/**
 * Get user-friendly error message based on status code
 */
function getErrorMessage(status: number, defaultMessage: string): string {
  switch (status) {
    case 401:
      return "You are not authenticated. Please log in again.";
    case 403:
      return "You don't have permission to perform this action.";
    case 404:
      return "The requested resource was not found.";
    case 409:
      return "This resource already exists or conflicts with existing data.";
    case 422:
      return "The data you provided is invalid.";
    case 500:
      return "An internal server error occurred. Please try again later.";
    case 503:
      return "The service is temporarily unavailable. Please try again later.";
    default:
      return defaultMessage || "An unexpected error occurred.";
  }
}

/**
 * Get all tenants/organizations that the current user belongs to
 * Server-side only - use in Server Components and Server Actions
 */
export async function getTenants(headers: Headers): Promise<Tenant[]> {
  return apiRequestServer<Tenant[]>("/me/tenants", {}, headers);
}

/**
 * Create a new tenant/organization with the current user as owner
 * Server-side only - use in Server Components and Server Actions
 */
export async function createTenant(name: string, headers: Headers): Promise<Tenant> {
  return apiRequestServer<Tenant>("/me/tenants", {
    method: "POST",
    body: JSON.stringify({ name } as CreateTenantRequest),
  }, headers);
}
