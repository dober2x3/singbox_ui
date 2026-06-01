/** Shared types for inbound protocol form components. */

/** sing-box format user types. */
export interface VLESSUser {
  uuid: string
  name?: string
  flow?: string
}

export interface VMESSUser {
  uuid: string
  name?: string
  alterId?: number
}

export interface TrojanUser {
  name?: string
  password: string
}

export interface TUICUser {
  name?: string
  uuid: string
  password?: string
}

export interface NaiveUser {
  username: string
  password: string
}

export interface Hysteria2User {
  name?: string
  password: string
}

export interface ShadowTLSUser {
  name?: string
  password: string
}

export interface AnyTLSUser {
  name?: string
  password: string
}

/** WireGuard Peer type used by local UI. */
export interface LocalPeer {
  publicKey: string
  privateKey?: string
  presharedKey?: string
  allowedIPs: string[]
  persistentKeepaliveInterval?: number
}

/** Supported QR code types for generating share links. */
export type QrCodeType = "wireguard" | "shadowsocks" | "socks5" | "vless" | "hysteria2" | "vmess" | "trojan" | "tuic"

/** Props shared by all inbound protocol form components. */
export interface ProtocolFormProps {
  initialConfig: any
  initialEndpoint?: any
  // Store actions
  setInbound: (index: number, inbound: any) => void
  setEndpoint: (index: number, endpoint: any) => void
  clearEndpoints: () => void
  // Shared state
  currentInstance: string | null
  // Callbacks
  onError: (msg: string) => void
  onShowQrCode: (content: string, type: QrCodeType, peerIndex?: number) => void
  // Server IP (shared for QR code generation)
  serverIP: string
  setServerIP: (ip: string) => void
  // Certificate management
  certLoading: boolean
  setCertLoading: (loading: boolean) => void
  certInfo: { common_name?: string; valid_to?: string } | null
  setCertInfo: (info: { common_name?: string; valid_to?: string } | null) => void
  onGenerateCert: (domain?: string) => void
  onUploadCert: () => void
}

/** Fetch the public IP of the server, used by QR code generators. */
export async function getPublicIP(
  serverIP: string,
  setServerIP: (ip: string) => void
): Promise<string> {
  if (serverIP) return serverIP
  const response = await fetch("/api/wireguard/public-ip")
  if (response.ok) {
    const data = await response.json()
    setServerIP(data.ip)
    return data.ip
  }
  throw new Error("Cannot get public IP")
}

/** Format a listen address to sing-box format (converts "0.0.0.0" to "::"). */
export function formatListen(listen: string): string {
  if (listen === "0.0.0.0" || listen === "") return "::"
  return listen
}

/** Parse a listen address from sing-box format (converts "::" to "0.0.0.0"). */
export function parseListen(listen?: string): string {
  if (listen === "::" || !listen) return "0.0.0.0"
  return listen
}
