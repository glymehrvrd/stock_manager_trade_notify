create table stared_stocks (
  id   serial primary key  not null,
  name text,
  code text unique
);

create table manager_trade (
  id                 serial primary key  not null,
  name               text,
  code               text,
  trade_date         date,
  trader             text,
  trade_count        int,
  transaction_price  money,
  transaction_reason text,
  transaction_amount int
)