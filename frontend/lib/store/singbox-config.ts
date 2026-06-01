import { create } from 'zustand'

// ============= Type Definitions (sing-box format) =============
// Synced from sing-box/option/*.go

// ============= Common Types =============

export type DomainStrategy = '' | 'prefer_ipv4' | 'prefer_ipv6' | 'ipv4_only' | 'ipv6_only'

export type NetworkList = string | string[] // "tcp" | "udp" | ["tcp", "udp"]

// Duration can be string like "30s", "5m", "1h" or number (nanoseconds)
export type Duration = string | number

// ============= Log Options (options.go) =============

export interface LogOptions {
  disabled?: boolean
  level?: string // "trace" | "debug" | "info" | "warn" | "error" | "fatal" | "panic"
  output?: string
  timestamp?: boolean
}

// ============= Listen Options (inbound.go) =============

export interface ListenOptions {
  listen?: string
  listen_port?: number
  bind_interface?: string
  routing_mark?: number
  reuse_addr?: boolean
  netns?: string
  disable_tcp_keep_alive?: boolean
  tcp_keep_alive?: Duration
  tcp_keep_alive_interval?: Duration
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  udp_fragment?: boolean
  udp_timeout?: Duration
  // Deprecated: InboundOptions (use rule action instead)
  sniff?: boolean
  sniff_override_destination?: boolean
  sniff_timeout?: Duration
  domain_strategy?: DomainStrategy
  udp_disable_domain_unmapping?: boolean
  detour?: string
}

// ============= Dialer Options (outbound.go) =============

export interface DomainResolveOptions {
  server: string
  strategy?: DomainStrategy
  disable_cache?: boolean
  rewrite_ttl?: number
  client_subnet?: string
}

export interface DialerOptions {
  detour?: string
  bind_interface?: string
  inet4_bind_address?: string
  inet6_bind_address?: string
  bind_address_no_port?: boolean
  protect_path?: string
  routing_mark?: number
  reuse_addr?: boolean
  netns?: string
  connect_timeout?: Duration
  tcp_fast_open?: boolean
  tcp_multi_path?: boolean
  disable_tcp_keep_alive?: boolean
  tcp_keep_alive?: Duration
  tcp_keep_alive_interval?: Duration
  udp_fragment?: boolean
  domain_resolver?: string | DomainResolveOptions
  network_strategy?: string
  network_type?: string[]
  fallback_network_type?: string[]
  fallback_delay?: Duration
  // Deprecated
  domain_strategy?: DomainStrategy
}

// ============= Server Options (outbound.go) =============

export interface ServerOptions {
  server: string
  server_port: number
}

// ============= TLS Options (tls.go) =============

export interface InboundTLSOptions {
  enabled?: boolean
  server_name?: string
  insecure?: boolean
  alpn?: string[]
  min_version?: string
  max_version?: string
  cipher_suites?: string[]
  curve_preferences?: string[]
  certificate?: string[]
  certificate_path?: string
  client_authentication?: string // "no" | "request" | "require-any" | "verify-if-given" | "require-and-verify"
  client_certificate?: string[]
  client_certificate_path?: string[]
  client_certificate_public_key_sha256?: string[]
  key?: string[]
  key_path?: string
  kernel_tx?: boolean
  kernel_rx?: boolean
  acme?: InboundACMEOptions
  ech?: InboundECHOptions
  reality?: InboundRealityOptions
}

export interface InboundACMEOptions {
  domain?: string[]
  data_directory?: string
  default_server_name?: string
  email?: string
  provider?: string
  disable_http_challenge?: boolean
  disable_tls_alpn_challenge?: boolean
  alternative_http_port?: number
  alternative_tls_port?: number
  external_account?: ACMEExternalAccountOptions
  dns01_challenge?: ACMEDNS01ChallengeOptions
}

export interface ACMEExternalAccountOptions {
  key_id?: string
  mac_key?: string
}

export interface ACMEDNS01ChallengeOptions {
  provider?: string
}

export interface InboundECHOptions {
  enabled?: boolean
  key?: string[]
  key_path?: string
}

export interface InboundRealityOptions {
  enabled?: boolean
  handshake?: InboundRealityHandshakeOptions
  private_key?: string
  short_id?: string[]
  max_time_difference?: Duration
}

export interface InboundRealityHandshakeOptions {
  server?: string
  server_port?: number
}

export interface OutboundTLSOptions {
  enabled?: boolean
  disable_sni?: boolean
  server_name?: string
  insecure?: boolean
  alpn?: string[]
  min_version?: string
  max_version?: string
  cipher_suites?: string[]
  curve_preferences?: string[]
  certificate?: string[]
  certificate_path?: string
  certificate_public_key_sha256?: string[]
  client_certificate?: string[]
  client_certificate_path?: string
  client_key?: string[]
  client_key_path?: string
  fragment?: boolean
  fragment_fallback_delay?: Duration
  record_fragment?: boolean
  kernel_tx?: boolean
  kernel_rx?: boolean
  ech?: OutboundECHOptions
  utls?: OutboundUTLSOptions
  reality?: OutboundRealityOptions
}

export interface OutboundECHOptions {
  enabled?: boolean
  config?: string[]
  config_path?: string
  query_server_name?: string
}

export interface OutboundUTLSOptions {
  enabled?: boolean
  fingerprint?: string
}

export interface OutboundRealityOptions {
  enabled?: boolean
  public_key?: string
  short_id?: string
}

// ============= V2Ray Transport Options (v2ray_transport.go) =============

export interface V2RayTransportOptions {
  type: string // "http" | "ws" | "quic" | "grpc" | "httpupgrade"
  // HTTP
  host?: string[]
  path?: string
  method?: string
  headers?: Record<string, string | string[]>
  idle_timeout?: Duration
  ping_timeout?: Duration
  // WebSocket
  max_early_data?: number
  early_data_header_name?: string
  // gRPC
  service_name?: string
  permit_without_stream?: boolean
}

// ============= Multiplex Options (multiplex.go) =============

export interface BrutalOptions {
  enabled?: boolean
  up_mbps?: number
  down_mbps?: number
}

export interface InboundMultiplexOptions {
  enabled?: boolean
  padding?: boolean
  brutal?: BrutalOptions
}

export interface OutboundMultiplexOptions {
  enabled?: boolean
  protocol?: string // "smux" | "yamux" | "h2mux"
  max_connections?: number
  min_streams?: number
  max_streams?: number
  padding?: boolean
  brutal?: BrutalOptions
}

// ============= UDP Over TCP Options =============

export interface UDPOverTCPOptions {
  enabled?: boolean
  version?: number
}

// ============= User Types =============

export interface AuthUser {
  username: string
  password: string
}

export interface VMessUser {
  name: string
  uuid: string
  alterId?: number
}

export interface VLESSUser {
  name: string
  uuid: string
  flow?: string // "xtls-rprx-vision"
}

export interface TrojanUser {
  name: string
  password: string
}

export interface ShadowsocksUser {
  name: string
  password: string
}

