"use client";

import { Button } from "@/components/ui/button";

export default function LoginButton() {
  return (
    <Button asChild className="w-full">
      <a href="/login">Log In</a>
    </Button>
  );
}
