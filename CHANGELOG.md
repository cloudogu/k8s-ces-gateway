# k8s-ces-gateway Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Fixed
- templating-string in component-patch-tpl

## [v1.0.2] - 2025-10-01
### Fixed
- [#8] component-patch-tpl to include all necessary images 

## [v1.0.1] - 2025-09-29
### Fixed
- [#6] missing network-policy

## [v1.0.0] - 2025-09-23
### Fixed
- [#4] Fix error for invalid prefix path when using regex expressions
- use generic Namespace name from Env-Var "POD_NAMESPACE" instead of hardcoded "ecosystem"

## [v0.0.1] - 2025-09-04
### Added
- initial release of the basic k8s-ces-gateway
