import { redirect } from "next/navigation"; // [!code ++]
import { auth0 } from "@/lib/auth0";

export default async function Home() {
  const session = await auth0.getSession();

  // 1. If user is logged in, send them to the dashboard immediately
  if (session?.user) {
    redirect("/dashboard");
  }

  // 2. If no session, show the landing page
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
              href="/auth/login"
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
