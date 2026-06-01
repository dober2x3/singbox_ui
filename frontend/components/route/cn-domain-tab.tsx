"use client"

import { Label } from "@/components/ui/label"
import { useTranslation } from "@/lib/i18n"

/** Props for the CnDomainTab component. */
interface CnDomainTabProps {
  enableCnDomain: boolean
  setEnableCnDomain: (v: boolean) => void
}

/** Geosite-CN domain routing toggle tab. */
export function CnDomainTab({ enableCnDomain, setEnableCnDomain }: CnDomainTabProps) {
  const { t } = useTranslation("routing")

  return (
    <div className="space-y-4">
      <div className="flex items-center space-x-2">
        <input
          type="checkbox"
          id="pw_cnDomain"
          checked={enableCnDomain}
          onChange={() => setEnableCnDomain(!enableCnDomain)}
          className="h-4 w-4 rounded border-gray-300"
        />
        <Label htmlFor="pw_cnDomain" className="text-sm font-normal cursor-pointer">
          {t("enableCnDomain")}
        </Label>
      </div>
      <p className="text-sm text-muted-foreground">{t("cnDomainDesc")}</p>
      <p className="text-xs text-muted-foreground">{t("cnDomainSource")}</p>
    </div>
  )
}
