import * as m from "$lib/paraglide/messages.js";

function sanitizeNetworkError(message: string): string {
  if (
    /could not resolve Nexus server|could not connect to Nexus/i.test(message)
  ) {
    return message;
  }
  if (/lookup|getaddrinfo|no such host/i.test(message)) {
    return m.error_network_dns();
  }
  if (/dial tcp|connectex|connection refused|i\/o timeout/i.test(message)) {
    return m.error_network_nexus();
  }
  return message;
}

export function formatUserError(error: unknown): string {
  if (error && typeof error === "object") {
    const record = error as { message?: string; kind?: string };
    if (typeof record.message === "string" && record.message.length > 0) {
      return sanitizeNetworkError(record.message);
    }
  }
  const message = error instanceof Error ? error.message : String(error);
  if (!message || message === "[object Object]") {
    return m.error_generic();
  }
  return sanitizeNetworkError(message);
}
