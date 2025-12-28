"use client";

import * as React from "react";
import {
  IconChartBar,
  IconChartDots,
  IconHelp,
  IconUserScan,
  IconSettings,
  IconToggleRight,
  IconUser,
} from "@tabler/icons-react";
import { useParams } from "next/navigation";

import { NavMain } from "@/components/nav-main";
import { NavSecondary } from "@/components/nav-secondary";
import { NavUser } from "@/components/nav-user";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuItem,
} from "@/components/ui/sidebar";
import { Logo } from "@/components/logo";

const staticData = {
  user: {
    name: "shadcn",
    email: "m@example.com",
    avatar: "/avatars/shadcn.jpg",
  },
  navSecondary: [
    {
      title: "Settings",
      url: "#",
      icon: IconSettings,
    },
    {
      title: "Get Help",
      url: "#",
      icon: IconHelp,
    },
  ],
};

export function AppSidebar({ ...props }: React.ComponentProps<typeof Sidebar>) {
  const params = useParams();
  const slug = params?.slug as string | undefined;

  const navMain = [
    {
      title: "Feature Flags",
      url: slug ? `/${slug}/flags` : "#",
      icon: IconToggleRight,
    },
    {
      title: "Projects",
      url: slug ? `/${slug}/projects` : "#",
      icon: IconChartDots,
    },
    {
      title: "Analytics",
      url: "#",
      icon: IconChartBar,
    },
    {
      title: "Users",
      url: "#",
      icon: IconUserScan,
    },
  ];

  return (
    <Sidebar collapsible="offcanvas" {...props}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <div className="flex items-center gap-2 pl-4 pt-4">
              <Logo size={24} />
              <span className="text-base font-semibold">Toggle</span>
            </div>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent className="pl-3">
        <NavMain items={navMain} />
        <NavSecondary
          items={staticData.navSecondary}
          className="mt-auto"
        />
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={staticData.user} />
      </SidebarFooter>
    </Sidebar>
  );
}
