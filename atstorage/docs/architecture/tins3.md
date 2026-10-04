# TinS3 — S3 API Gateway

TinS3 provides a fully S3‑compatible API layer for Storage Tin. It is stateless, horizontally scalable, and optimized for high throughput.

## Responsibilities

- Handle S3 requests (PUT, GET, DELETE, LIST)
- Multipart upload orchestration
- Presigned URL generation
- Authentication (SigV4)
- Bucket policies and IAM‑style permissions
- Routing requests to TinCore

## Architecture

TinS3 consists of:
- HTTP server for S3 endpoints
- Auth module for SigV4 validation
- Routing module for TinCore RPC
- Quota enforcement layer
- Request logging and metrics

## Key Features

- Full S3 compatibility
- High‑performance streaming uploads
- Multi‑tenant isolation
- Rate limiting and quotas
- Integration with TinControl for bucket management
