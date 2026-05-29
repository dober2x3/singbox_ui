import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

/**
 * Validate whether a port number is valid
 * @param port - Port number
 * @returns Whether valid
 */
export function isValidPort(port: number | string): boolean {
  const portNum = typeof port === 'string' ? parseInt(port, 10) : port
  return !isNaN(portNum) && portNum >= 1 && portNum <= 65535
}

/**
 * Safely parse a port number
 * @param value - Input value
 * @param defaultPort - Default port
 * @returns Valid port number
 */
export function parsePort(value: string, defaultPort: number = 8080): number {
  const port = parseInt(value, 10)
  return isValidPort(port) ? port : defaultPort
}

/**
 * Validate IPv4 address format
 * @param ip - IP address string
 * @returns Whether valid
 */
export function isValidIPv4(ip: string): boolean {
  if (!ip) return false

  const ipv4Regex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/
  const match = ip.match(ipv4Regex)

  if (!match) return false

  // Check each segment is in the 0-255 range
  for (let i = 1; i <= 4; i++) {
    const segment = parseInt(match[i], 10)
    if (segment < 0 || segment > 255) return false
  }

  return true
}

/**
 * Validate listen address (supports 0.0.0.0, 127.0.0.1, or valid IPv6)
 * @param address - Listen address
 * @returns Whether valid
 */
export function isValidListenAddress(address: string): boolean {
  return isValidIPv4(address) || address === '::' || address === '::1'
}

/**
 * Enhanced fetch error handling
 * @param response - Fetch response object
 * @returns Parsed error message
 */
export async function parseErrorResponse(response: Response): Promise<string> {
  let errorMsg = `HTTP ${response.status}: ${response.statusText}`

  try {
    const contentType = response.headers.get('content-type')
    if (contentType && contentType.includes('application/json')) {
      const error = await response.json()
      errorMsg = error.message || error.error || errorMsg
    } else {
      const text = await response.text()
      if (text) {
        errorMsg = text.substring(0, 200) // Limit error message length
      }
    }
  } catch {
    // If parsing fails, use default error message
  }

  return errorMsg
}

/**
 * Generate a cryptographically secure random string
 * @param length - String length
 * @param chars - Available character set
 * @returns Random string
 */
export function generateSecureRandomString(
  length: number,
  chars: string = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
): string {
  if (typeof window === 'undefined' || !window.crypto) {
    // Fallback for server-side rendering or when crypto API is unavailable
    return Array.from({ length }, () =>
      chars.charAt(Math.floor(Math.random() * chars.length))
    ).join('')
  }

  const array = new Uint8Array(length)
  window.crypto.getRandomValues(array)

  return Array.from(array, (byte) => chars[byte % chars.length]).join('')
}

/**
 * Generate a Base64 key for the Shadowsocks 2022 protocol
 * @param method - Encryption method
 * @returns Base64-encoded key
 */
export function generateSS2022Key(method: string): string {
  // Determine key byte length based on encryption method
  let keyLength: number
  if (method === "2022-blake3-aes-128-gcm") {
    keyLength = 16 // 128-bit
  } else if (method === "2022-blake3-aes-256-gcm" || method === "2022-blake3-chacha20-poly1305") {
    keyLength = 32 // 256-bit
  } else {
    // Non-2022 protocol, return a plain password
    return generateSecureRandomString(16)
  }

  // Generate random bytes
  const array = new Uint8Array(keyLength)
  if (typeof window !== 'undefined' && window.crypto) {
    window.crypto.getRandomValues(array)
  } else {
    // Fallback
    for (let i = 0; i < keyLength; i++) {
      array[i] = Math.floor(Math.random() * 256)
    }
  }

  // Convert to Base64
  let binary = ''
  array.forEach((byte) => {
    binary += String.fromCharCode(byte)
  })
  return btoa(binary)
}
