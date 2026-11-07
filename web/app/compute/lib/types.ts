export type ComputeType = {
  id: string;
  name: string;
  description: string;
};

export type ComputeRegion = {
  id: string;
  name: string;
  country: string;
  available: boolean;
  capacity_status: string;
};

export type ComputeImage = {
  id: string;
  name: string;
  status: string;
  min_disk_gb: number;
  min_memory_mb: number;
  available: boolean;
};

export type ComputePlan = {
  id: string;
  name: string;
  family: string;
  compute_type: string;
  region: string;
  vcpus: number;
  memory_mb: number;
  gpu_count?: number;
  boot_disk_gb: number;
  ephemeral_disk_gb?: number;
  swap_mb?: number;
  transfer_tb?: number;
  hourly_price: number | null;
  monthly_price: number | null;
  currency: string;
  available: boolean;
};

export type ComputeNetwork = {
  id: string;
  name: string;
  region: string;
  status: string;
  shared: boolean;
  external: boolean;
  available: boolean;
};

export type ComputeSSHKey = {
  id: string;
  name: string;
  fingerprint: string;
  key_type: string;
};

export type ComputeFirewall = {
  id: string;
  name: string;
  description?: string;
  project_id?: string;
  rules?: Array<{
    id: string;
    direction: string;
    ethertype: string;
    protocol: string;
    port_range_min?: number;
    port_range_max?: number;
    remote_ip_prefix?: string;
  }>;
};

export type ComputeCatalog = {
  provisioning_enabled: boolean;
  pricing_available: boolean;
  compute_types: ComputeType[];
  regions: ComputeRegion[];
  images: ComputeImage[];
  plans: ComputePlan[];
  networks: ComputeNetwork[];
};

export type ComputeEstimate = {
  hourly_price: number;
  monthly_price: number;
  quantity: number;
  currency: string;
  pricing_source: string;
  authoritative: boolean;
};

export type ComputeVolume = {
  name: string;
  type: "nvme_ssd";
  size_gb: number;
};

export type ComputeInstance = {
  id: string;
  provider_id: string;
  provider_status: string;
  deployment_id: string;
  name: string;
  hostname: string;
  label: string;
  region: string;
  compute_type: string;
  plan_id: string;
  image_id: string;
  authentication_method: string;
  network_id?: string;
  ssh_key_ids: string[];
  firewall_id?: string;
  public_ipv4: boolean;
  public_ipv6: boolean;
  private_networking: boolean;
  hostname_dns: boolean;
  user_data?: string;
  tags: string[];
  volumes: ComputeVolume[];
  status: string;
  provisioning: Array<{
    name: string;
    status: string;
    updated_at?: string;
    message?: string;
  }>;
  hourly_price: number | null;
  monthly_price: number | null;
  currency: string;
  node_id?: string;
  public_ip?: string;
  private_ip?: string;
  created_at: string;
  updated_at: string;
};

export type DeploymentDraft = {
  computeType: string;
  region: string;
  image: string;
  plan: string;
  network: string;
  sshKeyIds: string[];
  firewall: string;
  publicIPv4: boolean;
  publicIPv6: boolean;
  privateNetworking: boolean;
  hostnameDNS: boolean;
  userData: string;
  name: string;
  hostname: string;
  label: string;
  tags: string[];
  quantity: number;
  volumes: ComputeVolume[];
};

export const initialDraft: DeploymentDraft = {
  computeType: "",
  region: "",
  image: "",
  plan: "",
  network: "",
  sshKeyIds: [],
  firewall: "",
  publicIPv4: true,
  publicIPv6: false,
  privateNetworking: true,
  hostnameDNS: true,
  userData: "",
  name: "",
  hostname: "",
  label: "",
  tags: [],
  quantity: 1,
  volumes: [],
};
