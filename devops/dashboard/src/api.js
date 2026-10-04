const RUNNER_API = (import.meta.env.VITE_RUNNER_API_URL || "http://localhost:8086").replace(/\/+$/, "");

async function request(path, options = {}) {
  const response = await fetch(`${RUNNER_API}${path}`, {
    ...options,
    headers: {
      Accept: "application/json",
      ...(options.body ? { "Content-Type": "application/json" } : {}),
      ...options.headers,
    },
  });

  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Runner API returned ${response.status}`);
  }
  return response.json();
}

export async function getRunnerState() {
  const [pipelineData, jobData] = await Promise.all([
    request("/pipelines"),
    request("/jobs"),
  ]);

  return {
    pipelines: Array.isArray(pipelineData.pipelines) ? pipelineData.pipelines : [],
    summary: jobData.summary || pipelineData.summary || {},
    jobs: Array.isArray(jobData.jobs) ? jobData.jobs : [],
  };
}

export async function getAgentRegistry() {
  return request("/agent/registry");
}

export async function requestAgentPrediction(payload) {
  return request("/agent/predict", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function startPipeline({ project, ref, workflow }) {
  return request("/pipelines", {
    method: "POST",
    body: JSON.stringify({ project, ref, trigger: "manual", workflow }),
  });
}

export async function getJobLogs(endpoint) {
  const response = await fetch(`${RUNNER_API}${endpoint}`, {
    headers: { Accept: "text/plain" },
  });
  if (!response.ok) {
    const message = await response.text();
    throw new Error(message || `Runner API returned ${response.status}`);
  }
  return response.text();
}
