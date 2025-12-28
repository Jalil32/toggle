import Image from "next/image";
import { cn } from "@/lib/utils";

interface LogoProps {
	className?: string;
	size?: number;
}

export function Logo({ className, size = 40 }: LogoProps) {
	return (
		<div
			className={cn(
				"inline-flex items-center justify-center rounded-md border border-border/40 dark:border-border/60",
				className,
			)}
			style={{ width: size, height: size }}
		>
			<Image
				src="/toggle.svg"
				alt="Toggle Logo"
				width={size}
				height={size}
				className="rounded-[3px]"
				priority
			/>
		</div>
	);
}
