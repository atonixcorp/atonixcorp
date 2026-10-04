# T3FS Architecture

T3FS is the distributed filesystem layer of T3Cloud. It provides POSIX‑like semantics, NFS/SMB protocols, distributed metadata, and scalable directory trees.

---

## 1. Core Design Principles

- **POSIX‑like behavior**  
- **Distributed metadata**  
- **Scalable directory trees**  
- **Multi‑protocol support**  
- **High throughput for HPC workloads**  

---

## 2. System Components

### 2.1 Filesystem Nodes
- Store file blocks  
- Maintain local metadata  
- Handle protocol requests  

### 2.2 Metadata Service
- Inode store  
- Directory tree  
- File layout metadata  
- Strong consistency  

### 2.3 Protocol Layer
- NFS server  
- SMB server  
- gRPC/HTTP API  

---

## 3. Filesystem Model

Each file has:

- Inode  
- Layout (striping, replication)  
- Block list  
- Attributes (size, timestamps, permissions)  

Directories are hierarchical.

---

## 4. Write Path

1. Client sends write  
2. Protocol layer forwards to FS node  
3. Node writes block  
4. Metadata updated  
5. Acknowledgement returned  

---

## 5. Read Path

Reads may be:

- Sequential  
- Parallel (striped files)  

Metadata resolves block locations.

---

## 6. Developer Notes

- Inode operations must be strongly consistent  
- Directory operations must be atomic  
- Protocol servers must support caching  
- Striping improves HPC performance  

T3FS is ideal for shared storage.
