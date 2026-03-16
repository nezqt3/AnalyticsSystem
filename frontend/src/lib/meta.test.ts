import { parseEventMeta } from './meta';

describe('parseEventMeta', () => {
  it('returns parsed metadata for valid json', () => {
    expect(parseEventMeta('{"selector":"button.primary","depth_pct":80}')).toEqual({
      selector: 'button.primary',
      depth_pct: 80,
    });
  });

  it('returns empty object for invalid json', () => {
    expect(parseEventMeta('{')).toEqual({});
  });
});
