import { NextRequest, NextResponse } from "next/server";

export async function POST(request: NextRequest) {
  try {
    const { email, password, mode } = await request.json();

    if (!email || !password) {
      return NextResponse.json(
        { error: "Email and password are required" },
        { status: 400 },
      );
    }

    const domain = process.env.AUTH0_DOMAIN;

    if (mode === "signup") {
      // Handle signup
      const signupResponse = await fetch(
        `https://${domain}/dbconnections/signup`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({
            client_id: process.env.AUTH0_CLIENT_ID,
            email,
            password,
            connection:
              process.env.AUTH0_DB_CONNECTION ||
              "Username-Password-Authentication",
          }),
        },
      );

      if (!signupResponse.ok) {
        const error = await signupResponse.json();
        return NextResponse.json(
          { error: error.description || error.message || "Signup failed" },
          { status: signupResponse.status },
        );
      }

      // After successful signup, proceed to login
    }

    // Handle login using Resource Owner Password Grant
    const tokenResponse = await fetch(`https://${domain}/oauth/token`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        grant_type: "http://auth0.com/oauth/grant-type/password-realm",
        username: email,
        password,
        client_id: process.env.AUTH0_CLIENT_ID,
        client_secret: process.env.AUTH0_CLIENT_SECRET,
        realm:
          process.env.AUTH0_DB_CONNECTION || "Username-Password-Authentication",
        scope: "openid profile email",
      }),
    });

    if (!tokenResponse.ok) {
      const error = await tokenResponse.json();
      return NextResponse.json(
        { error: error.error_description || "Authentication failed" },
        { status: tokenResponse.status },
      );
    }

    const tokens = await tokenResponse.json();

    // Create a response with the tokens
    // The Auth0 Next.js SDK expects the session to be created via the callback
    // So we'll return the tokens and handle session creation on the client
    const response = NextResponse.json({
      success: true,
      redirectUrl: "/auth/callback-credentials",
      tokens,
    });

    return response;
  } catch (error: any) {
    console.error("Authentication error:", error);
    return NextResponse.json(
      { error: error.message || "Internal server error" },
      { status: 500 },
    );
  }
}
