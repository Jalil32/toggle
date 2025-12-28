import { betterAuth } from "better-auth";
import { magicLink, jwt } from "better-auth/plugins";
import { Pool } from "pg";
import { sendMagicLinkEmail } from "./email";

export const auth = betterAuth({
    database: new Pool({
        connectionString: process.env.DATABASE_URL,
    }),
    // Map table names and column names to snake_case
    user: {
        modelName: "users",
        fields: {
            emailVerified: "email_verified",
            createdAt: "created_at",
            updatedAt: "updated_at",
        },
        additionalFields: {
            lastActiveTenantId: {
                type: "string",
                required: false,
                fieldName: "last_active_tenant_id",
            },
        },
    },
    databaseHooks: {
        user: {
            create: {
                before: async (user) => {
                    // If name is not provided, use email as the default name
                    if (!user.name || user.name.trim() === "") {
                        return {
                            data: {
                                ...user,
                                name: user.email,
                            },
                        };
                    }
                    return { data: user };
                },
            },
        },
    },
    session: {
        modelName: "session",
        fields: {
            userId: "user_id",
            expiresAt: "expires_at",
            ipAddress: "ip_address",
            userAgent: "user_agent",
            createdAt: "created_at",
            updatedAt: "updated_at",
        },
    },
    account: {
        modelName: "account",
        fields: {
            userId: "user_id",
            accountId: "account_id",
            providerId: "provider_id",
            accessToken: "access_token",
            refreshToken: "refresh_token",
            accessTokenExpiresAt: "access_token_expires_at",
            refreshTokenExpiresAt: "refresh_token_expires_at",
            idToken: "id_token",
            createdAt: "created_at",
            updatedAt: "updated_at",
        },
    },
    verification: {
        modelName: "verification",
        fields: {
            expiresAt: "expires_at",
            createdAt: "created_at",
            updatedAt: "updated_at",
        },
    },
    socialProviders: {
        google: {
            clientId: process.env.GOOGLE_CLIENT_ID as string,
            clientSecret: process.env.GOOGLE_CLIENT_SECRET as string,
        },
    },
    advanced: {
        database: {
            generateId: "uuid", // Use UUIDs for all tables
        },
    },
    plugins: [
        magicLink({
            sendMagicLink: async ({ email, url }) => {
                await sendMagicLinkEmail({
                    to: email,
                    url,
                });
            },
            expiresIn: 300, // 5 minutes
        }),
        jwt({
            // JWT issuer and audience should match the backend configuration
            // These default to BETTER_AUTH_URL if not specified
            issuer: process.env.BETTER_AUTH_URL || process.env.NEXT_PUBLIC_APP_URL,
            audience: process.env.BETTER_AUTH_URL || process.env.NEXT_PUBLIC_APP_URL,
            jwt: {
                // Define custom payload to match backend expectations
                // Backend expects: { userId: string, email: string, name: string }
                definePayload: ({ user }) => {
                    return {
                        userId: user.id,  // Map 'id' to 'userId' for backend compatibility
                        email: user.email,
                        name: user.name,
                    };
                },
            },
            schema: {
                jwks: {
                    modelName: "jwks",
                    fields: {
                        publicKey: "public_key",
                        privateKey: "private_key",
                        createdAt: "created_at",
                    },
                },
            },
        }),
    ],
});
