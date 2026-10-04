# T3Vault Archive API

T3Vault provides multi‑tier archival, policy‑based retention, and background recall for long‑term storage. This API allows developers to archive, retrieve, and manage cold‑tier data.

---

## 1. API Endpoints

Base URL:

http(s)://<gateway-host>:9003/vault


## 2. Archive Operations

### Archive Object
POST /v1/archive
{
"bucket": "logs",
"key": "2024/01/system.log",
"policy": "cold-180-days"
}

### List Archived Objects
GET /v1/archive?bucket=logs

---

## 3. Recall Operations

### Request Recall
POST /v1/recall
{
"bucket": "logs",
"key": "2024/01/system.log"
}


### Check Recall Status
GET /v1/recall/status?bucket=logs&key=2024/01/system.log


## 4. Policy Management

### Create Policy
POST /v1/policies
{
"name": "cold-180-days",
"tier": "cold",
"retention_days": 180
}


### List Policies
GET /v1/policies


### Delete Policy
DELETE /v1/policies/{policy_id}

## 5. Developer Notes

- Tier transitions are asynchronous  
- Cold tier optimized for cost  
- Recall does not block other operations  
- Policies must be versioned  

T3Vault is ideal for compliance, backups, and long‑term retention.