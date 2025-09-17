import { useQuery } from '@tanstack/react-query';
import axios from 'axios';

const API_BASE = resolveBackendBase();

function resolveBackendBase() {
  const explicit = (import.meta.env.VITE_BACKEND_URL ?? '').trim();
  if (explicit) {
    if (/^https?:\/\//i.test(explicit)) {
      return stripTrailingSlash(explicit);
    }

    if (typeof window !== 'undefined') {
      try {
        const resolved = new URL(explicit, window.location.href);
        return stripTrailingSlash(resolved.toString());
      } catch {
        return stripTrailingSlash(explicit);
      }
    }

    return stripTrailingSlash(explicit);
  }

  if (typeof window !== 'undefined') {
    const baseUrl = import.meta.env.BASE_URL ?? '/';
    try {
      const publicBase = new URL(baseUrl, window.location.href);
      const apiUrl = new URL('./api/', publicBase);
      return stripTrailingSlash(apiUrl.toString());
    } catch {
      return stripTrailingSlash(baseUrl);
    }
  }

  return 'http://backend:3000';
}

function stripTrailingSlash(value) {
  if (value === '') return '';
  return value.endsWith('/') ? value.slice(0, -1) : value;
}

const fetchHealth = async () => {
  const res = await axios.get(`${API_BASE}/healthz`, {
    withCredentials: true
  });
  return res.data;
};

export default function App() {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['healthz'],
    queryFn: fetchHealth
  });

  return (
    <main style={{ fontFamily: 'system-ui, sans-serif', margin: '3rem auto', maxWidth: '40rem' }}>
      <h1>Administrative Session</h1>
      <p>This is a placeholder student interface. Replace with session dashboard.</p>
      {isLoading && <p>Checking backend health…</p>}
      {isError && <p role="alert">Backend unreachable.</p>}
      {data && <p>Backend status: {data.status}</p>}
    </main>
  );
}
