import { useMemo, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
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

const fetchSessions = async () => {
  const res = await axios.get(`${API_BASE}/admin/sessions`, {
    withCredentials: true
  });
  return res.data.sessions ?? [];
};

const createSession = async ({ label, startAt, endAt }) => {
  const payload = {
    label,
    start_at: startAt,
    end_at: endAt
  };
  const res = await axios.post(`${API_BASE}/admin/sessions`, payload, {
    withCredentials: true
  });
  return res.data.session;
};

const formatDateTime = (value) => {
  if (!value) return '-';
  try {
    return new Date(value).toLocaleString();
  } catch (error) {
    return value;
  }
};

export default function App() {
  const queryClient = useQueryClient();
  const [label, setLabel] = useState('');
  const [startAt, setStartAt] = useState('');
  const [endAt, setEndAt] = useState('');
  const [formError, setFormError] = useState('');

  const { data: sessions = [], isLoading, isError } = useQuery({
    queryKey: ['adminSessions'],
    queryFn: fetchSessions
  });

  const mutation = useMutation({
    mutationFn: createSession,
    onSuccess: () => {
      setLabel('');
      setStartAt('');
      setEndAt('');
      setFormError('');
      queryClient.invalidateQueries({ queryKey: ['adminSessions'] });
    },
    onError: (error) => {
      const message = error?.response?.data?.error ?? 'Failed to create session';
      setFormError(message);
    }
  });

  const handleSubmit = (event) => {
    event.preventDefault();
    setFormError('');

    if (!startAt || !endAt) {
      setFormError('Start date and end date are required.');
      return;
    }

    const startISO = new Date(startAt).toISOString();
    const endISO = new Date(endAt).toISOString();

    if (startISO >= endISO) {
      setFormError('End date must be after start date.');
      return;
    }

    mutation.mutate({
      label: label.trim(),
      startAt: startISO,
      endAt: endISO
    });
  };

  const activeSessions = useMemo(
    () => sessions.filter((session) => session.is_ongoing),
    [sessions]
  );

  return (
    <div className="pb-module theme-dark">
      <div className="pb-module__viewport">
        <header className="pb-header">
          <h1 className="pb-header__title">ADM Admin Dashboard</h1>
          <p className="pb-header__subtitle">Manage annual administrative sessions and track student progress.</p>
        </header>

        <section className="pb-section">
          <div className="pb-card">
            <div className="pb-card__header">
              <h2 className="pb-card__title">Create a New ADM Session</h2>
              <p className="pb-card__subtitle">Define the timeframe and publish it when you are ready.</p>
            </div>

            <form onSubmit={handleSubmit} className="pb-form">
              <div className="pb-field">
                <label className="pb-label" htmlFor="session-label">
                  Label (optional)
                </label>
                <input
                  id="session-label"
                  className="pb-input"
                  type="text"
                  placeholder="ADM 2025"
                  value={label}
                  onChange={(event) => setLabel(event.target.value)}
                />
              </div>

              <div className="pb-field">
                <label className="pb-label" htmlFor="session-start">
                  Start date
                </label>
                <input
                  id="session-start"
                  className="pb-input"
                  type="datetime-local"
                  required
                  value={startAt}
                  onChange={(event) => setStartAt(event.target.value)}
                />
              </div>

              <div className="pb-field">
                <label className="pb-label" htmlFor="session-end">
                  End date
                </label>
                <input
                  id="session-end"
                  className="pb-input"
                  type="datetime-local"
                  required
                  value={endAt}
                  onChange={(event) => setEndAt(event.target.value)}
                />
              </div>

              <button type="submit" className="pb-button" disabled={mutation.isPending}>
                {mutation.isPending ? 'Creating…' : 'Create session'}
              </button>

              {formError && (
                <p role="alert" className="pb-message pb-message--error">
                  {formError}
                </p>
              )}
            </form>
          </div>
        </section>

        <section className="pb-section">
          <div className="pb-card pb-card--table">
            <div className="pb-card__header">
              <h2 className="pb-card__title">Existing Sessions</h2>
              <p className="pb-card__subtitle">Review session dates, status, and publication progress.</p>
            </div>

            {isLoading && <p className="pb-message pb-message--muted">Loading sessions…</p>}
            {isError && (
              <p role="alert" className="pb-message pb-message--error">
                Unable to load sessions.
              </p>
            )}
            {!isLoading && !isError && sessions.length === 0 && (
              <p className="pb-empty-state">No ADM sessions yet. Create the first one above.</p>
            )}

            {!isLoading && !isError && sessions.length > 0 && (
              <div className="pb-table-wrapper">
                <table className="pb-table">
                  <thead>
                    <tr>
                      <th>Label</th>
                      <th>Dates</th>
                      <th>Status</th>
                      <th>Ongoing</th>
                      <th>Progress</th>
                    </tr>
                  </thead>
                  <tbody>
                    {sessions.map((session) => (
                      <tr key={session.id}>
                        <td>{session.label}</td>
                        <td>
                          <div>{formatDateTime(session.start_at)}</div>
                          <div className="pb-text-muted pb-text-small">to {formatDateTime(session.end_at)}</div>
                        </td>
                        <td>
                          <span className={getStatusChipClass(session.status)}>{session.status}</span>
                        </td>
                        <td>{session.is_ongoing ? 'Yes' : 'No'}</td>
                        <td>
                          {session.validated_count} / {session.student_count}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}

            {activeSessions.length > 0 && (
              <p className="pb-message pb-message--success pb-message--spaced">
                {activeSessions.length} session{activeSessions.length > 1 ? 's are' : ' is'} currently ongoing.
              </p>
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
  if (normalized.includes('active')) return 'pb-chip pb-chip--active';
  if (normalized.includes('draft')) return 'pb-chip pb-chip--draft';
  if (normalized.includes('archiv')) return 'pb-chip pb-chip--archived';
  return 'pb-chip';
}
