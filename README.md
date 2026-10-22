# schwab-cli

A Go command-line client for the Charles Schwab Trader API — a hand-written
port of the Python library [schwab-py](https://github.com/alexgolec/schwab-py)
(1.5.1), 1:1 in functional coverage: same endpoints, same parameters and
defaults, same enum wire values, and a token file format that is
byte-compatible with schwab-py's, so one token file can serve both tools.

## Install

```sh
# With Go 1.27+
go install github.com/rayli09/schwab-cli@latest

# Or build from source
git clone https://github.com/rayli09/schwab-cli.git
cd schwab-cli
go build -o schwab-cli .
```

## Auth setup

You need a Schwab developer application:

1. At [developer.schwab.com](https://developer.schwab.com), create an app
   with **both** API products: *Accounts and Trading Production* and
   *Market Data Production*.
2. Set the app's callback URL to `https://127.0.0.1` (it must match exactly;
   wait until the app's status is **Ready for use** — "Approved - Pending"
   apps cannot call the API).
3. Log in once to create a token file:

```sh
export SCHWAB_API_KEY=<your app key>
export SCHWAB_APP_SECRET=<your app secret>
schwab-cli auth login
```

`auth login` prints the authorization URL (schwab-py's
`client_from_manual_flow`). Open it, log in, allow access, then paste the
entire redirect URL back at the prompt. The token is written to
`./token.json` (override with `--token-path` / `SCHWAB_TOKEN_PATH`) in
schwab-py's format, with owner-only permissions.

**The 7-day rule:** Schwab refresh tokens expire absolutely **7 days** after
the original login and cannot be renewed. Access tokens are refreshed
automatically; once the grant is 7 days old, commands fail with a clear
error telling you to re-run `schwab-cli auth login`. No command ever prints
token values — `schwab-cli auth status` reports only ages and expiry state.

## Usage

Every `Client` method in schwab-py is a subcommand (kebab-case). Output is
pretty-printed JSON. Examples:

```sh
schwab-cli get-quote AAPL --fields quote,fundamental
schwab-cli get-quotes AAPL MSFT '$SPX'
schwab-cli get-option-chain AAPL --contract-type CALL --strike-count 10
schwab-cli get-price-history-every-day AAPL
schwab-cli get-account-numbers            # account number -> account hash
schwab-cli get-account <account-hash> --fields positions
schwab-cli get-orders-for-account <account-hash> --status WORKING
schwab-cli get-transactions <account-hash> --types TRADE,DIVIDEND_OR_INTEREST
schwab-cli stream --equities AAPL,MSFT   # JSON lines until Ctrl-C
```

Global flags: `--api-key`, `--app-secret`, `--token-path`, `--timeout`.
Every command has `--help`.

### Order safety: dry-run by default

`place-order`, `replace-order`, and `cancel-order` **send nothing** unless
you pass `--execute`. Without it they print the exact request (method, path,
JSON body) they would send. Order specifications are the same JSON objects
schwab-py accepts as `order_spec`; supply one with `--file`, `--json`, or
stdin:

```sh
schwab-cli place-order <account-hash> --file order.json          # dry-run
schwab-cli place-order <account-hash> --file order.json --execute
schwab-cli preview-order <account-hash> --file order.json         # API-side preview
```

## Parity checklist (schwab-py 1.5.1 → schwab-cli)

### Auth (`schwab.auth`)

| schwab-py | schwab-cli | Status |
|---|---|---|
| `client_from_manual_flow` | `auth login` | ✅ ported |
| `client_from_token_file` | token loading in every command | ✅ ported |
| `easy_client` | token file auto-load + auto-refresh in every command | ✅ ported (semantics) |
| `get_auth_context` / `client_from_received_url` | steps inside `auth login` | ✅ ported |
| `TokenMetadata.token_age` | `auth status` | ✅ ported |
| `client_from_login_flow` | — | ⚠️ equivalent: it automates a browser + local HTTPS callback; `auth login` performs the same grant via copy-paste |
| `client_from_access_functions` | — | ➖ library-embedding API; not applicable to a CLI |

Token file: `{"creation_timestamp": <unix>, "token": {...}}` — identical to
schwab-py, including preservation of the original creation timestamp across
refreshes (so the 7-day clock behaves identically) and lossless round-trip
of token fields this tool does not model.

### Trader API (`Client`)

| schwab-py | schwab-cli |
|---|---|
| `get_account_numbers` | `get-account-numbers` |
| `get_accounts` | `get-accounts` |
| `get_account` | `get-account` |
| `get_order` | `get-order` |
| `cancel_order` | `cancel-order` (dry-run unless `--execute`) |
| `get_orders_for_account` | `get-orders-for-account` |
| `get_orders_for_all_linked_accounts` | `get-orders-for-all-linked-accounts` |
| `place_order` | `place-order` (dry-run unless `--execute`) |
| `replace_order` | `replace-order` (dry-run unless `--execute`) |
| `preview_order` | `preview-order` |
| `get_transactions` | `get-transactions` |
| `get_transaction` | `get-transaction` |
| `get_user_preferences` | `get-user-preferences` |
| `schwab.utils.Utils.extract_order_id` | applied after `place-order --execute` |

### Market data API (`Client`)

| schwab-py | schwab-cli |
|---|---|
| `get_quote` | `get-quote` |
| `get_quotes` | `get-quotes` |
| `get_option_chain` | `get-option-chain` |
| `get_option_expiration_chain` | `get-option-expiration-chain` |
| `get_price_history` | `get-price-history` |
| `get_price_history_every_minute` | `get-price-history-every-minute` |
| `get_price_history_every_five_minutes` | `get-price-history-every-five-minutes` |
| `get_price_history_every_ten_minutes` | `get-price-history-every-ten-minutes` |
| `get_price_history_every_fifteen_minutes` | `get-price-history-every-fifteen-minutes` |
| `get_price_history_every_thirty_minutes` | `get-price-history-every-thirty-minutes` |
| `get_price_history_every_day` | `get-price-history-every-day` |
| `get_price_history_every_week` | `get-price-history-every-week` |
| `get_movers` | `get-movers` |
| `get_market_hours` (one or many markets) | `get-market-hours` |
| `get_instruments` | `get-instruments` |
| `get_instrument_by_cusip` | `get-instrument-by-cusip` |

### Enums

All 34 enum classes in schwab-py — the 19 `Client` enums (`internal/client`)
and the 15 order enums in `schwab/orders/common.py` (`internal/orders`) —
are ported as Go string/int types and were **machine-verified to have
identical wire-value sets** against the schwab-py source. Invalid values are
rejected with an error, mirroring schwab-py's `EnumEnforcer`.

The fluent order **builders** (`schwab.orders.equities`, `.options`,
`.generic`) are not separate commands: a CLI takes the finished order JSON
directly (the same object `place_order` accepts). The enum vocabulary the
builders use is fully ported in `internal/orders`.

### Streaming (`schwab.stream.StreamClient`)

`schwab-cli stream` ports the streaming basics: user-preferences discovery,
the ADMIN LOGIN handshake, subscriptions, a JSON-lines message loop, and
ADMIN LOGOUT on exit. All 13 subscription services are supported:

`LEVELONE_EQUITIES`, `LEVELONE_OPTIONS`, `LEVELONE_FUTURES`,
`LEVELONE_FOREX`, `LEVELONE_FUTURES_OPTIONS`, `CHART_EQUITY`,
`CHART_FUTURES`, `ACCT_ACTIVITY`, `NYSE_BOOK`, `NASDAQ_BOOK`,
`OPTIONS_BOOK`, `SCREENER_EQUITY`, `SCREENER_OPTION` — via the flags
`--equities`, `--options`, `--futures`, `--forex`, `--futures-options`,
`--chart-equity`, `--chart-futures`, `--account-activity`, `--nyse-book`,
`--nasdaq-book`, `--options-book`, `--screener-equity`, `--screener-option`.

Differences from the Python library, by design: messages are delivered raw
(schwab-py relabels numeric field keys to names in per-service handler
callbacks), and subscriptions are fixed at launch — schwab-py's mid-session
`*_unsubs` / `*_add` operations have no CLI analogue; run one `stream`
process per subscription set.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

CI (GitHub Actions) runs gofmt, vet, build, and tests on every push.

## License & attribution

MIT. This project is a port of
[schwab-py](https://github.com/alexgolec/schwab-py) — Copyright (c) 2023
Alex Golec and the schwab-py contributors (see `LICENSE` and `NOTICE`).
Not affiliated with or endorsed by Charles Schwab & Co., Inc.
