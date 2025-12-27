const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1";

export interface Tenant {
  id: string;
  name: string;
  slug: string;
  created_at: string;
  updated_at: string;
}

export interface CreateTenantRequest {
  name: string;
}

export class APIError extends Error {
  constructor(
    message: string,
    public status: number,
    public body?: any
  ) {
    super(message);
    this.name = "APIError";
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
 * Generic API request function with JWT authentication
 */
async function apiRequest<T>(
	endpoint: string,
	options: RequestInit = {},
): Promise<T> {
	try {
		// Get JWT token from Better Auth
		let token: string | null = null;
		try {
			// Dynamically import to avoid circular dependencies
			const { authClient } = await import("./auth-client");
			const tokenResponse = await authClient.token();
			if (tokenResponse.data) {
				token = tokenResponse.data.token;
			}
		} catch (error) {
			// Continue without token - backend will return 401 if auth is required
		}

		const url = `${API_BASE_URL}${endpoint}`;
		const headers: HeadersInit = {
			"Content-Type": "application/json",
			...options.headers,
		};

		// Add authorization header if token exists
		if (token) {
			headers.Authorization = `Bearer ${token}`;
		}

    const response = await fetch(url, {
      ...options,
      headers,
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
 * Get all tenants/organizations that the current user belongs to
 */
export async function getTenants(): Promise<Tenant[]> {
  return apiRequest<Tenant[]>("/me/tenants");
}

/**
 * Create a new tenant/organization with the current user as owner
 */
export async function createTenant(name: string): Promise<Tenant> {
  return apiRequest<Tenant>("/me/tenants", {
    method: "POST",
    body: JSON.stringify({ name } as CreateTenantRequest),
  });
}
