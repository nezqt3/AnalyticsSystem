create table if not exists sites (
  id bigserial primary key,
  name text not null,
  domain text not null,
  created_at timestamptz not null default now()
);

create table if not exists events (
  id bigserial not null,
  site_id bigint not null references sites(id) on delete cascade,
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
  screen_w integer,
  screen_h integer,
  x double precision,
  y double precision,
  session_id text,
  user_id text,
  ip inet,
  user_agent text,
  created_at timestamptz not null default now(),
  primary key (id, created_at)
)
partition by range (created_at);

create table if not exists events_default
partition of events default;

create index if not exists idx_events_site_time on events(site_id, created_at desc);
create index if not exists idx_events_path on events(site_id, path);
create index if not exists idx_events_type on events(site_id, event_type);

create table if not exists event_daily (
  day date not null,
  site_id bigint not null references sites(id) on delete cascade,
  event_type text not null,
  path text not null default '',
  count bigint not null,
  primary key (day, site_id, event_type, path)
);

create index if not exists idx_event_daily_site_day on event_daily(site_id, day desc);
