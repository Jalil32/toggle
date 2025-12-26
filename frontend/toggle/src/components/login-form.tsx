"use client";

import { useState } from "react";
import { Eclipse } from "lucide-react";
import { AnimatePresence, motion } from "motion/react";

import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import {
  Field,
  FieldDescription,
  FieldGroup,
  FieldLabel,
  FieldSeparator,
} from "@/components/ui/field";
import { Input } from "@/components/ui/input";

export function LoginForm({
  className,
  ...props
}: React.ComponentProps<"div">) {
  const [mode, setMode] = useState<"login" | "signup">("login");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    const formData = new FormData(e.currentTarget);
    const email = formData.get("email") as string;
    const password = formData.get("password") as string;
    const confirmPassword = formData.get("confirmPassword") as string;

    // Validate password confirmation for signup
    if (mode === "signup" && password !== confirmPassword) {
      setError("Passwords do not match");
      setLoading(false);
      return;
    }

    try {
      const response = await fetch("/api/auth/credentials", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          email,
          password,
          mode,
        }),
      });

      const data = await response.json();

      if (!response.ok) {
        throw new Error(data.error || "Authentication failed");
      }

      // Redirect to home page on success
      window.location.href = "/";
    } catch (err: any) {
      setError(err.message);
      setLoading(false);
    }
  };

  const handleModeSwitch = () => {
    setMode(mode === "login" ? "signup" : "login");
    setError(null);
  };

  return (
    <AnimatePresence>
      {
        <motion.div
          initial={{ opacity: 0, height: 0, scale: 0.7 }}
          animate={{ opacity: 1, height: "auto", scale: 1 }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.15 }}
          className="overflow-hidden"
        >
          <div
            className={cn("flex flex-col gap-6 rounded-lg p-16", className)}
            {...props}
          >
            {/* Header - stays in same position */}
            <div className="flex flex-col items-center gap-4 text-center">
              <div className="flex size-16 items-center justify-center rounded-md bg-fuchsia-400">
                <Eclipse className="size-10 text-primary-foreground" />
              </div>
              <h1 className="text-xl font-medium">Log in to Toggle</h1>
              <AnimatePresence mode="wait">
                <FieldDescription>
                  {mode === "login" ? (
                    <>
                      Don&apos;t have an account?{" "}
                      <button
                        type="button"
                        onClick={handleModeSwitch}
                        className="font-medium underline underline-offset-4 hover:text-primary"
                      >
                        Sign up
                      </button>
                    </>
                  ) : (
                    <>
                      Already have an account?{" "}
                      <button
                        type="button"
                        onClick={handleModeSwitch}
                        className="font-medium underline underline-offset-4 hover:text-primary"
                      >
                        Log in
                      </button>
                    </>
                  )}
                </FieldDescription>
              </AnimatePresence>
            </div>

            {/* Form - with fixed height container */}
            <form onSubmit={handleSubmit}>
              <FieldGroup>
                <AnimatePresence>
                  {error && (
                    <motion.div
                      initial={{ opacity: 0, height: 0 }}
                      animate={{ opacity: 1, height: "auto" }}
                      exit={{ opacity: 0, height: 0 }}
                      transition={{ duration: 0.05 }}
                      className="overflow-hidden"
                    >
                      <div className="rounded-md bg-red-50 p-3 text-sm text-red-800 dark:bg-red-900/20 dark:text-red-400">
                        {error}
                      </div>
                    </motion.div>
                  )}
                </AnimatePresence>

                <Field>
                  <FieldLabel htmlFor="email">Email</FieldLabel>
                  <Input
                    id="email"
                    name="email"
                    type="email"
                    placeholder="m@example.com"
                    required
                    disabled={loading}
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="password">Password</FieldLabel>
                  <Input
                    id="password"
                    name="password"
                    type="password"
                    placeholder="••••••••"
                    required
                    disabled={loading}
                    minLength={8}
                  />
                </Field>

                {/* Confirm Password - animated */}
                <AnimatePresence>
                  {mode === "signup" && (
                    <motion.div
                      initial={{ opacity: 0, height: 0, scale: 0 }}
                      animate={{ opacity: 1, height: "auto", scale: 1 }}
                      exit={{ opacity: 0, height: 0 }}
                      transition={{ duration: 0.05 }}
                      className="overflow-hidden"
                    >
                      <Field>
                        <FieldLabel htmlFor="confirmPassword">
                          Confirm Password
                        </FieldLabel>
                        <Input
                          id="confirmPassword"
                          name="confirmPassword"
                          type="password"
                          placeholder="••••••••"
                          required
                          disabled={loading}
                          minLength={8}
                        />
                      </Field>
                    </motion.div>
                  )}
                </AnimatePresence>

                <Field>
                  <Button type="submit" className="w-full" disabled={loading}>
                    {loading
                      ? "Please wait..."
                      : mode === "login"
                        ? "Log In"
                        : "Sign Up"}
                  </Button>
                </Field>

                <FieldSeparator>Or</FieldSeparator>

                <Field>
                  <Button
                    variant="outline"
                    type="button"
                    className="w-full"
                    disabled={loading}
                    asChild
                  >
                    <a href="/auth/login?connection=google-oauth2">
                      <svg
                        alt="google-logo"
                        xmlns="http://www.w3.org/2000/svg"
                        viewBox="0 0 24 24"
                        className="mr-2 h-4 w-4"
                      >
                        <path
                          d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"
                          fill="currentColor"
                        />
                      </svg>
                      Continue with Google
                    </a>
                  </Button>
                </Field>
              </FieldGroup>
            </form>

            {/* Footer - stays in same position */}
            <FieldDescription className="text-center text-xs">
              By clicking continue, you agree to our{" "}
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
      }
    </AnimatePresence>
  );
}
