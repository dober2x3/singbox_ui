"use client"

import { I18nProvider } from "@/lib/i18n"

/**
 * Root providers wrapper that sets up internationalization context.
 * Renders the I18nProvider around the application children.
 */
export function Providers({ children }: { children: React.ReactNode }) {
  return <I18nProvider>{children}</I18nProvider>
}
