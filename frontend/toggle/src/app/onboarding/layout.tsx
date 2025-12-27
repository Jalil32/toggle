"use client";

import { AnimatePresence } from "motion/react";

export default function OnboardingLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="min-h-screen flex flex-col">
      <div className="flex-1 flex items-center justify-center p-6">
        <div className="w-full max-w-md">
          <AnimatePresence mode="wait">{children}</AnimatePresence>
        </div>
      </div>
    </div>
  );
}
