import type { NextRequest, NextResponse } from "next/server";

export async function proxy(
  request: NextRequest,
): Promise<NextResponse | undefined> {
  // Proxy middleware - currently not used
  // If you need to add middleware logic, implement it here
  return undefined;
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico, sitemap.xml, robots.txt (metadata files)
     * - login (custom login page)
     * - api/auth (Auth0 authentication routes)
     */
    "/((?!_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt|login|api/auth).*)",
  ],
};
