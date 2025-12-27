import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { headers } from "next/headers";
import { getTenants } from "@/lib/api-server";

export default async function Page() {
  const requestHeaders = await headers();

  // Check if user is authenticated
  const session = await auth.api.getSession({
    headers: requestHeaders,
  });

  // Redirect to login if not authenticated
  if (!session) {
    redirect("/login");
  }

  // Get user's organizations and redirect to the appropriate one
  const tenants = await getTenants(requestHeaders);

  // Redirect to onboarding if user has no organizations
  if (!tenants || tenants.length === 0) {
    redirect("/onboarding/create-organization");
  }

  // Redirect to the first organization's dashboard
  // In the future, this could redirect to the user's last active organization
  redirect(`/${tenants[0].slug}/dashboard`);
}
