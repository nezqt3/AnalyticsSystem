import { render, screen } from '@testing-library/react';

import { PageOverview } from './PageOverview';

describe('PageOverview', () => {
  it('renders empty state when analytics are missing', () => {
    render(<PageOverview analytics={null} />);

    expect(screen.getByText(/выберите страницу/i)).toBeInTheDocument();
  });

  it('renders metrics and top targets', () => {
    render(
      <PageOverview
        analytics={{
          path: '/pricing',
          pageviews: 12,
          clicks: 5,
          form_submissions: 2,
          unique_visitors: 7,
          avg_scroll_depth: 64,
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
          scroll_depths: [{ depth: 75, count: 4 }],
          last_interaction_at: '2026-03-16 09:30:00',
        }}
      />,
    );

    expect(screen.getByText('12')).toBeInTheDocument();
    expect(screen.getByText('button.primary')).toBeInTheDocument();
    expect(screen.getByText('60%')).toBeInTheDocument();
    expect(screen.getByText('75%')).toBeInTheDocument();
  });
});
