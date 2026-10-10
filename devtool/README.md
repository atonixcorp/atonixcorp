# atcloud DevTool
## Unified Developer Experience • Cloud SDK • Runtime Control • DevOps Automation

The **atcloud DevTool** is the central command-line interface for all AtonixCorp development workflows.  
It provides a unified, consistent, and powerful developer experience across:

- AtonixCorp Cloud
- OrcaCloud Compute
- AtonixCorp Container Runtime
- HPC & AI Infrastructure
- DevOps pipelines and automation
- Local development environments
- Sandbox environments

This tool is the single entrypoint for developers, engineers, operators, and automation systems working within the AtonixCorp ecosystem.

---

##  Key Capabilities

### **Cloud SDK**
- `atcloud login` for authentication  
- Context switching (`dev`, `staging`, `prod`)  
- Resource inspection  
- Deployment helpers  

### **Runtime Integration**
- Deploy workloads  
- View logs, metrics, status  
- Manage lifecycle operations  

### **DevOps Automation**
- CI/CD pipeline helpers  
- Infrastructure provisioning  
- Sandbox orchestration  

### **HPC & AI Tools**
- Submit HPC jobs  
- Monitor distributed compute  
- Manage GPU/CPU workloads  

### **Project Scaffolding**
- Create new projects  
- Generate templates  
- Initialize cloud-ready structures  

### **Tooling & Downloads**
- Install/update SDKs  
- Manage versions  
- Download dependencies  

---

##  Project Structure

atonixcorp/devtool/
├── cmd/
│   └── atcloud/             # Main CLI entrypoint
├── internal/
│   ├── cloud/               # Cloud SDK, auth, contexts
│   ├── runtime/             # Container runtime integration
│   ├── hpc/                 # HPC & AI helpers
│   ├── tools/               # Installers, updaters, downloads
│   ├── project/             # Scaffolding, templates
│   ├── config/              # Global dev config
│   └── devops/              # DevOps automation tools
├── pkg/
│   ├── cli/                 # Command framework
│   ├── api/                 # Shared API clients
│   └── utils/               # Logging, errors, helpers
├── docs/
│   └── devtool.md           # Full documentation
└── README.md                # Project overview

Code

---

##  Getting Started

### Run DevTool
```bash
atcloud help
Example Commands
bash
atcloud login
atcloud runtime deploy ./app
atcloud devops pipeline run
atcloud project init --type api --name my-service
atcloud tools update all