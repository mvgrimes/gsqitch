# App::Sqitch add fixture (mysql)

Source:
- Commands:
  - `sqitch init myproj --engine mysql --top-dir . --plan-file sqitch.plan --extension sql`
  - `sqitch add --change widgets --note "Add widgets"`
- App::Sqitch version: `v1.6.1` (sqitch/sqitch:latest)

Notes:
- Captured from a clean working directory.
- Includes generated `sqitch.conf`, `sqitch.plan`, and deploy/revert/verify scripts.
- The plan entry includes the timestamp and author reported by the container.
