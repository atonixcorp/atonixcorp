# HPC execution profiles

Customer job requests use AtonixCorp execution profiles rather than backend,
partition, or scheduler accounting fields:

```json
{
	"tenant_id": "tenant",
	"user_id": "user",
	"name": "simulation",
	"execution_profile": "batch",
	"script_path": "/work/run.sh",
	"cpus": 4,
	"memory_mb": 8192,
	"gpus": 1,
	"working_dir": "/work",
	"env": {"DATASET": "experiment"}
}
```

Script, resource, GPU, environment, and working-directory functionality is
retained. Operator configuration maps profiles to private scheduler settings:

```yaml
scheduler:
	execution_profiles:
		batch:
			backend: slurm
			partition: compute
			account: research
```

The backend must also be enabled in scheduler configuration. Without explicit
profiles, `batch` and `container` map to the existing Slurm and Kubernetes
adapters respectively; disabled adapters and unknown profiles fail closed.
Defining a profile does not prove scheduler readiness or grant user access.
Tenant authorization and scheduler credentials still need a real deployment.
Existing jobs remain pinned to their original backend, partition, and account;
changing a profile binding prevents refresh/cancel through the wrong target.

The HTTP submit handler rejects legacy scheduler fields and additional JSON
values. Existing callers must move to snake_case domain fields. Existing
stored jobs need an execution-profile mapping before refresh/cancel.

Job/profile translation and response filtering have unit coverage. The overall
HPC application still has pre-existing partition inventory and Kubernetes
resource parsing build failures, and repository/event dependencies are not
implemented in startup. Slurm/Kubernetes adapters are implementation
foundations, not integrated/tested infrastructure support. No additional
scheduler or cloud provider is advertised by this refactor.
