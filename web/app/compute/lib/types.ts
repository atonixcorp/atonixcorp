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
  family: string;
  version: string;
  source: string;
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
  gpu_count: number;
  boot_disk_gb: number;
  transfer_tb: number;
  hourly_price: number;
  monthly_price: number;
  currency: string;
  available: boolean;
};

export type ComputeNetwork = {
  id: string;
  name: string;
  region: string;
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
};

export type ComputeCatalog = {
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
  hourly_price: number;
  monthly_price: number;
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
