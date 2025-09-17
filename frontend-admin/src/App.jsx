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
    <main style={{ fontFamily: 'system-ui, sans-serif', margin: '3rem auto', maxWidth: '70rem', padding: '0 1rem' }}>
      <header style={{ marginBottom: '2rem' }}>
        <h1>ADM Admin Dashboard</h1>
        <p>Manage annual administrative sessions and track student progress.</p>
      </header>

      <section style={{ marginBottom: '3rem' }}>
        <h2>Create a New ADM Session</h2>
        <form onSubmit={handleSubmit} style={{ display: 'grid', gap: '0.75rem', maxWidth: '32rem' }}>
          <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            <span>Label (optional)</span>
            <input
              type="text"
              placeholder="ADM 2025"
              value={label}
              onChange={(event) => setLabel(event.target.value)}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            <span>Start date</span>
            <input
              type="datetime-local"
              required
              value={startAt}
              onChange={(event) => setStartAt(event.target.value)}
            />
          </label>

          <label style={{ display: 'flex', flexDirection: 'column', gap: '0.25rem' }}>
            <span>End date</span>
            <input
              type="datetime-local"
              required
              value={endAt}
              onChange={(event) => setEndAt(event.target.value)}
            />
          </label>

          <button type="submit" disabled={mutation.isPending} style={{ padding: '0.5rem 1.25rem', cursor: mutation.isPending ? 'wait' : 'pointer' }}>
            {mutation.isPending ? 'Creating…' : 'Create session'}
          </button>

          {formError && (
            <p role="alert" style={{ color: '#d32f2f' }}>
              {formError}
            </p>
          )}
        </form>
      </section>

      <section>
        <h2>Existing Sessions</h2>
        {isLoading && <p>Loading sessions…</p>}
        {isError && <p role="alert">Unable to load sessions.</p>}
        {!isLoading && !isError && sessions.length === 0 && <p>No ADM sessions yet.</p>}

        {!isLoading && !isError && sessions.length > 0 && (
          <div style={{ overflowX: 'auto' }}>
            <table style={{ borderCollapse: 'collapse', width: '100%' }}>
              <thead>
                <tr>
                  <th style={cellStyle}>Label</th>
                  <th style={cellStyle}>Dates</th>
                  <th style={cellStyle}>Status</th>
                  <th style={cellStyle}>Ongoing</th>
                  <th style={cellStyle}>Progress</th>
                </tr>
              </thead>
              <tbody>
                {sessions.map((session) => (
                  <tr key={session.id}>
                    <td style={cellStyle}>{session.label}</td>
                    <td style={cellStyle}>
                      <div>{formatDateTime(session.start_at)}</div>
                      <div style={{ fontSize: '0.85rem', color: '#666' }}>to {formatDateTime(session.end_at)}</div>
                    </td>
                    <td style={cellStyle}>{session.status}</td>
                    <td style={cellStyle}>{session.is_ongoing ? 'Yes' : 'No'}</td>
                    <td style={cellStyle}>
                      {session.validated_count} / {session.student_count}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {activeSessions.length > 0 && (
          <p style={{ marginTop: '1rem', color: '#2e7d32' }}>
            {activeSessions.length} session{activeSessions.length > 1 ? 's are' : ' is'} currently ongoing.
          </p>
        )}
      </section>
    </main>
  );
}

const cellStyle = {
  borderBottom: '1px solid #ddd',
  padding: '0.75rem',
  textAlign: 'left',
  verticalAlign: 'top'
};
