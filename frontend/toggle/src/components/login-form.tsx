"use client";

import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { useRouter } from "next/navigation";
import { Logo } from "@/components/logo";

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
import { authClient } from "@/lib/auth-client";

type AuthMode = "login" | "signup";

export function LoginForm({
	className,
	...props
}: React.ComponentProps<"div">) {
	const router = useRouter();
	const [mode, setMode] = useState<AuthMode>("login");
	const [loading, setLoading] = useState(false);
	const [error, setError] = useState<string | null>(null);
	const [success, setSuccess] = useState(false);

	const handleSubmit = async (e: React.FormEvent<HTMLFormElement>) => {
		e.preventDefault();
		setLoading(true);
		setError(null);
		setSuccess(false);

		const formData = new FormData(e.currentTarget);
		const email = formData.get("email") as string;
		const name = formData.get("name") as string | undefined;

		try {
			const { error } = await authClient.signIn.magicLink({
				email,
				name: mode === "signup" && name ? name : undefined,
				callbackURL: "/dashboard",
				newUserCallbackURL: mode === "signup" ? "/onboarding" : undefined,
			});

			if (error) {
				setError(error.message || "Failed to send magic link");
				setLoading(false);
				return;
			}

			setSuccess(true);
			setLoading(false);
		} catch (err: any) {
			setError(err.message || "Something went wrong");
			setLoading(false);
		}
	};

	const handleGoogleSignIn = async () => {
		setLoading(true);
		setError(null);

		try {
			await authClient.signIn.social({
				provider: "google",
				callbackURL: "/",
			});
		} catch (err: any) {
			setError(err.message || "Failed to sign in with Google");
			setLoading(false);
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
			<div
				className={cn("flex flex-col gap-6 rounded-lg p-16", className)}
				{...props}
			>
				{/* Header */}
				<div className="flex flex-col items-center gap-4 text-center">
					<Logo size={64} />
					<h1 className="text-xl font-medium">
						{success
							? "Check your email"
							: mode === "signup"
								? "Create your account"
								: "Sign in to Toggle"}
					</h1>
					<FieldDescription>
						{success
							? "We sent you a magic link. Click it to continue."
							: mode === "signup"
								? "Enter your details to create an account"
								: "Enter your email to receive a sign-in link"}
					</FieldDescription>
				</div>

				{/* Form */}
				{!success && (
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

							<AnimatePresence>
								{mode === "signup" && (
									<motion.div
										initial={{ opacity: 0, height: 0 }}
										animate={{ opacity: 1, height: "auto" }}
										exit={{ opacity: 0, height: 0 }}
										transition={{ duration: 0.15 }}
										className="overflow-hidden"
									>
										<Field>
											<FieldLabel htmlFor="name">Name</FieldLabel>
											<Input
												id="name"
												name="name"
												type="text"
												placeholder="John Doe"
												required={mode === "signup"}
												disabled={loading}
											/>
										</Field>
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
								<Button type="submit" className="w-full" disabled={loading}>
									{loading
										? "Sending link..."
										: mode === "signup"
											? "Create account"
											: "Send magic link"}
								</Button>
							</Field>

							<FieldSeparator>Or</FieldSeparator>

							<Field>
								<Button
									variant="outline"
									type="button"
									className="w-full"
									disabled={loading}
									onClick={handleGoogleSignIn}
								>
									<svg
										xmlns="http://www.w3.org/2000/svg"
										viewBox="0 0 24 24"
										className="mr-2 h-4 w-4"
										aria-label="Google logo"
									>
										<path
											d="M12.48 10.92v3.28h7.84c-.24 1.84-.853 3.187-1.787 4.133-1.147 1.147-2.933 2.4-6.053 2.4-4.827 0-8.6-3.893-8.6-8.72s3.773-8.72 8.6-8.72c2.6 0 4.507 1.027 5.907 2.347l2.307-2.307C18.747 1.44 16.133 0 12.48 0 5.867 0 .307 5.387.307 12s5.56 12 12.173 12c3.573 0 6.267-1.173 8.373-3.36 2.16-2.16 2.84-5.213 2.84-7.667 0-.76-.053-1.467-.173-2.053H12.48z"
											fill="currentColor"
										/>
									</svg>
									Continue with Google
								</Button>
							</Field>
						</FieldGroup>
					</form>
				)}

				{/* Footer */}
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

				{/* Mode Toggle */}
				{!success && (
					<FieldDescription className="text-center text-sm">
						{mode === "login" ? (
							<>
								Don't have an account?{" "}
								<button
									type="button"
									onClick={() => {
										setMode("signup");
										setError(null);
									}}
									className="font-medium text-primary hover:underline"
									disabled={loading}
								>
									Sign up
								</button>
							</>
						) : (
							<>
								Already have an account?{" "}
								<button
									type="button"
									onClick={() => {
										setMode("login");
										setError(null);
									}}
									className="font-medium text-primary hover:underline"
									disabled={loading}
								>
									Sign in
								</button>
							</>
						)}
					</FieldDescription>
				)}
			</div>
		</motion.div>
	);
}