export interface Hysteria2User {
  name?: string
  password?: string
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

export interface ShadowTLSUser {
  name?: string
  password: string
}

export interface AnyTLSUser {
  name?: string
  password: string
}

// ============= ShadowTLS Options =============

export interface ShadowTLSHandshake {
  server: string
  server_port: number
}

// ============= Hysteria2 Options (hysteria2.go) =============

export interface Hysteria2Obfs {
  type?: string // "salamander"
  password?: string
}

export interface Hysteria2Masquerade {
  type?: string // "file" | "proxy" | "string"
  // file
  directory?: string
  // proxy
  url?: string
  rewrite_host?: boolean
  // string
  status_code?: number
  headers?: Record<string, string | string[]>
  content?: string
}

// ============= WireGuard Types (wireguard.go) =============

export interface WireGuardPeer {
  address?: string
  port?: number
  public_key?: string
  pre_shared_key?: string
  allowed_ips?: string[]
  persistent_keepalive_interval?: number
  reserved?: number[]
}

// ============= Inbound Options =============

// Socks Inbound (simple.go)
export interface SocksInboundOptions extends ListenOptions {
  users?: AuthUser[]
  domain_resolver?: string | DomainResolveOptions
}

// HTTP/Mixed Inbound (simple.go)
export interface HTTPMixedInboundOptions extends ListenOptions {
  users?: AuthUser[]
  domain_resolver?: string | DomainResolveOptions
  set_system_proxy?: boolean
  tls?: InboundTLSOptions
}

// VMess Inbound (vmess.go)
export interface VMessInboundOptions extends ListenOptions {
  users?: VMessUser[]
  tls?: InboundTLSOptions
  multiplex?: InboundMultiplexOptions
  transport?: V2RayTransportOptions
}

// VLESS Inbound (vless.go)
export interface VLESSInboundOptions extends ListenOptions {
  users?: VLESSUser[]
  tls?: InboundTLSOptions
  multiplex?: InboundMultiplexOptions
  transport?: V2RayTransportOptions
}

// Trojan Inbound (trojan.go)
export interface TrojanInboundOptions extends ListenOptions {
  users?: TrojanUser[]
  tls?: InboundTLSOptions
  fallback?: ServerOptions
  fallback_for_alpn?: Record<string, ServerOptions>
  multiplex?: InboundMultiplexOptions
  transport?: V2RayTransportOptions
}

// Shadowsocks Inbound (shadowsocks.go)
export interface ShadowsocksInboundOptions extends ListenOptions {
  network?: NetworkList
  method: string
  password?: string
  users?: ShadowsocksUser[]
  destinations?: (ShadowsocksUser & ServerOptions)[]
  multiplex?: InboundMultiplexOptions
  managed?: boolean
}

// Hysteria2 Inbound (hysteria2.go)
export interface Hysteria2InboundOptions extends ListenOptions {
  up_mbps?: number
  down_mbps?: number
  obfs?: Hysteria2Obfs
  users?: Hysteria2User[]
  ignore_client_bandwidth?: boolean
  tls?: InboundTLSOptions
  masquerade?: string | Hysteria2Masquerade
  brutal_debug?: boolean
}

// WireGuard Endpoint (wireguard.go)
export interface WireGuardEndpointOptions extends DialerOptions {
  system?: boolean
  name?: string
  mtu?: number
  address: string[]
  private_key: string
  listen_port?: number
  peers?: WireGuardPeer[]
  udp_timeout?: Duration
  workers?: number
}

// Unified Inbound type
export interface Inbound {
  type: string // "socks" | "http" | "mixed" | "vmess" | "vless" | "trojan" | "shadowsocks" | "hysteria2" | "tuic" | "naive" | "shadowtls" | "anytls"
  tag: string
  // Common options
  listen?: string
  listen_port?: number
  // Protocol-specific options (flattened for simplicity)
  users?: (AuthUser | VMessUser | VLESSUser | TrojanUser | ShadowsocksUser | Hysteria2User | TUICUser | NaiveUser | ShadowTLSUser | AnyTLSUser)[]
  tls?: InboundTLSOptions
  multiplex?: InboundMultiplexOptions
  transport?: V2RayTransportOptions
  // Shadowsocks specific
  method?: string
  password?: string
  network?: NetworkList
  // Hysteria2 specific
  up_mbps?: number
  down_mbps?: number
  obfs?: Hysteria2Obfs
  masquerade?: string | Hysteria2Masquerade
  // WireGuard specific
  private_key?: string
  address?: string[]
  peers?: WireGuardPeer[]
  mtu?: number
  // TUIC specific
  congestion_control?: string
  zero_rtt_handshake?: boolean
  // ShadowTLS specific
  version?: number
  handshake?: ShadowTLSHandshake
  strict_mode?: boolean
  handshake_for_server_name?: Record<string, { server: string; server_port: number }>
  wildcard_sni?: string
  // Hysteria2 specific (additional)
  ignore_client_bandwidth?: boolean
  // Trojan specific
  fallback?: { server: string; server_port: number }
  // AnyTLS specific
  padding_scheme?: string[]
  // Sniff options (deprecated, use rule action)
  sniff?: boolean
  sniff_override_destination?: boolean
  domain_strategy?: DomainStrategy
}

// ============= Outbound Options =============

// Direct Outbound (direct.go)
export interface DirectOutboundOptions extends DialerOptions {
  override_address?: string
  override_port?: number
  proxy_protocol?: number
}

// Block Outbound - no additional options

// DNS Outbound - deprecated, use rule action

// SOCKS Outbound (simple.go)
export interface SOCKSOutboundOptions extends DialerOptions, ServerOptions {
  version?: string // "4" | "4a" | "5"
  username?: string
  password?: string
  network?: NetworkList
  udp_over_tcp?: boolean | UDPOverTCPOptions
}

// HTTP Outbound (simple.go)
export interface HTTPOutboundOptions extends DialerOptions, ServerOptions {
  username?: string
  password?: string
  tls?: OutboundTLSOptions
  path?: string
  headers?: Record<string, string | string[]>
}

// VMess Outbound (vmess.go)
export interface VMessOutboundOptions extends DialerOptions, ServerOptions {
  uuid: string
  security?: string // "auto" | "none" | "zero" | "aes-128-gcm" | "chacha20-poly1305"
  alter_id?: number
  global_padding?: boolean
  authenticated_length?: boolean
  network?: NetworkList
  tls?: OutboundTLSOptions
  packet_encoding?: string // "none" | "packetaddr" | "xudp"
  multiplex?: OutboundMultiplexOptions
  transport?: V2RayTransportOptions
}

// VLESS Outbound (vless.go)
export interface VLESSOutboundOptions extends DialerOptions, ServerOptions {
  uuid: string
  flow?: string // "xtls-rprx-vision"
  network?: NetworkList
  tls?: OutboundTLSOptions
  multiplex?: OutboundMultiplexOptions
  transport?: V2RayTransportOptions
  packet_encoding?: string
}

// Trojan Outbound (trojan.go)
export interface TrojanOutboundOptions extends DialerOptions, ServerOptions {
  password: string
  network?: NetworkList
  tls?: OutboundTLSOptions
  multiplex?: OutboundMultiplexOptions
  transport?: V2RayTransportOptions
}

// Shadowsocks Outbound (shadowsocks.go)
export interface ShadowsocksOutboundOptions extends DialerOptions, ServerOptions {
  method: string
  password: string
  plugin?: string
  plugin_opts?: string
  network?: NetworkList
  udp_over_tcp?: boolean | UDPOverTCPOptions
  multiplex?: OutboundMultiplexOptions
}

// Hysteria2 Outbound (hysteria2.go)
export interface Hysteria2OutboundOptions extends DialerOptions, ServerOptions {
  server_ports?: string[]
  hop_interval?: Duration
  up_mbps?: number
  down_mbps?: number
  obfs?: Hysteria2Obfs
  password?: string
  network?: NetworkList
  tls?: OutboundTLSOptions
  brutal_debug?: boolean
}

// WireGuard Outbound (wireguard.go) - Legacy
export interface WireGuardOutboundOptions extends DialerOptions {
  system_interface?: boolean
  gso?: boolean
  interface_name?: string
  local_address: string[]
  private_key: string
  peers?: WireGuardPeer[]
  server?: string
  server_port?: number
  peer_public_key?: string
  pre_shared_key?: string
  reserved?: number[]
  workers?: number
  mtu?: number
  network?: NetworkList
}

// Selector Outbound (group.go)
export interface SelectorOutboundOptions {
  outbounds: string[]
  default?: string
  interrupt_exist_connections?: boolean
}

// URLTest Outbound (group.go)
export interface URLTestOutboundOptions {
  outbounds: string[]
  url?: string
  interval?: Duration
  tolerance?: number
  idle_timeout?: Duration
  interrupt_exist_connections?: boolean
}

// Unified Outbound type
export interface Outbound {
  type: string // "direct" | "block" | "dns" | "socks" | "http" | "vmess" | "vless" | "trojan" | "shadowsocks" | "hysteria2" | "anytls" | "wireguard" | "selector" | "urltest"
  tag: string
  // Server options
  server?: string
  server_port?: number
  // Common protocol options
  uuid?: string
  password?: string
  method?: string
  flow?: string
  security?: string
  network?: NetworkList
  tls?: OutboundTLSOptions
  multiplex?: OutboundMultiplexOptions
  transport?: V2RayTransportOptions
  // Auth options (socks/http)
  username?: string
  // SOCKS specific
  version?: string // "4" | "4a" | "5"
  // HTTP specific
  path?: string
  headers?: Record<string, string | string[]>
  // Shadowsocks specific
  plugin?: string
  plugin_opts?: string
  // VMess specific
  alter_id?: number
  global_padding?: boolean
  authenticated_length?: boolean
  packet_encoding?: string
  // Dialer options
  detour?: string
  bind_interface?: string
  domain_strategy?: DomainStrategy
  domain_resolver?: string | DomainResolveOptions
  // Hysteria2 specific
  up_mbps?: number
  down_mbps?: number
  obfs?: Hysteria2Obfs
  // WireGuard specific
  local_address?: string[]
  private_key?: string
  peer_public_key?: string
  pre_shared_key?: string
  reserved?: number[]
  peers?: WireGuardPeer[]
  mtu?: number
  // AnyTLS specific
  idle_session_check_interval?: Duration
  idle_session_timeout?: Duration
  min_idle_session?: number
  // Selector/URLTest specific
  outbounds?: string[]
  default?: string
  url?: string
  interval?: Duration
  tolerance?: number
  interrupt_exist_connections?: boolean
}

// ============= DNS Options (dns.go) =============

export interface DNSServerOptions {
  tag: string
  type?: string // "udp" | "tcp" | "tls" | "https" | "quic" | "h3" | "dhcp" | "fakeip" | "local" | "hosts"
  // Common options
  server?: string
  server_port?: number
  // Legacy address field
  address?: string
  // Dialer options
  detour?: string
  domain_resolver?: string | DomainResolveOptions
  // TLS/HTTPS specific
  tls?: OutboundTLSOptions
  // HTTPS specific
  path?: string
  method?: string
  headers?: Record<string, string | string[]>
  // DHCP specific
  interface?: string
  // FakeIP specific
  inet4_range?: string
  inet6_range?: string
  // Hosts specific
  predefined?: Record<string, string[]>
  // Local specific
  prefer_go?: boolean
}

export interface DNSRuleAction {
  action?: string // "route" | "route-options" | "reject" | "predefined"
  server?: string
  strategy?: DomainStrategy
  disable_cache?: boolean
  rewrite_ttl?: number
  client_subnet?: string
  // Reject options
  method?: string // "default" | "drop"
  no_drop?: boolean
  // Predefined options
  rcode?: string
  answer?: DNSRecordOptions[]
  ns?: DNSRecordOptions[]
  extra?: DNSRecordOptions[]
}

export interface DNSRecordOptions {
  type?: string
  name?: string
  value?: string
  ttl?: number
  priority?: number
}

export interface DNSRule {
  // Match conditions
  inbound?: string[]
  ip_version?: number
  query_type?: (string | number)[]
  network?: string[]
  auth_user?: string[]
  protocol?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  source_ip_cidr?: string[]
  source_ip_is_private?: boolean
  ip_cidr?: string[]
  ip_is_private?: boolean
  source_port?: number[]
  source_port_range?: string[]
  port?: number[]
  port_range?: string[]
  clash_mode?: string
  rule_set?: string[]
  rule_set_ip_cidr_match_source?: boolean
  invert?: boolean
  // Logical rule
  type?: string // "logical"
  mode?: string // "and" | "or"
  rules?: DNSRule[]
  // Action
  action?: string
  server?: string
  strategy?: DomainStrategy
  disable_cache?: boolean
  rewrite_ttl?: number
  client_subnet?: string
  rcode?: string
}

export interface DNSClientOptions {
  strategy?: DomainStrategy
  disable_cache?: boolean
  disable_expire?: boolean
  independent_cache?: boolean
  cache_capacity?: number
  client_subnet?: string
}

export interface DNSOptions extends DNSClientOptions {
  servers?: DNSServerOptions[]
  rules?: DNSRule[]
  final?: string
  reverse_mapping?: boolean
}

// ============= Route Rule Options (rule.go, rule_action.go) =============

export interface RouteActionOptions {
  outbound?: string
  override_address?: string
  override_port?: number
  network_strategy?: string
  fallback_delay?: number
  udp_disable_domain_unmapping?: boolean
  udp_connect?: boolean
  udp_timeout?: Duration
  tls_fragment?: boolean
  tls_fragment_fallback_delay?: Duration
  tls_record_fragment?: boolean
}

export interface RejectActionOptions {
  method?: string // "default" | "drop"
  no_drop?: boolean
}

export interface SniffActionOptions {
  sniffer?: string[]
  timeout?: Duration
}

export interface ResolveActionOptions {
  server?: string
  strategy?: DomainStrategy
  disable_cache?: boolean
  rewrite_ttl?: number
  client_subnet?: string
}

export interface RouteRule {
  // Match conditions
  inbound?: string[]
  ip_version?: number
  network?: string[]
  auth_user?: string[]
  protocol?: string[]
  client?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  geosite?: string[]
  source_geoip?: string[]
  geoip?: string[]
  source_ip_cidr?: string[]
  source_ip_is_private?: boolean
  ip_cidr?: string[]
  ip_is_private?: boolean
  source_port?: number[]
  source_port_range?: string[]
  port?: number[]
  port_range?: string[]
  process_name?: string[]
  process_path?: string[]
  process_path_regex?: string[]
  package_name?: string[]
  user?: string[]
  user_id?: number[]
  clash_mode?: string
  network_type?: string[]
  network_is_expensive?: boolean
  network_is_constrained?: boolean
  wifi_ssid?: string[]
  wifi_bssid?: string[]
  rule_set?: string[]
  rule_set_ip_cidr_match_source?: boolean
  invert?: boolean
  // Logical rule
  type?: string // "logical"
  mode?: string // "and" | "or"
  rules?: RouteRule[]
  // Action (required in sing-box 1.11.0+)
  action?: string // "route" | "route-options" | "direct" | "bypass" | "reject" | "hijack-dns" | "sniff" | "resolve"
  outbound?: string
  // Route options
  override_address?: string
  override_port?: number
  udp_disable_domain_unmapping?: boolean
  udp_connect?: boolean
  udp_timeout?: Duration
  // Sniff options
  sniffer?: string[]
  timeout?: Duration
  // Resolve options
  server?: string
  strategy?: DomainStrategy
  // Reject options
  method?: string
  no_drop?: boolean
}

// ============= Rule Set Options (rule_set.go) =============

export interface LocalRuleSet {
  path: string
}

export interface RemoteRuleSet {
  url: string
  download_detour?: string
  update_interval?: Duration
}

export interface RuleSet {
  tag: string
  type: string // "inline" | "local" | "remote"
  format?: string // "source" | "binary"
  // Inline
  rules?: HeadlessRule[]
  // Local
  path?: string
  // Remote
  url?: string
  download_detour?: string
  update_interval?: Duration
}

export interface HeadlessRule {
  // Match conditions (subset of RouteRule)
  query_type?: (string | number)[]
  network?: string[]
  domain?: string[]
  domain_suffix?: string[]
  domain_keyword?: string[]
  domain_regex?: string[]
  source_ip_cidr?: string[]
  ip_cidr?: string[]
  source_port?: number[]
  source_port_range?: string[]
  port?: number[]
  port_range?: string[]
  process_name?: string[]
  process_path?: string[]
  process_path_regex?: string[]
  package_name?: string[]
  invert?: boolean
  // Logical
  type?: string
  mode?: string
  rules?: HeadlessRule[]
}

// ============= GeoIP/Geosite Options (route.go) =============

export interface GeoIPOptions {
  path?: string
  download_url?: string
  download_detour?: string
}

export interface GeositeOptions {
  path?: string
  download_url?: string
  download_detour?: string
}

// ============= Route Options (route.go) =============

export interface RouteOptions {
  geoip?: GeoIPOptions
  geosite?: GeositeOptions
  rules?: RouteRule[]
  rule_set?: RuleSet[]
  final?: string
  find_process?: boolean
  auto_detect_interface?: boolean
  override_android_vpn?: boolean
  default_interface?: string
  default_mark?: number
  default_domain_resolver?: string | DomainResolveOptions
  default_network_strategy?: string
  default_network_type?: string[]
  default_fallback_network_type?: string[]
  default_fallback_delay?: Duration
}

// ============= Experimental Options (experimental.go) =============

export interface CacheFileOptions {
  enabled?: boolean
  path?: string
  cache_id?: string
  store_fakeip?: boolean
  store_rdrc?: boolean
  rdrc_timeout?: Duration
}

export interface ClashAPIOptions {
  external_controller?: string
  external_ui?: string
  external_ui_download_url?: string
  external_ui_download_detour?: string
  secret?: string
  default_mode?: string
  access_control_allow_origin?: string[]
  access_control_allow_private_network?: boolean
}

export interface V2RayAPIOptions {
  listen?: string
  stats?: V2RayStatsServiceOptions
}

export interface V2RayStatsServiceOptions {
  enabled?: boolean
  inbounds?: string[]
  outbounds?: string[]
  users?: string[]
}

export interface DebugOptions {
  listen?: string
  gc_percent?: number
  max_stack?: number
  max_threads?: number
  panic_on_fault?: boolean
  trace_back?: string
  memory_limit?: number
  oom_killer?: boolean
}

export interface ExperimentalOptions {
  cache_file?: CacheFileOptions
  clash_api?: ClashAPIOptions
  v2ray_api?: V2RayAPIOptions
  debug?: DebugOptions
}

// ============= Root Config (options.go) =============

export interface SingBoxConfig {
  log?: LogOptions
  dns?: DNSOptions
  ntp?: NTPOptions
  certificate?: CertificateOptions
  endpoints?: Endpoint[]
  inbounds?: Inbound[]
  outbounds?: Outbound[]
  route?: RouteOptions
  services?: Service[]
  experimental?: ExperimentalOptions
}

// Additional types referenced but not fully defined
export interface NTPOptions {
  enabled?: boolean
  server?: string
  server_port?: number
  interval?: Duration
  write_to_system?: boolean
}

export interface CertificateOptions {
  certificate?: string[]
  certificate_path?: string
}

// WireGuard Endpoint (sing-box 1.11.0+)
export interface WireGuardEndpoint {
  type: "wireguard"
  tag: string
  system?: boolean
  name?: string
  mtu?: number
  address: string[]
  private_key: string
  listen_port?: number
  peers?: WireGuardPeer[]
  udp_timeout?: Duration
  workers?: number
}

export interface Endpoint {
  type: string
  tag: string
  // WireGuard specific
  system?: boolean
  name?: string
  mtu?: number
  address?: string[]
  private_key?: string
  listen_port?: number
  peers?: WireGuardPeer[]
  udp_timeout?: Duration
  workers?: number
}

export interface Service {
  type: string
  tag?: string
}

/** @deprecated Use DNSServerOptions instead. */
export type DnsServer = DNSServerOptions
/** @deprecated Use DNSRule instead. */
export type DnsRule = DNSRule
/** @deprecated Use DNSOptions instead. */
export type DnsConfig = DNSOptions
/** @deprecated Use RouteRule instead. */
export type RoutingRule = RouteRule
/** @deprecated Use RouteOptions instead. */
export type RoutingConfig = RouteOptions
/** @deprecated Use RouteOptions instead. */
export type RouteConfig = RouteOptions
/** @deprecated Use CacheFileOptions instead. */
export type CacheFile = CacheFileOptions
/** @deprecated Use ClashAPIOptions instead. */
export type ClashApi = ClashAPIOptions
/** @deprecated Use ExperimentalOptions instead. */
export type Experimental = ExperimentalOptions

// ============= Default Config (sing-box format) =============

const defaultDns: DNSOptions = {
  servers: [
    {
      tag: "default_dns",
      type: "udp",
      server: "8.8.8.8",
    },
    {
      tag: "local_dns",
      type: "udp",
      server: "8.8.8.8",
    },
    {
      tag: "remote_dns",
      type: "udp",
      server: "8.8.8.8",
      detour: "proxy_out",
    },
  ],
  rules: [
    {
      action: "route",
      server: "local_dns",
      rule_set: ["geosite-cn"],
    },
  ],
  final: "remote_dns",
  independent_cache: true,
}

const defaultConfig: SingBoxConfig = {
  log: {
    level: "warn",
    timestamp: true,
  },
  dns: defaultDns,
  inbounds: [],
  outbounds: [],
}

// ============= Store Interface =============

/** State for the URL-test balancer outbound mode. */
interface BalancerState {
  enabled: boolean
  selectedOutbounds: string[]
  strategy: string
  allOutbounds: Outbound[]
}

/** Result of a save operation returned by store actions. */
interface SaveResult {
  success: boolean
  path?: string
  error?: string
  valid?: boolean
  warning?: string
}

/** Zustand store interface for the full sing-box configuration state and actions. */
interface SingboxConfigStore {
  // Current instance
  currentInstance: string | null
  instances: InstanceInfo[]

