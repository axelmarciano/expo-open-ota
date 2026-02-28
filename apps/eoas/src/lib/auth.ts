export function getEoasApiKey(): string {
  const apiKey = process.env.EOAS_API_KEY;
  if (!apiKey) {
    throw new Error('EOAS_API_KEY environment variable is not set');
  }
  return apiKey;
}

export function getAuthHeaders(): Record<string, string> {
  return {
    Authorization: `Bearer ${getEoasApiKey()}`,
  };
}
