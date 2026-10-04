# T3Vault Architecture

T3Vault is the archive and cold‑storage engine of T3Cloud. It provides multi‑tier retention, low‑cost storage, background recall, and policy‑driven archival.

---

## 1. Core Design Principles

- **Low‑cost storage**  
- **Long‑term retention**  
- **Policy‑driven archival**  
- **Background recall**  
- **Multi‑tier design**  

---

## 2. System Components

### 2.1 Vault Nodes
- High‑density HDD storage  
- Cold tier management  
- Background recall workers  

### 2.2 Archive Engine
- Tiering logic  
- Retention policies  
- Recall scheduling  

### 2.3 Metadata Layer
- Archive metadata  
- Policy metadata  
- Recall state  

---

## 3. Tiering Model

Tiers:

- **Hot** — NVMe  
- **Warm** — HDD  
- **Cold** — deep archive  

Objects move between tiers based on:

- Access frequency  
- Age  
- Policy rules  

---

## 4. Recall Model

Recall steps:

1. User requests object  
2. Vault checks tier  
3. If cold → background restore  
4. Object promoted to warm tier  
5. Returned to user  

---

## 5. Developer Notes

- Tier transitions must be asynchronous  
- Policies must be versioned  
- Cold tier must be optimized for cost  
- Recall must not block other operations  

T3Vault is ideal for long‑term storage.
