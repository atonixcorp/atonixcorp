import { proxyAgentRequest } from "../../../../lib/agent-api";

export function GET(): Promise<Response> {
  return proxyAgentRequest("/v1/agent/registry");
}
