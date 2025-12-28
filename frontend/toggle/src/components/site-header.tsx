import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { ThemeToggle } from "@/components/theme-toggle";

interface SiteHeaderProps {
  title?: string;
  actionButton?: React.ReactNode;
}

export function SiteHeader({
  title = "Documents",
  actionButton,
}: SiteHeaderProps) {
  return (
    <header className="flex h-(--header-height) shrink-0 items-center gap-2 border-b transition-[width,height] ease-linear group-has-data-[collapsible=icon]/sidebar-wrapper:h-(--header-height)">
      <div className="flex w-full items-center gap-1 px-4 lg:gap-2 lg:px-6">
        <SidebarTrigger className="-ml-1" />
        <Separator
          orientation="vertical"
          className="mx-2 data-[orientation=vertical]:h-4"
        />
        <h1 className="text-base font-medium">{title}</h1>
        {actionButton && (
          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
            {actionButton}
          </div>
        )}
        {!actionButton && (
          <div className="ml-auto flex items-center gap-2">
            <ThemeToggle />
            <Button
              variant="secondary"
              asChild
              size="sm"
              className="hidden sm:flex"
            >
              <div>Create Gate</div>
            </Button>
          </div>
        )}
      </div>
    </header>
  );
}
