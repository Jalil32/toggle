import { FlagDetailView } from "@/components/flag-detail-view";
import { SiteHeader } from "@/components/site-header";
import {
  Breadcrumb,
  BreadcrumbList,
  BreadcrumbItem,
  BreadcrumbLink,
  BreadcrumbSeparator,
  BreadcrumbPage,
} from "@/components/ui/breadcrumb";
import { getTenants, getFlagById } from "@/lib/api-server";
import { notFound } from "next/navigation";
import { headers } from "next/headers";
import Link from "next/link";

interface PageProps {
  params: Promise<{
    slug: string;
    flagId: string;
  }>;
}

export default async function FlagDetailPage({ params }: PageProps) {
  const { slug, flagId } = await params;
  const requestHeaders = await headers();

  // Get tenant
  const tenants = await getTenants(requestHeaders);
  const tenant = tenants.find((t) => t.slug === slug);

  if (!tenant) {
    notFound();
  }

  // Load flag data from backend
  let flag;
  try {
    flag = await getFlagById(tenant.id, flagId, requestHeaders);
  } catch (error) {
    notFound();
  }

  return (
    <>
      <SiteHeader title={flag.name} />
      <div className="flex flex-1 flex-col">
        <div className="@container/main flex flex-1 flex-col gap-2">
          <div className="flex flex-col gap-4 px-12 py-4 md:gap-6 md:py-6 lg:px-24">
            <Breadcrumb>
              <BreadcrumbList>
                <BreadcrumbItem>
                  <BreadcrumbLink asChild>
                    <Link href={`/${slug}/flags`}>Feature Flags</Link>
                  </BreadcrumbLink>
                </BreadcrumbItem>
                <BreadcrumbSeparator />
                <BreadcrumbItem>
                  <BreadcrumbPage>{flag.name}</BreadcrumbPage>
                </BreadcrumbItem>
              </BreadcrumbList>
            </Breadcrumb>

            <FlagDetailView flag={flag} tenantId={tenant.id} />
          </div>
        </div>
      </div>
    </>
  );
}
