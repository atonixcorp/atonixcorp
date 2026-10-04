# T3FS Filesystem API

T3FS provides a distributed filesystem with POSIX‑like semantics, NFS/SMB protocol support, and scalable metadata. This API allows developers to create filesystems, manage directories, and interact with metadata.

---

## 1. API Endpoints

Base URL:http(s)://0.0.0.0:9002/fs

Protocols:

- NFSv4  
- SMBv3  
- HTTP/JSON API  
- gRPC API  

---

## 2. Filesystem Operations

### Create Filesystem
POST /v1/filesystems
{
"name": "shared-fs",
"replication": "zone"
}


### Delete Filesystem
DELETE /v1/filesystems/{fs_id}



### List Filesystems
GET /v1/filesystems

---

## 3. Directory Operations

### Create Directory
POST /v1/fs/{fs_id}/mkdir
{
"path": "/projects/alpha"
}


### List Directory
GET /v1/fs/{fs_id}/ls?path=/projects


### Delete Directory
DELETE /v1/fs/{fs_id}/rmdir?path=/projects/alpha


---

## 4. File Operations

### Write File (HTTP)
PUT /v1/fs/{fs_id}/write?path=/projects/alpha/file.txt
<binary data>



### Read File
GET /v1/fs/{fs_id}/read?path=/projects/alpha/file.txt



### Delete File
DELETE /v1/fs/{fs_id}/unlink?path=/projects/alpha/file.txt


## 5. Metadata Operations

### Get File Metadata
GET /v1/fs/{fs_id}/stat?path=/projects/alpha/file.txt


### Set Attributes
POST /v1/fs/{fs_id}/chmod
{
"path": "/projects/alpha/file.txt",
"mode": "755"
}


## 6. Developer Notes

- Inode operations are strongly consistent  
- Directory operations are atomic  
- NFS/SMB servers support caching  
- Striping improves HPC performance  

T3FS is ideal for shared storage and HPC workloads.