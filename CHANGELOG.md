# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- LLM steps now support an optional `verbose` attribute to control LLM output in logs
  - When `verbose="true"`, LLM responses are logged at INFO level
  - Default is `verbose="false"` (silent mode) to maintain current behavior
