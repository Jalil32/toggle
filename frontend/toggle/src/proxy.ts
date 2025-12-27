import type { NextRequest, NextResponse } from "next/server";

export async function proxy(
  request: NextRequest,
): Promise<NextResponse | undefined> {
  // Middleware - add your authentication logic here
  return undefined;
}

export const config = {
  matcher: [
    /*
     * Match all request paths except for the ones starting with:
     * - _next/static (static files)
     * - _next/image (image optimization files)
     * - favicon.ico, sitemap.xml, robots.txt (metadata files)
     */
    "/((?!_next/static|_next/image|favicon.ico|sitemap.xml|robots.txt).*)",
  ],
};
