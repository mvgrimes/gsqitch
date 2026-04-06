# App::Sqitch deploy fixture (mysql registry)

Source:
- Commands:
  - `sqitch init myproj --engine mysql --top-dir . --plan-file sqitch.plan --extension sql --target db:mysql://sqitch:sqitch@localhost:3307/test --registry sqitch`
  - `sqitch add --change widgets --note "Add widgets"`
  - `sqitch deploy --target db:mysql://sqitch:sqitch@host.containers.internal:3307/test --registry sqitch`
- App::Sqitch version: `v1.6.1` (sqitch/sqitch:latest)

Notes:
- Fixtures capture the contents of the `sqitch` database.
- Timestamp and host fields reflect the container run and will vary.
