"use client"

import { createContext, useContext, useState, useEffect, useCallback, ReactNode } from "react"
import zh from "@/messages/zh.json"
import en from "@/messages/en.json"

type Locale = "zh" | "en"
type Messages = typeof zh

const messages: Record<Locale, Messages> = { zh, en }

/** Context value providing locale state and the translation function. */
interface I18nContextType {
  locale: Locale
  setLocale: (locale: Locale) => void
  t: (key: string, params?: Record<string, string | number>) => string
}

const I18nContext = createContext<I18nContextType | null>(null)

/** Retrieves a deeply nested value from an object by dot-separated path. */
function getNestedValue(obj: any, path: string): string | undefined {
  return path.split(".").reduce((acc, key) => acc?.[key], obj)
}

/** Reads the initial locale from localStorage or falls back to browser language. */
function getInitialLocale(): Locale {
  if (typeof window !== "undefined") {
    const saved = localStorage.getItem("locale") as Locale
    if (saved === "zh" || saved === "en") return saved
    return navigator.language.startsWith("zh") ? "zh" : "en"
  }
  return "zh"
}

/** Provider component that wraps children with I18nContext. */
export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(getInitialLocale)

  useEffect(() => {
    document.documentElement.lang = locale === "zh" ? "zh-CN" : "en"
  }, [locale])

  /** Persists the locale choice and updates the document language attribute. */
  const setLocale = useCallback((newLocale: Locale) => {
    setLocaleState(newLocale)
    localStorage.setItem("locale", newLocale)
    document.documentElement.lang = newLocale === "zh" ? "zh-CN" : "en"
  }, [])

  /** Looks up a translation key with optional interpolation params, falling back to Chinese then the key itself. */
  const t = useCallback((key: string, params?: Record<string, string | number>): string => {
    let value = getNestedValue(messages[locale], key) ?? getNestedValue(messages["zh"], key) ?? key
    if (params) {
      Object.entries(params).forEach(([k, v]) => {
        value = value.replace(new RegExp(`\\{${k}\\}`, "g"), String(v))
      })
    }
    return value
  }, [locale])

  return (
    <I18nContext.Provider value={{ locale, setLocale, t }}>
      {children}
    </I18nContext.Provider>
  )
}

/** Hook to access the i18n context (locale, setLocale, and t function). */
export function useI18n() {
  const context = useContext(I18nContext)
  if (!context) {
    throw new Error("useI18n must be used within I18nProvider")
  }
  return context
}

/** Hook for scoped translations with an optional dot-separated namespace prefix. */
export function useTranslation(namespace?: string) {
  const { locale, setLocale, t } = useI18n()
  const tn = useCallback((key: string, params?: Record<string, string | number>) => {
    return t(namespace ? `${namespace}.${key}` : key, params)
  }, [t, namespace])
  return { t: tn, locale, setLocale }
}
