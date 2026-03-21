import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}


/**
 * Safely ensure a value is an array. Handles Go nil → JSON null.
 * Usage: safeArray(goResult).map(...)
 */
export function safeArray<T>(val: T[] | null | undefined): T[] {
  return Array.isArray(val) ? val : []
}
