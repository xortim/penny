# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Add test coverage, SlackAPI interface, and integration test harness (#34)
- Add threaded reply support and fix early return bug (#37)
- Add goreleaser config and release Make targets
- Add git-cliff changelog generation
- Add "what's new?" @mention command (#14)
- Add structured debug logging to whatsnew gadget

### Fixed

- Fix Slack mrkdwn rendering in whatsnew responses

### Other

- Auto-join spam channel, Go 1.26, and slack-go v0.18 compat (#36)
- Display "Latest Changes" instead of "Unreleased" in whatsnew responses
- Use Slack Markdown Block for proper bullet rendering in whatsnew

### Removed

- Remove global mutable state, tighten bold regex, and improve UX
- Remove dead mrkdwn code, improve test assertions, and fix block ID

## [0.1] - 2026-02-21

### Changed

- Update main (#33)

### Other

- Initial

