import { AppSidebar } from "@/components/app-sidebar";
import { ChartAreaInteractive } from "@/components/chart-area-interactive";
import { DataTable } from "@/components/data-table";
import { SectionCards } from "@/components/section-cards";
import { SiteHeader } from "@/components/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { headers } from "next/headers";
import { getTenants } from "@/lib/api-server";

import data from "./data.json";

interface PageProps {
  params: Promise<{
    slug: string;
  }>;
}

export default async function Page({ params }: PageProps) {
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
      <SidebarInset>
        <SiteHeader />
        <div className="flex flex-1 flex-col">
          <div className="@container/main flex flex-1 flex-col gap-2">
            <div className="flex flex-col gap-4 py-4 md:gap-6 md:py-6">
              <DataTable data={data} />
            </div>
          </div>
        </div>
      </SidebarInset>
    </SidebarProvider>
  );
}
