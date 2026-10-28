"use client";

import { useParams } from "next/navigation";

const sectionContent: Record<string, { title: string; description: string }> = {
  dedicated: { title: "Dedicated compute", description: "Dedicated CPU plans will appear here when the Compute API publishes available inventory." },
  images: { title: "Images", description: "Operating system and custom image management will be available when image lifecycle APIs are connected." },
  snapshots: { title: "Snapshots", description: "Create and manage instance snapshots after the storage service is connected." },
  "ssh-keys": { title: "SSH keys", description: "SSH key management requires an authenticated account and tenant-scoped security API." },
  groups: { title: "Instance groups", description: "Group related instances after the compute group API is available." },
  settings: { title: "Compute settings", description: "Compute defaults and account quotas will be available when account settings APIs are connected." },
};

export default function ComputeSectionPage() {
  const { section } = useParams<{ section: string }>();
  const content = sectionContent[section] ?? { title: "Compute", description: "This compute section is not available." };
  return (
    <div className="compute-page">
      <header className="compute-page-heading"><div><p className="compute-eyebrow">COMPUTE / {section.toUpperCase()}</p><h1>{content.title}</h1><p>{content.description}</p></div></header>
      <div className="compute-state-panel"><span className="compute-empty-icon" aria-hidden="true">◈</span><h3>Service integration pending</h3><p>This section will show live account data after its API and tenant authorization are connected.</p></div>
    </div>
  );
}
