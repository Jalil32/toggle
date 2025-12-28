import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { headers } from "next/headers";

export default async function Home() {
  // Check if user is authenticated
  const session = await auth.api.getSession({
    headers: await headers(),
  });

  // Redirect authenticated users to dashboard
  if (session) {
    redirect("/dashboard");
  }

  // Landing page for unauthenticated users
  return (
    <div className="min-h-screen flex flex-col">
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-md text-center space-y-6">
          <div className="space-y-2">
            <h1 className="text-4xl font-bold">Welcome to Toggle</h1>
            <p className="text-muted-foreground">
              Please sign in to access your account
            </p>
          </div>

          <div className="space-y-3">
            <a
              href="/login"
              className="inline-flex items-center justify-center rounded-md text-sm font-medium bg-primary text-primary-foreground hover:bg-primary/90 h-10 px-8 w-full"
            >
              Sign In / Sign Up
            </a>
          </div>
        </div>
      </div>
    </div>
  );
}
