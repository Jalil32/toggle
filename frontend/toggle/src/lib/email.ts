import { Resend } from "resend";

const resend = new Resend(process.env.RESEND_API_KEY);

interface SendMagicLinkEmailParams {
    to: string;
    url: string;
}

export async function sendMagicLinkEmail({
    to,
    url,
}: SendMagicLinkEmailParams) {
    try {
        await resend.emails.send({
            from: process.env.EMAIL_FROM || "Toggle <onboarding@resend.dev>",
            to,
            subject: "Sign in to Toggle",
            html: `
        <div style="font-family: Arial, sans-serif; max-width: 600px; margin: 0 auto;">
          <h2>Sign in to Toggle</h2>
          <p>Click the button below to sign in to your account. This link will expire in 5 minutes.</p>
          <div style="margin: 30px 0;">
            <a href="${url}" style="background-color: #000; color: white; padding: 12px 24px; text-decoration: none; border-radius: 5px; display: inline-block;">
              Sign In
            </a>
          </div>
          <p style="color: #666; font-size: 14px;">If you didn't request this email, you can safely ignore it.</p>
          <p style="color: #666; font-size: 12px;">Or copy and paste this link: ${url}</p>
        </div>
      `,
        });
    } catch (error) {
        console.error("Failed to send magic link email:", error);
        throw new Error("Failed to send magic link email");
    }
}
