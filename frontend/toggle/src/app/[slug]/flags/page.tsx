import { FlagsTable } from "@/components/flags-table";
import { CreateFlagDialog } from "@/components/create-flag-dialog";
import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";
import { getTenants, getFlags } from "@/lib/api-server";
import { headers } from "next/headers";

interface PageProps {
  params: Promise<{
    slug: string;
  }>;
}

export default async function FlagsPage({ params }: PageProps) {
  const { slug } = await params;
  const requestHeaders = await headers();

  // Get tenant
  const tenants = await getTenants(requestHeaders);
  const tenant = tenants.find((t) => t.slug === slug);

  if (!tenant) {
    return <div>Tenant not found</div>;
  }

  // Load all flags for this tenant (not filtered by project)
  const flags = await getFlags(tenant.id, null, requestHeaders);

  return (
    <>
      <SiteHeader
        title="Feature Flags"
        actionButton={
          <CreateFlagDialog slug={slug} tenantId={tenant.id}>
            <Button variant="secondary" size="sm" className="hidden sm:flex">
              Create Flag
            </Button>
          </CreateFlagDialog>
        }
      />
      <div className="flex flex-1 flex-col overflow-hidden">
        <div className="@container/main flex flex-1 flex-col overflow-hidden py-6">
          <FlagsTable data={flags} slug={slug} />
        </div>
      </div>
    </>
  );
}
