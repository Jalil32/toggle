"use client";

import { Eclipse } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  FieldDescription,
  FieldGroup,
  Field,
} from "@/components/ui/field";

interface ProfileProps {
  user: {
    name?: string | null;
    email?: string | null;
    picture?: string | null;
  };
}

export default function Profile({ user }: ProfileProps) {
  return (
    <div className="flex min-h-screen w-full items-center justify-center bg-background p-6 md:p-10">
      <div className="w-full max-w-md">
        <div className="flex flex-col gap-6 rounded-lg p-16">
          {/* Header */}
          <div className="flex flex-col items-center gap-4 text-center">
            <div className="flex size-16 items-center justify-center rounded-md bg-fuchsia-400">
              <Eclipse className="size-10 text-primary-foreground" />
            </div>
            <h1 className="text-xl font-medium">Welcome to Toggle</h1>
          </div>

          {/* Profile Info */}
          <div className="flex flex-col items-center gap-4">
            {user.picture && (
              <img
                src={user.picture}
                alt={user.name || "User"}
                className="h-20 w-20 rounded-full object-cover"
              />
            )}
            {user.name && (
              <h2 className="text-lg font-medium text-foreground">
                {user.name}
              </h2>
            )}
            {user.email && (
              <FieldDescription className="text-center">
                {user.email}
              </FieldDescription>
            )}
          </div>

          {/* Actions */}
          <FieldGroup>
            <Field>
              <Button type="button" className="w-full" asChild>
                <a href="/dashboard">Go to Dashboard</a>
              </Button>
            </Field>

            <Field>
              <Button
                type="button"
                variant="outline"
                className="w-full"
                asChild
              >
                <a href="/api/auth/logout">Log Out</a>
              </Button>
            </Field>
          </FieldGroup>
        </div>
      </div>
    </div>
  );
}
