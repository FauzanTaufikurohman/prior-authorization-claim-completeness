import Image from "next/image";

type BrandLogoProps = {
  alt?: string;
  className?: string;
  height?: number;
  priority?: boolean;
  width?: number;
};

export function BrandLogo({
  alt = "Logo Ayuprama",
  className,
  height = 40,
  priority = false,
  width = 48,
}: BrandLogoProps) {
  return (
    <Image
      src="/images/logo-pa.jpeg"
      alt={alt}
      width={width}
      height={height}
      sizes={`${width}px`}
      priority={priority}
      className={className}
    />
  );
}
