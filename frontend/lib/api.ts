// API base URL - use relative paths
// Dev mode: Next.js rewrites proxy to backend
// Production: frontend and backend run on the same port
const API_BASE_URL = '';

/** API client for communicating with the sing-box backend server. */
class ApiClient {
  private baseUrl: string;

  /** @param baseUrl - Base URL for API requests (empty for same-origin). */
  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  /** Generic JSON request method with error handling. */
  private async request<T>(
    endpoint: string,
    options?: RequestInit
  ): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;

    try {
      const response = await fetch(url, {
        ...options,
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
        },
      });

      if (!response.ok) {
        const error = await response.json().catch(() => ({ error: 'Request failed' }));
        throw new Error(error.message || error.error || `HTTP ${response.status}`);
      }

      return await response.json();
    } catch (error) {
      if (error instanceof Error) {
        throw error;
      }
      throw new Error('Network request failed');
    }
  }

  // Generate WireGuard key pair
  async generateWireGuardKeys(): Promise<{ privateKey: string; publicKey: string }> {
    return this.request('/api/wireguard/keygen', {
      method: 'POST',
    });
  }
  // Generate self-signed certificate (in instance directory)
  async generateSelfSignedCert(instance: string, domain: string, validDays: number = 365): Promise<CertificateInfo> {
    return this.request('/api/singbox/certificate', {
      method: 'POST',
      body: JSON.stringify({ instance, domain, valid_days: validDays }),
    });
  }

  // Generate Reality x25519 key pair
  async generateRealityKeypair(): Promise<{ private_key: string; public_key: string }> {
    return this.request('/api/singbox/reality/keypair', {
      method: 'POST',
    });
  }

  // Derive public key from Reality private key
  async deriveRealityPublicKey(privateKey: string): Promise<{ public_key: string }> {
    return this.request('/api/singbox/reality/public-key', {
      method: 'POST',
      body: JSON.stringify({ private_key: privateKey }),
    });
  }

  // Check if domain supports TLS 1.3
  async checkTls13Support(server: string, port: number = 443): Promise<{ supported: boolean; tls_version: string; error?: string }> {
    return this.request('/api/singbox/reality/check-tls', {
      method: 'POST',
      body: JSON.stringify({ server, port }),
    });
  }

  // Get certificate info
  async getCertificateInfo(instance: string): Promise<CertificateInfo & { exists: boolean }> {
    return this.request(`/api/singbox/certificate?instance=${encodeURIComponent(instance)}`, {
      method: 'GET',
    });
  }

  // Upload certificate files
  async uploadCertificate(instance: string, certFile: File, keyFile: File): Promise<CertificateInfo> {
    const formData = new FormData();
    formData.append('instance', instance);
    formData.append('cert', certFile);
    formData.append('key', keyFile);

    const url = `${this.baseUrl}/api/singbox/certificate/upload`;
    const response = await fetch(url, {
      method: 'POST',
      body: formData,
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ error: 'Request failed' }));
      throw new Error(error.message || error.error || `HTTP ${response.status}`);
    }

    return await response.json();
  }

  // ========== Multi-instance multi-container API ==========

  // List all named configs with their container status
  async listInstances(): Promise<{ configs: InstanceInfo[] }> {
    return this.request('/api/singbox/instances', {
      method: 'GET',
    });
  }

  // Save config to named instance
  async saveInstanceConfig(instanceName: string, config: any): Promise<{ message: string; name: string; valid?: boolean; warning?: string }> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}/config`, {
      method: 'POST',
      body: JSON.stringify(config),
    });
  }

  // Validate named instance config
  async checkInstanceConfig(instanceName: string): Promise<{ valid: boolean; message: string }> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}/check`, {
      method: 'POST',
    });
  }

  // Load named instance config
  async loadInstanceConfig(instanceName: string): Promise<any> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}/config`, {
      method: 'GET',
    });
  }

  // Delete named instance
  async deleteInstance(instanceName: string): Promise<{ message: string; name: string }> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}`, {
      method: 'DELETE',
    });
  }

  // Start named instance container
  async runInstance(instanceName: string): Promise<{ message: string; name: string; containerId: string }> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}/run`, {
      method: 'POST',
    });
  }

  // Stop named instance container
  async stopInstance(instanceName: string): Promise<{ message: string; name: string }> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}/stop`, {
      method: 'POST',
    });
  }

  // Get named instance status
  async getInstanceStatus(instanceName: string): Promise<{ name: string; running: boolean; containerId: string }> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}/status`, {
      method: 'GET',
    });
  }

  // Get named instance logs
  async getInstanceLogs(instanceName: string): Promise<{ name: string; logs: string }> {
    return this.request(`/api/singbox/instances/${encodeURIComponent(instanceName)}/logs`, {
      method: 'GET',
    });
  }

  // List all containers
  async listContainers(): Promise<{ containers: ContainerStatus[] }> {
    return this.request('/api/singbox/containers', {
      method: 'GET',
    });
  }

  // ========== Prober API ==========

  // Sync subscription nodes to prober service
  async syncProberNodes(): Promise<{ message: string; nodeCount: number }> {
    return this.request('/api/prober/sync', {
      method: 'POST',
    });
  }

  // Get all probe results
  async getProberResults(): Promise<{ count: number; results: ProbeResult[] }> {
    return this.request('/api/prober/results', {
      method: 'GET',
    });
  }

  // Get prober service status
  async getProberStatus(): Promise<ProberStats> {
    return this.request('/api/prober/status', {
      method: 'GET',
    });
  }

  // Start prober service
  async startProber(): Promise<{ message: string }> {
    return this.request('/api/prober/start', {
      method: 'POST',
    });
  }

  // Stop prober service
  async stopProber(): Promise<{ message: string }> {
    return this.request('/api/prober/stop', {
      method: 'POST',
    });
  }

  // Save probe results to subscription file
  async saveProberResults(): Promise<{ message: string; count: number }> {
    return this.request('/api/prober/save', {
      method: 'POST',
    });
  }

  // ========== Proxy speed test API (starts temporary sing-box instance testing through SOCKS proxy) ==========

  /** Starts a proxy speed test via a temporary SOCKS proxy instance. */
  async startSpeedTest(): Promise<{ message: string }> {
    return this.request('/api/speedtest/start', { method: 'POST' });
  }

  /** Gets the current speed test status including per-node latency and bandwidth results. */
  async getSpeedTestStatus(): Promise<{
    running: boolean;
    total: number;
    done: number;
    current?: string;
    started_at?: string;
    results: Record<string, {
      tag: string;
      name: string;
      status: string;
      latency_ms: number;
      speed_kbps: number;
      error?: string;
      tested_at?: string;
    }>;
  }> {
    return this.request('/api/speedtest/status', { method: 'GET' });
  }

  /** Stops the running speed test. */
  async stopSpeedTest(): Promise<{ message: string }> {
    return this.request('/api/speedtest/stop', { method: 'POST' });
  }
}

/** Singleton API client instance for backend communication. */
export const apiClient = new ApiClient(API_BASE_URL);

// Type definitions

/** Result of a self-signed certificate generation request. */
export interface CertificateInfo {
  cert_path: string;
  key_path: string;
  host_cert_path: string;
  host_key_path: string;
  common_name: string;
  valid_from: string;
  valid_to: string;
  fingerprint: string;
}

/** Metadata for a named sing-box instance. */
export interface InstanceInfo {
  name: string;
  created_at: number;
  size: number;
  running: boolean;
  container_id?: string;
}

/** Status information for a running sing-box container. */
export interface ContainerStatus {
  name: string;
  container_id: string;
  state: string;
  status: string;
  created: number;
}

/** Result of probing a single proxy node for connectivity. */
export interface ProbeResult {
  tag: string;
  protocol: string;
  address: string;
  port: number;
  online: boolean;
  latency: number;
  last_probe: string;
  success_rate: number;
}

/** Statistics and configuration for the prober service. */
export interface ProberStats {
  running: boolean;
  totalNodes: number;
  onlineNodes: number;
  offlineNodes: number;
  timeoutNodes: number;
  config: {
    probeInterval: string;
    probeTimeout: string;
    maxRetries: number;
    maxConcurrent: number;
  };
}
