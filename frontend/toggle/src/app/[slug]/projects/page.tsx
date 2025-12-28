import { ProjectsTable } from "@/components/projects-table";
import { CreateProjectDialog } from "@/components/create-project-dialog";
import { SiteHeader } from "@/components/site-header";
import { Button } from "@/components/ui/button";
import { getTenants, getProjects } from "@/lib/api-server";
import { headers } from "next/headers";

interface PageProps {
  params: Promise<{
    slug: string;
  }>;
}

export default async function ProjectsPage({ params }: PageProps) {
  const { slug } = await params;
  const requestHeaders = await headers();

  // Get tenant
  const tenants = await getTenants(requestHeaders);
  const tenant = tenants.find((t) => t.slug === slug);

  if (!tenant) {
    return <div>Tenant not found</div>;
  }

  // Load all projects for this tenant
  const projects = await getProjects(tenant.id, requestHeaders);

  return (
    <>
      <SiteHeader
        title="Projects"
        actionButton={
          <CreateProjectDialog slug={slug} tenantId={tenant.id}>
            <Button variant="secondary" size="sm" className="hidden sm:flex">
              Create Project
            </Button>
          </CreateProjectDialog>
        }
      />
      <div className="flex flex-1 flex-col overflow-hidden">
        <div className="@container/main flex flex-1 flex-col overflow-hidden py-6">
          <ProjectsTable data={projects} slug={slug} />
        </div>
      </div>
    </>
  );
}
