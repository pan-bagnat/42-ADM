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
    <div className="pb-module theme-dark">
      <div className="pb-module__viewport pb-module__viewport--narrow">
        <header className="pb-header">
          <h1 className="pb-header__title">Administrative Session</h1>
          <p className="pb-header__subtitle">
            This is a placeholder student interface. Replace with session dashboard content.
          </p>
        </header>

        <section className="pb-section">
          <div className="pb-card pb-card--center">
            <div className="pb-card__header">
              <h2 className="pb-card__title">Backend health</h2>
              <p className="pb-card__subtitle">Connectivity check with the ADM API.</p>
            </div>

            {isLoading && <p className="pb-message pb-message--muted">Checking backend health…</p>}
            {isError && (
              <p role="alert" className="pb-message pb-message--error">
                Backend unreachable.
              </p>
            )}
            {data && (
              <div className="pb-badge-group">
                <span className={getStatusChipClass(data.status)}>Status: {data.status}</span>
              </div>
            )}
          </div>
        </section>
      </div>
    </div>
  );
}

function getStatusChipClass(status) {
  if (!status) return 'pb-chip';
  const normalized = status.toLowerCase();
  if (normalized.includes('ok') || normalized.includes('up')) {
    return 'pb-chip pb-chip--active';
  }
  if (normalized.includes('warn')) {
    return 'pb-chip pb-chip--draft';
  }
  if (normalized.includes('down') || normalized.includes('error')) {
    return 'pb-chip pb-chip--archived';
  }
  return 'pb-chip';
}
