import { useQuery } from '@tanstack/react-query';
import axios from 'axios';

const API_BASE = resolveBackendBase();

function resolveBackendBase() {
  const explicit = import.meta.env.VITE_BACKEND_URL;
  if (explicit) return explicit;

  if (typeof window !== 'undefined') {
    const { protocol, hostname } = window.location;
    const port = import.meta.env.VITE_BACKEND_PORT ?? '3000';
    return `${protocol}//${hostname}:${port}`;
  }

  return 'http://localhost:3000';
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
