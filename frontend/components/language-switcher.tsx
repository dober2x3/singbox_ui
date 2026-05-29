"use client"

import { useI18n } from "@/lib/i18n"
import { Button } from "@/components/ui/button"
import { Globe } from "lucide-react"

export function LanguageSwitcher() {
  const { locale, setLocale } = useI18n()

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={() => setLocale(locale === "zh" ? "en" : "zh")}
      className="gap-1.5"
    >
      <Globe className="h-4 w-4" />
      <span className="text-xs">{locale === "zh" ? "EN" : "ZH"}</span>
    </Button>
  )
}
