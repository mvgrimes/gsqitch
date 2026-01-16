# Sqitch (Go Port)

A Go port of the excellent [Sqitch](https://sqitch.org/) database change management application.

## Motivation

This project is a port of the original Perl Sqitch to Go. The primary motivations for this port are:

*   **Dependency Management:** Avoiding the regular issues associated with Perl dependencies, particularly `DBD::mysql` and `SSLeay`, which can be challenging to manage across different environments.
*   **Single Binary:** Leveraging Go's ability to produce a single, native, compiled binary for easier distribution and installation.
*   **Respect for the Original:** This port is born out of great respect for Sqitch and its authors. I have been a happy user of Sqitch since at least November 2014 and believe its approach to database change management is the best in class.

## Project Status

This project is currently in active development and does not yet support all the features of the original Perl Sqitch.

### Supported Engines

*   [MySQL](https://www.mysql.com/) / [MariaDB](https://mariadb.org/)

### Supported Commands

The following core commands have been implemented (invoked as `gsqitch <command>`):

*   `init`: Initialize a new Sqitch project.
*   `add`: Add a new change to the plan.
*   `deploy`: Deploy changes to a database.
*   `revert`: Revert deployed changes.
*   `status`: Show the current deployment status.

### Planned Features (Phase 1)

See [PHASE1.md](docs/plan/PHASE1.md) for the detailed implementation plan of the initial foundation.

## Getting Started

### Prerequisites

*   Go 1.21+

### Installation

You can install the binary using `go install`:

```bash
go install ./cmd/sqitch
```

Or build it locally using [just](https://github.com/casey/just):

```bash
just build
```

The resulting binary will be named `gsqitch`.

## How it Works

Sqitch is a database change management application. What makes it different from your typical migration approaches?

*   **No opinions:** Sqitch is not tied to any framework, ORM, or platform.
*   **Native scripting:** Changes are implemented as scripts native to your selected database engine.
*   **Dependency resolution:** Database changes may declare dependencies on other changes.
*   **Deployment integrity:** Sqitch manages changes and dependencies via a plan file, ensuring deployment integrity.
*   **Iterative Development:** You can modify your change deployment scripts as often as you like until they are tagged and released.

## Acknowledgments

This project is a port of the original [Sqitch](https://github.com/sqitchers/sqitch) created by David E. Wheeler and many contributors.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
