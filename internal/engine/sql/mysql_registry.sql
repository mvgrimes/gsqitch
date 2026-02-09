-- Sqitch registry schema for MySQL/MariaDB
-- This file is embedded into the Go binary

CREATE TABLE IF NOT EXISTS `releases` (
    `version`         REAL         NOT NULL PRIMARY KEY,
    `installed_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `installer_name`  VARCHAR(255) NOT NULL,
    `installer_email` VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS `projects` (
    `project`       VARCHAR(255) NOT NULL PRIMARY KEY,
    `uri`           VARCHAR(512),
    `created_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `creator_name`  VARCHAR(255) NOT NULL,
    `creator_email` VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS `changes` (
    `change_id`       CHAR(40)     NOT NULL PRIMARY KEY,
    `script_hash`     CHAR(40),
    `change`          VARCHAR(255) NOT NULL,
    `project`         VARCHAR(255) NOT NULL REFERENCES `projects`(`project`),
    `note`            TEXT         NOT NULL,
    `committed_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `committer_name`  VARCHAR(255) NOT NULL,
    `committer_email` VARCHAR(255) NOT NULL,
    `planned_at`      DATETIME(6)  NOT NULL,
    `planner_name`    VARCHAR(255) NOT NULL,
    `planner_email`   VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS `tags` (
    `tag_id`          CHAR(40)     NOT NULL PRIMARY KEY,
    `tag`             VARCHAR(255) NOT NULL,
    `project`         VARCHAR(255) NOT NULL REFERENCES `projects`(`project`),
    `change_id`       CHAR(40)     NOT NULL REFERENCES `changes`(`change_id`),
    `note`            TEXT         NOT NULL,
    `committed_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `committer_name`  VARCHAR(255) NOT NULL,
    `committer_email` VARCHAR(255) NOT NULL,
    `planned_at`      DATETIME(6)  NOT NULL,
    `planner_name`    VARCHAR(255) NOT NULL,
    `planner_email`   VARCHAR(255) NOT NULL,
    UNIQUE(`project`, `tag`)
);

CREATE TABLE IF NOT EXISTS `dependencies` (
    `change_id`     CHAR(40)     NOT NULL REFERENCES `changes`(`change_id`),
    `type`          VARCHAR(16)  NOT NULL,
    `dependency`    VARCHAR(255) NOT NULL,
    `dependency_id` CHAR(40)     REFERENCES `changes`(`change_id`),
    PRIMARY KEY(`change_id`, `dependency`)
);

CREATE TABLE IF NOT EXISTS `events` (
    `event`           VARCHAR(16)  NOT NULL,
    `change_id`       CHAR(40)     NOT NULL,
    `script_hash`     CHAR(40),
    `change`          VARCHAR(255) NOT NULL,
    `project`         VARCHAR(255) NOT NULL REFERENCES `projects`(`project`),
    `note`            TEXT         NOT NULL,
    `requires`        TEXT         NOT NULL,
    `conflicts`       TEXT         NOT NULL,
    `tags`            TEXT         NOT NULL,
    `committed_at`    DATETIME(6)  NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
    `committer_name`  VARCHAR(255) NOT NULL,
    `committer_email` VARCHAR(255) NOT NULL,
    `planned_at`      DATETIME(6)  NOT NULL,
    `planner_name`    VARCHAR(255) NOT NULL,
    `planner_email`   VARCHAR(255) NOT NULL
);
