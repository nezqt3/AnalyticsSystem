import { fireEvent, render, screen, waitFor } from '@testing-library/react';

jest.mock('./env', () => ({
  API_BASE_URL: 'http://localhost:8080',
  DEFAULT_SITE_ID: 1,
}));

import App from './App';
import * as api from './api';

jest.mock('./api', () => ({
  API_BASE: 'http://localhost:8080',
  getRealtime: jest.fn(),
  getPages: jest.fn(),
  getTrafficSources: jest.fn(),
  getHeatmap: jest.fn(),
  getEvents: jest.fn(),
  getPageAnalytics: jest.fn(),
}));

const mockedApi = api as jest.Mocked<typeof api>;

beforeEach(() => {
  mockedApi.getRealtime.mockResolvedValue({
    active_users: 4,
    series: [{ minute: '10:00', count: 3 }],
  });
  mockedApi.getPages.mockResolvedValue([
    {
      path: '/pricing',
      pageviews: 10,
      clicks: 5,
      form_submissions: 1,
      unique_visitors: 4,
      last_seen: '2026-03-16 10:00:00',
    },
    {
      path: '/blog',
      pageviews: 4,
      clicks: 1,
      form_submissions: 0,
      unique_visitors: 3,
      last_seen: '2026-03-16 09:00:00',
    },
  ]);
  mockedApi.getTrafficSources.mockResolvedValue([{ source: 'google', count: 7 }]);
  mockedApi.getHeatmap.mockResolvedValue([{ x_pct: 20, y_pct: 30, count: 4 }]);
  mockedApi.getEvents.mockResolvedValue([
    {
      created_at: '2026-03-16 10:00:00',
      event_type: 'click',
      path: '/pricing',
      title: 'Pricing',
      meta: '{"selector":"button.primary"}',
      ref_domain: '',
      utm_source: 'google',
      utm_medium: '',
      utm_campaign: '',
    },
  ]);
  mockedApi.getPageAnalytics.mockResolvedValue({
    path: '/pricing',
    pageviews: 10,
    clicks: 5,
    form_submissions: 1,
    unique_visitors: 4,
    avg_scroll_depth: 67,
    top_targets: [
      {
        selector: 'button.primary',
        text: 'Start trial',
        href: '',
        tag: 'button',
        count: 3,
        share: 0.6,
      },
    ],
    scroll_depths: [{ depth: 80, count: 3 }],
    last_interaction_at: '2026-03-16 10:00:00',
  });

  class MockWebSocket {
    static instances: MockWebSocket[] = [];
    onmessage: ((event: MessageEvent) => void) | null = null;
    onclose: (() => void) | null = null;
    onerror: (() => void) | null = null;

    constructor() {
      MockWebSocket.instances.push(this);
    }

    close() {}
  }

  Object.defineProperty(window, 'WebSocket', {
    writable: true,
    value: MockWebSocket,
  });
});

afterEach(() => {
  jest.clearAllMocks();
});

describe('App', () => {
  it('loads dashboard data and switches selected page', async () => {
    render(<App />);

    expect(await screen.findByText(/панель поведения пользователей/i)).toBeInTheDocument();
    expect(await screen.findByText('/pricing')).toBeInTheDocument();

    await waitFor(() => {
      expect(mockedApi.getPageAnalytics).toHaveBeenCalledWith(
        1,
        '/pricing',
        expect.any(String),
        expect.any(String),
      );
    });

    fireEvent.click(screen.getByRole('button', { name: /\/blog/i }));

    await waitFor(() => {
      expect(mockedApi.getPageAnalytics).toHaveBeenCalledWith(
        1,
        '/blog',
        expect.any(String),
        expect.any(String),
      );
    });
  });
});
