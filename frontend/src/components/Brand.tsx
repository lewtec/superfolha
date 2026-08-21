type BrandMarkProps = {
  className?: string;
  alt?: string;
};

export function BrandMark({ className = "h-8 w-8", alt = "" }: BrandMarkProps) {
  return (
    <img
      src="/logo.png"
      alt={alt}
      width={512}
      height={512}
      decoding="async"
      className={className}
    />
  );
}
