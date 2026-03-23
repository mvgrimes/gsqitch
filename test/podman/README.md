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