  // Core config state
  config: SingBoxConfig

  // Balancer state (derived from outbound selection)
  balancerState: BalancerState

  // Loading/saving state
  isLoading: boolean
  isSaving: boolean
  isLoaded: boolean
  lastSavedAt: number | null
  error: string | null

  /** Sets the log level. */
  setLogLevel: (level: string) => void

  /** Replaces the entire DNS configuration. */
  setDns: (dns: DNSOptions) => void

  /** Updates an endpoint at the given index. */
  setEndpoint: (index: number, endpoint: Endpoint) => void
  /** Appends a new endpoint. */
  addEndpoint: (endpoint: Endpoint) => void
  /** Removes an endpoint at the given index. */
  removeEndpoint: (index: number) => void
  /** Removes all endpoints. */
  clearEndpoints: () => void

  /** Updates an inbound at the given index. */
  setInbound: (index: number, inbound: Inbound) => void
  /** Appends a new inbound. */
  addInbound: (inbound: Inbound) => void
  /** Removes an inbound at the given index. */
  removeInbound: (index: number) => void
  /** Removes all inbounds. */
  clearInbounds: () => void

  /** Updates an outbound at the given index. */
  setOutbound: (index: number, outbound: Outbound) => void
  /** Appends a new outbound. */
  addOutbound: (outbound: Outbound) => void
  /** Removes an outbound at the given index. */
  removeOutbound: (index: number) => void
  /** Removes all outbounds. */
  clearOutbounds: () => void
  /** Replaces the entire outbounds array. */
  setOutbounds: (outbounds: Outbound[]) => void

