create table if not exists sites (
  id integer primary key autoincrement,
  name text not null,
  domain text not null,
  created_at text not null default (datetime('now'))
);

create table if not exists events (
  id integer primary key autoincrement,
  site_id integer not null,
  event_type text not null,
  path text not null,
  title text,
  referrer text,
  ref_domain text,
  source text,
  utm_source text,
  utm_medium text,
  utm_campaign text,
  entry_url text,
  meta text,
  screen_w integer,
  screen_h integer,
  x real,
  y real,
  session_id text,
  user_id text,
  ip text,
  user_agent text,
  created_at text not null default (datetime('now'))
);

create index if not exists idx_events_site_time on events(site_id, created_at);
create index if not exists idx_events_path on events(site_id, path);
create index if not exists idx_events_type on events(site_id, event_type);

create table if not exists event_daily (
  day text not null,
  site_id integer not null,
  event_type text not null,
  path text not null default '',
  count integer not null,
  primary key (day, site_id, event_type, path)
);

create index if not exists idx_event_daily_site_day on event_daily(site_id, day);
