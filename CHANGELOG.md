# k8s-ces-gateway Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres
to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Security
- [#21] Fix Go stdlib CVE-2025-68121

## [v2.1.0] - 2026-01-22
### Added
- [#17] Add default imagePullSecret `ces-container-registries`.

## [v2.0.1] - 2025-12-02
### Fixed
- [#15] Use correct controller class name `ingress-nginx`.

## [v2.0.0] - 2025-11-27

### Changed
- [#13] k8s-ces-gateway is now responsible to apply the ingressclass `k8s-ecosystem-ces-service` required by the ecosystem.

**Attention**:

This change requires a manuel edit of the existing ingressclass deployed by the k8s-service-discovery.

The following patch is required to add annotations and labels:

`kubectl patch ingressclass k8s-ecosystem-ces-service -p '{"metadata": {"annotations": {"meta.helm.sh/release-namespace": "ecosystem", "meta.helm.sh/release-name": "k8s-ces-gateway"}, "labels": {"app.kubernetes.io/managed-by": "Helm"}}}'`

This prevents the following error when installing this version of k8s-ces-gateway with an existing ingressclass:

`IngressClass "k8s-ecosystem-ces-service" in namespace "" exists and cannot be imported into the current release`


## [v1.0.4] - 2025-11-13

### Changed
- [#12] Disable the creation of the default ingress class. The ecosystem uses a different ingress class created by the service discovery.

### Added
- [#10] documentation how to add external ingresses

## [v1.0.3] - 2025-10-01
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