  /** Sets the route configuration (was renamed from setRouting). */
  setRoute: (route: RouteOptions | undefined) => void
  /** @deprecated Use setRoute instead. */
  setRouting: (routing: RouteOptions | undefined) => void

  /** Sets the balancer state (null resets to defaults). */
  setBalancerState: (state: BalancerState | null) => void

  /** Sets the currently selected instance name. */
  setCurrentInstance: (instance: string | null) => void
  /** Replaces the list of known instances. */
  setInstances: (instances: InstanceInfo[]) => void
  /** Fetches the list of instances from the backend. */
  loadInstances: () => Promise<void>
  /** Loads a named instance config from the backend. */
  loadInstanceConfig: (instance: string) => Promise<boolean>
  /** Saves the current config to the selected instance. */
  saveInstanceConfig: () => Promise<SaveResult>
  /** Creates a new named instance with the current config. */
  createInstance: (name: string) => Promise<boolean>
  /** Deletes a named instance from the backend. */
  deleteInstance: (name: string) => Promise<boolean>

  /** Loads the config from the legacy server endpoint. */
  loadFromServer: () => Promise<boolean>
  /** Saves the config to the legacy server endpoint. */
  saveToServer: () => Promise<SaveResult>

  /** Merges a partial config into the current state. */
  loadConfig: (config: Partial<SingBoxConfig>) => void
  /** Resets the config to defaults and clears the loaded state. */
  resetConfig: () => void

