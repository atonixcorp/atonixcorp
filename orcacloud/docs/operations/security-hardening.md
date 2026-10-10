# Security Hardening Guide

This document describes the security best practices and hardening steps required to operate T3Cloud safely in production environments.

---

## 1. Network Security

### Segmentation
- Separate VLANs for:
  - Storage traffic
  - API traffic
  - Management traffic

### Firewall Rules
- Gateways exposed publicly
- Storage nodes internal only
- Metadata cluster isolated

### TLS Everywhere
- TLS for all gateways
- Optional mTLS for internal traffic

---

## 2. Authentication & Authorization

### Access Keys
- Managed via T3Control
- Rotate every 90 days
- Encrypted at rest

### Policies
- Bucket policies (S3)
- Tenant IAM policies
- Volume access policies
- Filesystem ACLs

### Presigned URLs
- Short expiration
- Limited permissions

---

## 3. Data Security

### Encryption at Rest
- SSE‑T3 (T3Control‑managed keys)
- SSE‑KMS (tenant‑specific keys)

### Encryption in Transit
- TLS 1.2+

### Checksums
- SHA‑256 for object integrity
- Block‑level checksums for T3Block

---

## 4. Node Hardening

### OS Hardening
- Disable unused services
- Enable auditd
- Enforce SSH key‑only access

### Kernel Tuning
- Disable swap
- Enable I/O scheduler tuning
- Enable NVMe optimizations

### Filesystem Hardening
- Noexec on temp directories
- Read‑only root (optional)

---

## 5. Logging & Auditing

### Audit Logs
- Auth events
- S3 operations
- Volume operations
- Filesystem operations
- Archive transitions

### Log Retention
- 30 days hot
- 180 days warm
- 1 year cold

---

## 6. Threat Model

T3Cloud protects against:

- Unauthorized access  
- Replay attacks (SigV4)  
- Data tampering  
- Node compromise  
- Network sniffing  
- Metadata corruption  

---

## 7. Developer Notes

- Never store secrets in code  
- Always validate SigV4  
- Enforce rate limits  
- Validate all user input  
- Monitor for abnormal patterns  

---

Security hardening ensures T3Cloud remains safe, compliant, and resilient.
