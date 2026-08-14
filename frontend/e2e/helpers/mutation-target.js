export const assertDisposableMutationTarget = (baseURL) => {
  const target = new URL(baseURL);
  const normalDevelopmentHosts = new Set(["localhost", "127.0.0.1", "::1", "host.docker.internal"]);
  if (normalDevelopmentHosts.has(target.hostname) && target.port === "8080") {
    throw new Error("Mutation E2E must not target the normal development origin on port 8080");
  }
};