  /** Computes the full sing-box config with balancer logic, WireGuard migration, and DNS defaults. */
  getFullConfig: () => SingBoxConfig
}

// ============= Store Implementation =============

export const useSingboxConfigStore = create<SingboxConfigStore>((set, get) => ({
  currentInstance: null,
  instances: [],
  config: { ...defaultConfig },
  balancerState: {
    enabled: false,
    selectedOutbounds: [],
    strategy: '50',
    allOutbounds: [],
  },
  isLoading: false,
  isSaving: false,
  isLoaded: false,
  lastSavedAt: null,
  error: null,

  // ============= Log Actions =============

  setLogLevel: (level) => set((state) => ({
    config: {
      ...state.config,
      log: { ...state.config.log, level },
    },
  })),

  // ============= DNS Actions =============

  setDns: (dns) => set((state) => ({
    config: {
      ...state.config,
      dns,
    },
  })),

  // ============= Endpoint Actions (sing-box 1.11.0+) =============

  setEndpoint: (index, endpoint) => set((state) => {
    const endpoints = [...(state.config.endpoints || [])]
    if (index >= endpoints.length) {
      endpoints.push(endpoint)
    } else {
      endpoints[index] = endpoint
    }
    return {
      config: {
        ...state.config,
        endpoints,
      },
    }
  }),

  addEndpoint: (endpoint) => set((state) => ({
    config: {
      ...state.config,
      endpoints: [...(state.config.endpoints || []), endpoint],
    },
  })),

  removeEndpoint: (index) => set((state) => ({
    config: {
      ...state.config,
      endpoints: (state.config.endpoints || []).filter((_, i) => i !== index),
    },
  })),

  clearEndpoints: () => set((state) => ({
    config: {
      ...state.config,
      endpoints: [],
    },
  })),

  // ============= Inbound Actions =============

  setInbound: (index, inbound) => set((state) => {
    const inbounds = [...(state.config.inbounds || [])]
    if (index >= inbounds.length) {
      inbounds.push(inbound)
    } else {
      inbounds[index] = inbound
    }
    return {
      config: {
        ...state.config,
        inbounds,
      },
    }
  }),

  addInbound: (inbound) => set((state) => ({
    config: {
      ...state.config,
      inbounds: [...(state.config.inbounds || []), inbound],
    },
  })),

  removeInbound: (index) => set((state) => ({
    config: {
      ...state.config,
      inbounds: (state.config.inbounds || []).filter((_, i) => i !== index),
    },
  })),

  clearInbounds: () => set((state) => ({
    config: {
      ...state.config,
      inbounds: [],
    },
  })),

  // ============= Outbound Actions =============

  setOutbound: (index, outbound) => set((state) => {
    const outbounds = [...(state.config.outbounds || [])]
    if (index >= outbounds.length) {
      outbounds.push(outbound)
    } else {
      outbounds[index] = outbound
    }
    return {
      config: {
        ...state.config,
        outbounds,
      },
    }
  }),

  addOutbound: (outbound) => set((state) => ({
    config: {
      ...state.config,
      outbounds: [...(state.config.outbounds || []), outbound],
    },
  })),

  removeOutbound: (index) => set((state) => ({
    config: {
      ...state.config,
      outbounds: (state.config.outbounds || []).filter((_, i) => i !== index),
    },
  })),

  clearOutbounds: () => set((state) => ({
    config: {
      ...state.config,
      outbounds: [],
    },
  })),

  setOutbounds: (outbounds) => set((state) => ({
    config: {
      ...state.config,
      outbounds,
    },
  })),

  // ============= Route Actions =============

  setRoute: (route) => set((state) => ({
    config: {
      ...state.config,
      route,
    },
  })),

  // Legacy alias for backward compatibility
  setRouting: (routing) => set((state) => ({
    config: {
      ...state.config,
      route: routing,
    },
  })),

  // ============= Balancer Actions =============

  setBalancerState: (balancerState) => set({
    balancerState: balancerState || {
      enabled: false,
      selectedOutbounds: [],
      strategy: '50',
      allOutbounds: [],
    },
  }),

  // ============= Instance Actions =============

  setCurrentInstance: (instance) => set({ currentInstance: instance }),

  setInstances: (instances) => set({ instances }),

  loadInstances: async () => {
    try {
      const response = await fetch("/api/singbox/instances")
      if (response.ok) {
        const data = await response.json()
        set({ instances: data.configs || [] })
      }
    } catch (error) {
      console.error("Failed to load instances:", error)
    }
  },

  loadInstanceConfig: async (instance: string) => {
    set({ isLoading: true, error: null })
    try {
      const response = await fetch(`/api/singbox/instances/${encodeURIComponent(instance)}/config`)
      if (response.ok) {
        const configData = await response.json()
        set((state) => ({
          currentInstance: instance,
          config: {
            ...state.config,
            log: configData.log || state.config.log,
            dns: configData.dns || state.config.dns,
            endpoints: configData.endpoints || [],
            inbounds: configData.inbounds || [],
            outbounds: configData.outbounds || [],
            route: configData.route,
            experimental: configData.experimental,
          },
          isLoaded: true,
          isLoading: false,
        }))
        return true
      } else {
        set({ isLoading: false })
        return false
      }
    } catch (error) {
      set({
        isLoading: false,
        error: error instanceof Error ? error.message : "Failed to load config"
      })
      return false
    }
  },

  saveInstanceConfig: async () => {
    const { currentInstance } = get()
    if (!currentInstance) {
      return { success: false, error: "No instance selected" }
    }
    set({ isSaving: true, error: null })
    try {
      const fullConfig = get().getFullConfig()
      const response = await fetch(`/api/singbox/instances/${encodeURIComponent(currentInstance)}/config`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(fullConfig, null, 2),
      })

      const data = await response.json()

      if (response.ok) {
        set({
          isSaving: false,
          lastSavedAt: Date.now()
        })
        return { success: true, valid: data.valid, warning: data.warning, error: data.message }
      } else {
        set({
          isSaving: false,
          error: data.message
        })
        return { success: false, error: data.message }
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : "Failed to save config"
      set({ isSaving: false, error: errorMsg })
      return { success: false, error: errorMsg }
    }
  },

  createInstance: async (name: string) => {
    try {
      const fullConfig = get().getFullConfig()
      const response = await fetch(`/api/singbox/instances/${encodeURIComponent(name)}/config`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(fullConfig, null, 2),
      })
      if (response.ok) {
        await get().loadInstances()
        set({ currentInstance: name })
        return true
      }
      return false
    } catch (error) {
      console.error("Failed to create instance:", error)
      return false
    }
  },

  deleteInstance: async (name: string) => {
    try {
      const response = await fetch(`/api/singbox/instances/${encodeURIComponent(name)}`, {
        method: "DELETE",
      })
      if (response.ok) {
        const { currentInstance } = get()
        if (currentInstance === name) {
          set({ currentInstance: null })
          get().resetConfig()
        }
        await get().loadInstances()
        return true
      }
      return false
    } catch (error) {
      console.error("Failed to delete instance:", error)
      return false
    }
  },

  // ============= Server Sync Actions =============

  loadFromServer: async () => {
    set({ isLoading: true, error: null })
    try {
      const response = await fetch("/api/singbox/config")
      if (response.ok) {
        const configData = await response.json()
        set((state) => ({
          config: {
            ...state.config,
            log: configData.log || state.config.log,
            dns: configData.dns || state.config.dns,
            endpoints: configData.endpoints || [],
            inbounds: configData.inbounds || [],
            outbounds: configData.outbounds || [],
            route: configData.route,
            experimental: configData.experimental,
          },
          isLoaded: true,
          isLoading: false,
        }))
        return true
      } else {
        set({ isLoading: false })
        return false
      }
    } catch (error) {
      set({
        isLoading: false,
        error: error instanceof Error ? error.message : "Failed to load config"
      })
      return false
    }
  },

  saveToServer: async () => {
    set({ isSaving: true, error: null })
    try {
      const fullConfig = get().getFullConfig()
      const response = await fetch("/api/singbox/config", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(fullConfig, null, 2),
      })

      if (response.ok) {
        const data = await response.json()
        set({
          isSaving: false,
          lastSavedAt: Date.now()
        })
        return { success: true, path: data.path }
      } else {
        const errorData = await response.json()
        set({
          isSaving: false,
          error: errorData.message
        })
        return { success: false, error: errorData.message }
      }
    } catch (error) {
      const errorMsg = error instanceof Error ? error.message : "Failed to save config"
      set({ isSaving: false, error: errorMsg })
      return { success: false, error: errorMsg }
    }
  },

  // ============= Local Config Actions =============

  loadConfig: (config) => set((state) => ({
    config: {
      ...state.config,
      ...config,
      log: config.log || state.config.log,
      dns: config.dns || state.config.dns,
      endpoints: config.endpoints || state.config.endpoints,
      inbounds: config.inbounds || state.config.inbounds,
      outbounds: config.outbounds || state.config.outbounds,
      route: config.route,
      experimental: config.experimental,
    },
    isLoaded: true,
  })),

  resetConfig: () => set({
    config: { ...defaultConfig },
    balancerState: {
      enabled: false,
      selectedOutbounds: [],
      strategy: '50',
      allOutbounds: [],
    },
    isLoaded: false,
    lastSavedAt: null,
    error: null,
  }),

  // ============= Computed =============

  getFullConfig: () => {
    const state = get()
    const { config, balancerState } = state

    // Build outbounds array
    let outbounds: Outbound[] = []

    if (balancerState.enabled && balancerState.allOutbounds.length > 0) {
      // Balancer mode: add all participating outbounds + keep direct/block from config
      const seenTags = new Set<string>()
      for (const outbound of balancerState.allOutbounds) {
        if (outbound.tag && !seenTags.has(outbound.tag)) {
          seenTags.add(outbound.tag)
          outbounds.push(outbound)
        }
      }
      // Preserve non-proxy outbounds (direct, block, dns) from config
      for (const outbound of (config.outbounds || [])) {
        if (outbound.tag && !seenTags.has(outbound.tag) &&
            (outbound.type === "direct" || outbound.type === "block" || outbound.type === "dns")) {
          seenTags.add(outbound.tag)
          outbounds.push(outbound)
        }
      }
    } else {
      // Normal mode (deduplicated by tag)
      const seenTags = new Set<string>()
      for (const outbound of (config.outbounds || [])) {
        if (outbound.tag && !seenTags.has(outbound.tag)) {
          seenTags.add(outbound.tag)
          outbounds.push(outbound)
        } else if (!outbound.tag) {
          outbounds.push(outbound)
        }
      }
    }

    // sing-box 1.11.0 deprecated wireguard as an outbound, removed entirely in 1.13.0.
    // Various parts of the UI (WireGuard / WARP tab, subscription parsing, etc.) still
    // write via setOutbound(0, ...). We migrate them to endpoints[] at serialization
    // to avoid fatal errors like `outbounds[0].address: json: unknown field "address"`.
    // Rules:
    //   1. Extract all entries with type==="wireguard" from outbounds
    //   2. Merge into config.endpoints by tag (same tag → replace, new tag → append)
    //   3. Keep only non-wireguard entries in outbounds
    const wgFromOutbounds = outbounds.filter((o) => o.type === "wireguard")
    outbounds = outbounds.filter((o) => o.type !== "wireguard")

    const mergedEndpoints: Endpoint[] = [...(config.endpoints || [])]
    for (const wg of wgFromOutbounds) {
      const idx = wg.tag
        ? mergedEndpoints.findIndex((ep) => ep.tag && ep.tag === wg.tag)
        : -1
      if (idx >= 0) {
        mergedEndpoints[idx] = wg as unknown as Endpoint
      } else {
        mergedEndpoints.push(wg as unknown as Endpoint)
      }
    }

    // Ensure direct and block outbounds exist if route is configured
    if (config.route && config.route.rules && config.route.rules.length > 0) {
      const hasDirectTag = outbounds.some((o) => o.tag === "direct")
      const hasBlockTag = outbounds.some((o) => o.tag === "block")

      if (!hasDirectTag) {
        outbounds.push({ type: "direct", tag: "direct" })
      }
      if (!hasBlockTag) {
        outbounds.push({ type: "block", tag: "block" })
      }
    }

    // Build full config
    const fullConfig: SingBoxConfig = {
      log: config.log,
      dns: config.dns,
      // Filter out invalid endpoints (e.g., WireGuard without private_key)
      endpoints: mergedEndpoints.filter((ep) => {
        if (ep.type === "wireguard") {
          return ep.private_key && ep.private_key.length > 0
        }
        return true
      }),
      inbounds: config.inbounds,
      outbounds,
    }

    // Remove endpoints array if empty
    if (fullConfig.endpoints && fullConfig.endpoints.length === 0) {
      delete fullConfig.endpoints
    }

    // Check if there's a proxy outbound or an outbound-role endpoint.
    // endpoint and outbound share the tag namespace; route.final can reference both, but a WG endpoint
    // in config.endpoints represents an "inbound VPN server" role (written by WireguardForm, tag is
    // usually wireguard-ep, simply waiting for peers to connect). Such endpoints should not be counted
    // as proxy outbounds. Only WG endpoints migrated from outbounds[] (wgFromOutbounds) are proxy outbounds.
    //
    // Previously there was no distinction here; a pure WG inbound scenario would cause hasProxyOutbound
    // to incorrectly be true, skipping the proxy_out fallback, while the later branch still hardcoded
    // route.final: "proxy_out" and dns.servers[0].detour: "proxy_out", causing sing-box to fail with
    //   "default outbound not found: proxy_out" on startup.
    const wgOutboundTags = new Set(
      wgFromOutbounds.map((o) => o.tag).filter((t): t is string => typeof t === "string" && t.length > 0)
    )
    const hasProxyOutbound = outbounds.some((o) =>
      o.type !== "direct" && o.type !== "block" && o.type !== "dns"
    ) || (fullConfig.endpoints || []).some((ep) => {
      if (ep.type === "direct" || ep.type === "block" || ep.type === "dns") return false
      // WG endpoint: only those migrated from outbounds count as proxy outbounds
      if (ep.type === "wireguard") return !!ep.tag && wgOutboundTags.has(ep.tag)
      // Other endpoint types (future extensions) default to proxy outbound handling
      return true
    })
    const hasProxyOutTag = outbounds.some((o) => o.tag === "proxy_out") ||
      (fullConfig.endpoints || []).some((ep) => ep.tag === "proxy_out")
    // If no proxy outbound/endpoint exists, add a direct outbound with tag "proxy_out"
    // This ensures route.final: "proxy_out" always works
    if (!hasProxyOutbound && !hasProxyOutTag) {
      outbounds.push({ type: "direct", tag: "proxy_out" })
      fullConfig.outbounds = outbounds
    }

    // Auto-fill domain_resolver for DNS servers that use domain addresses (https/tls/quic/h3)
    if (fullConfig.dns?.servers) {
      const domainTypes = new Set(["https", "tls", "quic", "h3"])
      // Find a suitable resolver: udp/tcp server with IP address (not domain)
      const ipResolver = fullConfig.dns.servers.find(
        (s) => (s.type === "udp" || s.type === "tcp") && s.server && /^[\d.:]+$/.test(s.server)
      )
      if (ipResolver) {
        fullConfig.dns = {
          ...fullConfig.dns,
          servers: fullConfig.dns.servers.map((server) => {
            if (
              domainTypes.has(server.type || "") &&
              server.server &&
              !/^[\d.:]+$/.test(server.server) &&
              !server.domain_resolver
            ) {
              return { ...server, domain_resolver: { server: ipResolver.tag } }
            }
            return server
          }),
        }
      }
    }

    // If no proxy outbound, override DNS and route with minimal direct config
    // (split-routing DNS and geo rule_sets are pointless when everything goes direct)
    if (!hasProxyOutbound) {
      fullConfig.dns = {
        servers: [{ tag: "local_dns", type: "udp", server: "8.8.8.8" }],
        final: "local_dns",
        independent_cache: true,
      }
      fullConfig.route = {
        rules: [],
        final: "proxy_out",
        default_domain_resolver: "local_dns",
      }
      return fullConfig
    }

    // Proxy outbound (non-balancer): override DNS and route with minimal global-proxy config
    //
    // DNS design:
    //   - remote_dns goes through proxy_out, handles user traffic DNS, prevents pollution/leakage
    //   - local_resolver has no detour; sing-box 1.13 defaults to direct
    //     (explicit detour to an empty direct outbound is rejected by 1.13 with:
    //      "detour to an empty direct outbound makes no sense")
    //   - default_domain_resolver points to local_resolver, so route rule domain resolution
    //     can complete before the proxy tunnel is ready (WG peer domain needs resolution before handshake)
    //   - dns.final is still remote_dns: user traffic domains resolve through WARP during normal use
    if (!balancerState.enabled) {
      fullConfig.dns = {
        servers: [
          { tag: "remote_dns", type: "udp", server: "8.8.8.8", detour: "proxy_out" },
          { tag: "local_resolver", type: "udp", server: "1.1.1.1" },
        ],
        final: "remote_dns",
        independent_cache: true,
      }
      fullConfig.route = {
        rules: [],
        final: "proxy_out",
        default_domain_resolver: "local_resolver",
      }
      return fullConfig
    }

    // Balancer mode: build urltest outbound and route from existing config.
    // endpoints[] tags are included because WireGuard/WARP have been migrated from outbounds to endpoints
    // (see the migration logic at the start of getFullConfig); both share the tag namespace;
    // if we only took outbounds here, route rules targeting WG/WARP tags would be incorrectly filtered out.
    const validOutboundTags = new Set<string>([
      ...outbounds.map((o) => o.tag),
      ...mergedEndpoints.map((ep) => ep.tag).filter((t): t is string => typeof t === "string" && t.length > 0),
    ])

    /** Generates remote RuleSet definitions for all rule_set tags referenced in route and DNS rules. */
    const generateRuleSetDefinitions = (routeRules: RouteRule[], dnsRules?: DNSRule[]): RuleSet[] => {
      const usedRuleSets = new Set<string>()
      for (const rule of routeRules) {
        if (rule.rule_set) {
          for (const rs of rule.rule_set) {
            usedRuleSets.add(rs)
          }
        }
      }
      // Also collect from DNS rules
      if (dnsRules) {
        for (const rule of dnsRules) {
          if (rule.rule_set) {
            for (const rs of rule.rule_set) {
              usedRuleSets.add(rs)
            }
          }
        }
      }

      const ruleSetDefinitions: RuleSet[] = []
      const ruleSetUrls: Record<string, string> = {
        "geosite-cn": "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
        "geoip-cn": "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
        "geosite-category-ads-all": "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-category-ads-all.srs",
        "geosite-geolocation-!cn": "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-geolocation-!cn.srs",
        "geoip-private": "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-private.srs",
        "geosite-gfw": "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-gfw.srs",
      }

      for (const tag of usedRuleSets) {
        if (ruleSetUrls[tag]) {
          ruleSetDefinitions.push({
            tag,
            type: "remote",
            format: "binary",
            url: ruleSetUrls[tag],
          })
        }
      }

      return ruleSetDefinitions
    }

    // sing-box route config
    if (balancerState.enabled && balancerState.selectedOutbounds.length >= 2) {
      // Balancer mode: use urltest outbound
      const tolerance = parseInt(balancerState.strategy) || 50
      const urltestOutbound: Outbound = {
        type: "urltest",
        tag: "proxy_out",
        outbounds: balancerState.selectedOutbounds,
        url: "https://www.gstatic.com/generate_204",
        interval: "3m",
        tolerance,
      }
      outbounds.push(urltestOutbound)

      if (config.route) {
        // Filter rules: remove rules referencing non-existent outbounds, remove catch-all rules (handled by final)
        const existingRules = (config.route.rules || []).filter((r) => {
          if (r.outbound && !validOutboundTags.has(r.outbound)) return false
          // Filter out catch-all rules without specific match conditions (non direct/block)
          if (r.outbound && r.outbound !== "direct" && r.outbound !== "block") {
            const hasSpecificMatch = r.port || r.protocol || r.inbound ||
              r.rule_set || r.ip_cidr || r.domain || r.domain_suffix || r.domain_keyword ||
              r.ip_is_private || r.clash_mode || r.action
            if (!hasSpecificMatch) return false
          }
          return true
        })
        const ruleSetDefs = generateRuleSetDefinitions(existingRules, fullConfig.dns?.rules)
        fullConfig.route = {
          ...config.route,
          rule_set: ruleSetDefs.length > 0 ? ruleSetDefs : undefined,
          rules: existingRules,
          final: "proxy_out",
        }
      } else {
        fullConfig.route = {
          rules: [],
          final: "proxy_out",
        }
      }
    } else if (config.route) {
      // No balancer, use existing route with final
      const filteredRules = (config.route.rules || []).filter((r) => {
        if (r.outbound && !validOutboundTags.has(r.outbound)) return false
        return true
      })
      if (filteredRules.length > 0 || config.route.final) {
        const ruleSetDefs = generateRuleSetDefinitions(filteredRules, fullConfig.dns?.rules)
        fullConfig.route = {
          ...config.route,
          rule_set: ruleSetDefs.length > 0 ? ruleSetDefs : undefined,
          rules: filteredRules,
          final: config.route.final || "proxy_out",
        }
      }
    } else if (outbounds.length > 0) {
      // Create default route with final pointing to proxy_out
      // Note: action is required in sing-box 1.11.0+
      fullConfig.route = {
        rule_set: [
          {
            tag: "geosite-cn",
            type: "remote",
            format: "binary",
            url: "https://raw.githubusercontent.com/SagerNet/sing-geosite/rule-set/geosite-cn.srs",
          },
          {
            tag: "geoip-cn",
            type: "remote",
            format: "binary",
            url: "https://raw.githubusercontent.com/SagerNet/sing-geoip/rule-set/geoip-cn.srs",
          },
        ],
        rules: [
          {
            action: "route",
            rule_set: ["geosite-cn"],
            outbound: "direct",
          },
          {
            action: "route",
            rule_set: ["geoip-cn"],
            outbound: "direct",
          },
          {
            action: "route",
            ip_is_private: true,
            outbound: "direct",
          },
        ],
        final: "proxy_out",
        default_domain_resolver: "local_dns",
      }
    }

    // Ensure route.default_domain_resolver is always set (required by sing-box 1.12.0+)
    if (fullConfig.route && !fullConfig.route.default_domain_resolver && fullConfig.dns?.servers) {
      const ipResolver = fullConfig.dns.servers.find(
        (s) => (s.type === "udp" || s.type === "tcp") && s.server && /^[\d.:]+$/.test(s.server)
      )
      if (ipResolver) {
        fullConfig.route.default_domain_resolver = ipResolver.tag
      }
    }

    return fullConfig
  },
}))

/** @deprecated Use useSingboxConfigStore instead. */
export const useXrayConfigStore = useSingboxConfigStore

/** Metadata for a named sing-box instance on the backend. */
export interface InstanceInfo {
  name: string
  created_at: number
  size: number
  running: boolean
  container_id?: string
}
