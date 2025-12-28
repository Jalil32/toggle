import { AppSidebar } from "@/components/app-sidebar";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { auth } from "@/lib/auth";
import { getTenants } from "@/lib/api-server";
import { redirect } from "next/navigation";
import { headers } from "next/headers";

interface LayoutProps {
  children: React.ReactNode;
  params: Promise<{
    slug: string;
  }>;
}

export default async function SlugLayout({ children, params }: LayoutProps) {
  const { slug } = await params;
  const requestHeaders = await headers();

  // Check if user is authenticated
  const session = await auth.api.getSession({
    headers: requestHeaders,
  });

  // Redirect to login if not authenticated
  if (!session) {
    redirect("/login");
  }

  // Check if user has access to this organization
  const tenants = await getTenants(requestHeaders);

  // Redirect to onboarding if user has no organizations
  if (!tenants || tenants.length === 0) {
    redirect("/onboarding/create-organization");
  }

  // Check if user has access to this specific organization
  const tenant = tenants.find((t) => t.slug === slug);
  if (!tenant) {
    // User doesn't have access to this organization
    // Redirect to their first organization's dashboard
    redirect(`/${tenants[0].slug}/dashboard`);
  }

  return (
    <SidebarProvider
      style={
        {
          "--sidebar-width": "calc(var(--spacing) * 72)",
          "--header-height": "calc(var(--spacing) * 12)",
        } as React.CSSProperties
      }
    >
      <AppSidebar variant="sidebar" />
      <SidebarInset>{children}</SidebarInset>
    </SidebarProvider>
  );
}
