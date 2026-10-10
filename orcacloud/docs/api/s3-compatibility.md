# T3Storage S3 Compatibility API

T3Storage provides a fully S3‑compatible API surface, enabling seamless integration with AWS CLI, SDKs, Terraform, Kubernetes, and existing S3‑based applications. This document describes supported operations, behaviors, request formats, and compatibility guarantees.

---

## 1. Compatibility Overview

T3Storage supports:

- AWS S3 REST API (major operations)
- AWS Signature Version 4 (SigV4)
- Multipart uploads
- Bucket policies
- Presigned URLs
- Versioning (optional)
- Tags, ACLs (partial)

Fully compatible with:

- AWS CLI  
- s3cmd  
- MinIO client (mc)  
- All major SDKs (Go, Python, Java, JS, Rust)

---

## 2. Endpoints

- Examples:  --endpoint-ur http(s)://0.0.0.0:9000


---

## 3. Supported Bucket Operations

| Operation | Description |
|----------|-------------|
| CreateBucket | Create a new bucket |
| DeleteBucket | Delete an empty bucket |
| ListBuckets | List all buckets |
| GetBucketLocation | Return region/zone |
| PutBucketPolicy | Apply bucket policy |
| GetBucketPolicy | Retrieve policy |
| DeleteBucketPolicy | Remove policy |
| PutBucketTagging | Add tags |
| GetBucketTagging | Retrieve tags |

---

## 4. Supported Object Operations

| Operation | Description |
|----------|-------------|
| PutObject | Upload object |
| GetObject | Download object |
| DeleteObject | Remove object |
| CopyObject | Server‑side copy |
| HeadObject | Metadata only |
| ListObjectsV2 | List objects |
| PutObjectTagging | Add tags |
| GetObjectTagging | Retrieve tags |

---

## 5. Multipart Uploads

Supported operations:

- CreateMultipartUpload  
- UploadPart  
- CompleteMultipartUpload  
- AbortMultipartUpload  
- ListMultipartUploads  

Max object size: **5 TB**

---

## 6. Error Model

T3Storage returns AWS‑style errors:

- `NoSuchBucket`
- `NoSuchKey`
- `AccessDenied`
- `InvalidPart`
- `InvalidRequest`
- `InternalError`

---

## 7. Example Usage

### Upload

aws s3 cp file.txt s3://mybucket/file.txt --endpoint-url http://gateway:9000


### Download
aws s3 cp s3://mybucket/file.txt . --endpoint-url http://gateway:9000

### List
aws s3 ls s3://mybucket --endpoint-url http://gateway:9000

---

## 8. Developer Notes

- Multipart uploads follow AWS semantics exactly  
- SigV4 validation required for all authenticated requests  
- Bucket policies override tenant policies  
- Object metadata is strongly consistent  

T3Storage is fully S3‑compatible for modern cloud workloads.



