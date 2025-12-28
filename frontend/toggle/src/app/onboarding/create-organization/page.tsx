"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { z } from "zod";
import { toast } from "sonner";
import { IconLoader2 } from "@tabler/icons-react";
import { motion } from "motion/react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Field,
  FieldGroup,
  FieldLabel,
  FieldError,
  FieldDescription,
} from "@/components/ui/field";
import { createTenantAction } from "@/app/actions/tenants";
import { Eclipse } from "lucide-react";

const organizationSchema = z.object({
  name: z
    .string()
    .min(1, "Organization name is required")
    .max(255, "Organization name must be less than 255 characters")
    .trim(),
});

type OrganizationFormData = z.infer<typeof organizationSchema>;

export default function CreateOrganizationPage() {
  const router = useRouter();
  const [isSubmitting, setIsSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    watch,
    formState: { errors },
  } = useForm<OrganizationFormData>({
    resolver: zodResolver(organizationSchema),
    defaultValues: {
      name: "",
    },
  });

  // Watch the name field to generate slug preview
  const organizationName = watch("name");
  const slugPreview = organizationName
    ? organizationName
        .toLowerCase()
        .replace(/[^a-z0-9\s-]/g, "")
        .replace(/\s+/g, "-")
        .replace(/-+/g, "-")
        .substring(0, 50)
    : "";

  const onSubmit = async (data: OrganizationFormData) => {
    setIsSubmitting(true);

    try {
      const result = await createTenantAction(data.name);

      if (result.success) {
        toast.success("Organization created successfully!");
        router.push(`/${result.slug}/dashboard`);
      } else {
        toast.error(result.error || "Failed to create organization");
      }
    } catch (error) {
      console.error("Error creating organization:", error);
      toast.error("An unexpected error occurred");
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <motion.div
      initial={{ opacity: 0, height: 0, scale: 0.7 }}
      animate={{ opacity: 1, height: "auto", scale: 1 }}
      exit={{ opacity: 0, height: 0 }}
      transition={{ duration: 0.15 }}
      className="overflow-hidden"
    >
      <div className="flex flex-col gap-6 rounded-lg p-16">
        {/* Header - stays in same position */}
        <div className="flex flex-col items-center gap-4 text-center">
          <div className="flex size-16 items-center justify-center rounded-md bg-fuchsia-400">
            <Eclipse className="size-10 text-primary-foreground" />
          </div>
          <h1 className="text-xl font-medium">Create Your Organization</h1>
          <FieldDescription>
            Get started by creating your first organization.
          </FieldDescription>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit(onSubmit)}>
          <FieldGroup>
            <Field>
              <FieldLabel htmlFor="name">Organization Name</FieldLabel>
              <Input
                id="name"
                placeholder="Acme Inc"
                autoFocus
                {...register("name")}
                disabled={isSubmitting}
              />
              {errors.name && (
                <FieldDescription className="text-red-600">
                  {errors.name.message}
                </FieldDescription>
              )}
              {slugPreview && !errors.name && (
                <FieldDescription>
                  URL slug:{" "}
                  <code className="font-mono text-xs">{slugPreview}</code>
                </FieldDescription>
              )}
            </Field>

            <Field>
              <Button
                type="submit"
                className="w-full"
                disabled={isSubmitting}
              >
                {isSubmitting ? (
                  <>
                    <IconLoader2 className="mr-2 h-4 w-4 animate-spin" />
                    Creating...
                  </>
                ) : (
                  "Create Organization"
                )}
              </Button>
            </Field>
          </FieldGroup>
        </form>

        {/* Footer - stays in same position */}
        <FieldDescription className="text-center text-xs">
          By creating an organization, you agree to our{" "}
          <a href="#" className="underline underline-offset-4">
            Terms of Service
          </a>{" "}
          and{" "}
          <a href="#" className="underline underline-offset-4">
            Privacy Policy
          </a>
          .
        </FieldDescription>
      </div>
    </motion.div>
  );
}
