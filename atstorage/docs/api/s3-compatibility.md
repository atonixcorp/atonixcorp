# S3 Compatibility — TinS3 API Layer

TinS3 provides a fully S3‑compatible API surface for Storage Tin. This document describes the supported operations, behaviors, and compatibility guarantees.

---

## 1. Compatibility Goals

TinS3 aims to be compatible with:

- AWS S3 REST API
- AWS Signature Version 4 (SigV4)
- AWS S3 bucket policies
- AWS S3 ACLs (optional)
- AWS S3 multipart upload semantics

TinS3 is designed to work seamlessly with:

- AWS CLI
- s3cmd
- MinIO client (mc)
- SDKs for Go, Python, JavaScript, Java, Rust

---

## 2. Supported Operations

### Bucket Operations
- `CreateBucket`
- `DeleteBucket`
- `ListBuckets`
- `GetBucketLocation`
- `PutBucketPolicy`
- `GetBucketPolicy`
- `DeleteBucketPolicy`
- `PutBucketTagging`
- `GetBucketTagging`

### Object Operations
- `PutObject`
- `GetObject`
- `DeleteObject`
- `CopyObject`
- `HeadObject`
- `ListObjectsV2`
- `PutObjectTagging`
- `GetObjectTagging`

### Multipart Uploads
- `CreateMultipartUpload`
- `UploadPart`
- `CompleteMultipartUpload`
- `AbortMultipartUpload`
- `ListMultipartUploads`

Multipart uploads support objects up to **5 TB**.

---

## 3. Unsupported or Partially Supported Features

### Not Supported (Initial Release)
- S3 Select
- Object Lambda
- Intelligent Tiering
- Glacier‑style deep archive

### Partially Supported
- ACLs (TinControl policies preferred)
- Versioning (enabled per bucket)

---

## 4. Endpoint Structure

TinS3 exposes S3 endpoints via:

