import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

const RELATIVE_TIME_UNITS: [Intl.RelativeTimeFormatUnit, number][] = [
  ["year", 60 * 60 * 24 * 365],
  ["month", 60 * 60 * 24 * 30],
  ["week", 60 * 60 * 24 * 7],
  ["day", 60 * 60 * 24],
  ["hour", 60 * 60],
  ["minute", 60],
]

const relativeTimeFormatter = new Intl.RelativeTimeFormat(undefined, { numeric: "auto" })

// "2h ago" / "in 3 days" style label. Callers should also render the exact
// timestamp (e.g. via a `title` tooltip) since relative labels go stale.
export function formatRelativeTime(date: Date | string | number): string {
  const seconds = Math.round((new Date(date).getTime() - Date.now()) / 1000)
  for (const [unit, secondsInUnit] of RELATIVE_TIME_UNITS) {
    if (Math.abs(seconds) >= secondsInUnit) {
      return relativeTimeFormatter.format(Math.round(seconds / secondsInUnit), unit)
    }
  }
  return relativeTimeFormatter.format(seconds, "second")
}
