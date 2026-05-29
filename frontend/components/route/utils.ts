// Helper utilities for routing config

// Parse textarea lines to array (filter empty lines and comments)
export const parseLines = (text: string): string[] => {
  return text
    .split(/\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith("#"))
}

// Normalize IPs to CIDR (plain IPs get /32 or /128 appended)
export const normalizeIpCidrs = (ips: string[]): string[] => {
  return ips.map((ip) => {
    if (ip.includes("/")) return ip
    return ip.includes(":") ? `${ip}/128` : `${ip}/32`
  })
}

// Domain match classification results
export interface DomainGroups {
  domain: string[]        // full: exact match
  domain_suffix: string[] // default, suffix match
  domain_keyword: string[] // keyword: keyword match
  domain_regex: string[]  // regex: regex match
}

// Parse domain lines, classify by prefix:
//   full:xxx     → domain (exact)
//   keyword:xxx  → domain_keyword
//   regex:xxx    → domain_regex
//   suffix:xxx   → domain_suffix (explicit)
//   xxx          → domain_suffix (default)
export const parseDomainLines = (text: string): DomainGroups => {
  const groups: DomainGroups = { domain: [], domain_suffix: [], domain_keyword: [], domain_regex: [] }
  for (const line of parseLines(text)) {
    if (line.startsWith("full:")) {
      groups.domain.push(line.slice(5))
    } else if (line.startsWith("keyword:")) {
      groups.domain_keyword.push(line.slice(8))
    } else if (line.startsWith("regex:")) {
      groups.domain_regex.push(line.slice(6))
    } else if (line.startsWith("suffix:")) {
      groups.domain_suffix.push(line.slice(7))
    } else {
      groups.domain_suffix.push(line)
    }
  }
  return groups
}

// Restore domain fields from sing-box rules to prefixed text lines
export const domainFieldsToLines = (rule: {
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
}): string[] => {
  const lines: string[] = []
  for (const d of rule.domain || []) lines.push(`full:${d}`)
  for (const d of rule.domain_suffix || []) lines.push(d)
  for (const d of rule.domain_keyword || []) lines.push(`keyword:${d}`)
  for (const d of rule.domain_regex || []) lines.push(`regex:${d}`)
  return lines
}

// Merge DomainGroups into RouteRule object (only add non-empty fields)
export const applyDomainGroups = (rule: Record<string, any>, groups: DomainGroups) => {
  if (groups.domain.length > 0) rule.domain = groups.domain
  if (groups.domain_suffix.length > 0) rule.domain_suffix = groups.domain_suffix
  if (groups.domain_keyword.length > 0) rule.domain_keyword = groups.domain_keyword
  if (groups.domain_regex.length > 0) rule.domain_regex = groups.domain_regex
}

// Whether DomainGroups has any content
export const hasDomainEntries = (groups: DomainGroups): boolean => {
  return groups.domain.length > 0 || groups.domain_suffix.length > 0 ||
    groups.domain_keyword.length > 0 || groups.domain_regex.length > 0
}
