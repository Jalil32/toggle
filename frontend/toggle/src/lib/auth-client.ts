import { createAuthClient } from "better-auth/react";
import { magicLinkClient, jwtClient } from "better-auth/client/plugins";

export const authClient = createAuthClient({
    baseURL: process.env.NEXT_PUBLIC_APP_URL || "http://localhost:3000",
    plugins: [magicLinkClient(), jwtClient()],
});

export const {
    signIn,
    signOut,
    signUp,
    useSession,
    user,
    organization,
    getSession,
} = authClient;
