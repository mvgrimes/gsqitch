# Podman test infrastructure

This directory provides helper scripts for running integration tests against
MariaDB and for invoking the original App::Sqitch in a container to confirm
behavior. These scripts are optional, but they establish a repeatable local
setup for parity testing.

## MariaDB container

Start a local MariaDB container for tests:

```
./test/podman/mariadb.sh start
```

Defaults (override via env vars):

- `NAME=gsqitch-mariadb`
- `IMAGE=docker.io/library/mariadb:11`
- `PORT=3307`
- `ROOT_PASSWORD=root`
- `DB=sqitch`
- `USER=sqitch`
- `PASSWORD=sqitch`

Stop, remove, or view logs:

```
./test/podman/mariadb.sh stop
./test/podman/mariadb.sh rm
./test/podman/mariadb.sh logs
```

Sample target URI for gsqitch:

```
db:mysql://sqitch:sqitch@localhost:3307/sqitch
```

## App::Sqitch reference container

Use the official Sqitch container to run the Perl implementation while keeping
your working directory mounted. This helps confirm expected behavior without
adding Perl dependencies to the host.

```
./test/podman/sqitch-perl.sh run status
./test/podman/sqitch-perl.sh run init myproj --engine mysql
```

To compare with the bundled reference source in `docs/ref`, mount that directory
and invoke the Perl binary manually inside the container if needed.

## Recreate deploy fixtures

Reset the registry database and regenerate registry fixtures using App::Sqitch:

```
mysql -h 127.0.0.1 -P 3307 -u sqitch -psqitch -e "CREATE DATABASE IF NOT EXISTS test"
mysql -h 127.0.0.1 -P 3307 -u sqitch -psqitch -e "DROP DATABASE IF EXISTS sqitch; CREATE DATABASE sqitch"

tmpdir=$(mktemp -d)
cd "$tmpdir"
/path/to/gsqitch/test/podman/sqitch-perl.sh run init myproj --engine mysql --top-dir . --plan-file sqitch.plan --extension sql --target db:mysql://sqitch:sqitch@localhost:3307/test --registry sqitch
/path/to/gsqitch/test/podman/sqitch-perl.sh run add widgets --note "Add widgets"
/path/to/gsqitch/test/podman/sqitch-perl.sh run deploy --target db:mysql://sqitch:sqitch@host.containers.internal:3307/test --registry sqitch

MYSQL_HOST=127.0.0.1 MYSQL_PORT=3307 MYSQL_USER=sqitch MYSQL_PASSWORD=sqitch MYSQL_DB=sqitch \
  OUT_DIR=/path/to/gsqitch/testdata/fixtures/deploy/mysql/registry \
  go run /path/to/gsqitch/test/podman/sqitch_dump_registry.go
```
