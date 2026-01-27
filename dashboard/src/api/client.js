const API_BASE = '/api/v1';

class UltrathinkAPI {
  async request(endpoint, options = {}) {
    const url = `${API_BASE}${endpoint}`;
    const response = await fetch(url, {
      headers: {
        'Content-Type': 'application/json',
        ...options.headers,
      },
      ...options,
    });

    if (!response.ok) {
      throw new Error(`API Error: ${response.status} ${response.statusText}`);
    }

    return response.json();
  }

  // Memories
  async getMemories(params = {}) {
    const queryString = new URLSearchParams(params).toString();
    return this.request(`/memories${queryString ? `?${queryString}` : ''}`);
  }

  async getMemory(id) {
    return this.request(`/memories/${id}`);
  }

  async createMemory(data) {
    return this.request('/memories', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateMemory(id, data) {
    return this.request(`/memories/${id}`, {
      method: 'PUT',
      body: JSON.stringify(data),
    });
  }

  async deleteMemory(id) {
    return this.request(`/memories/${id}`, {
      method: 'DELETE',
    });
  }

  async searchMemories(query, options = {}) {
    return this.request('/memories/search', {
      method: 'POST',
      body: JSON.stringify({ query, ...options }),
    });
  }

  // Stats
  async getStats() {
    return this.request('/stats');
  }

  async getDomainStats(domain) {
    return this.request(`/domains/${domain}/stats`);
  }

  // Health
  async getHealth() {
    return this.request('/health');
  }

  // Domains
  async getDomains() {
    return this.request('/domains');
  }

  // Categories
  async getCategories() {
    return this.request('/categories');
  }

  // Sessions
  async getSessions() {
    return this.request('/sessions');
  }
}

export const api = new UltrathinkAPI();
export default api;
