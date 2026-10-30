import { brandOf } from "@/lib/driver-brand";
import { cn } from "@/lib/utils";

// DriverMark is a driver's monogram tile in its brand colour. The colour is a theme token and the tile is that colour at
// low opacity, so one asset reads on a light and a dark surface; the name always sits beside it, so it is decorative.
export function DriverMark({ driver, className }: { driver?: string; className?: string }) {
  const brand = brandOf(driver);
  return (
    <span
      aria-hidden="true"
      data-slot="driver-mark"
      className={cn(
        "inline-flex size-5 shrink-0 items-center justify-center rounded-[5px] text-[10px] font-semibold",
        className,
      )}
      style={{
        color: `var(${brand.token})`,
        backgroundColor: `color-mix(in oklab, var(${brand.token}) 16%, transparent)`,
      }}
    >
      {brand.monogram}
    </span>
  );
}
