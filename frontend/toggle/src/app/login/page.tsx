import { LoginForm } from "@/components/login-form";
import { ThemeToggle } from "@/components/theme-toggle";
import { auth } from "@/lib/auth";
import { redirect } from "next/navigation";
import { headers } from "next/headers";

export default async function LoginPage() {
    // Check if user is already authenticated
    const session = await auth.api.getSession({
        headers: await headers(),
    });

    // Redirect authenticated users to dashboard
    if (session) {
        redirect("/dashboard");
    }

    return (
        <div className="flex min-h-screen w-full items-center justify-center bg-background p-6 md:p-10">
            <div className="w-full max-w-md">
                <LoginForm />
            </div>
        </div>
    );
}
