# terrifi

> [!IMPORTANT]
> **This fork is retired (2026-11-02).** Use the upstream provider,
> [`alexklibisz/terrifi`](https://github.com/alexklibisz/terraform-provider-terrifi)
> ([OpenTofu Registry](https://search.opentofu.org/provider/alexklibisz/terrifi/latest)).
> The fork existed to carry fixes while they were upstreamed. They have since been merged
> upstream or replaced by the maintainer's own work. The `terrifi_network` PXE-boot and
> `domain_name` options are still being proposed upstream
> ([#264](https://github.com/alexklibisz/terraform-provider-terrifi/issues/264)). The published `fdcastel/terrifi` versions (up to v0.10.0) stay in the registry
> but get no further updates. Note that `fdcastel/terrifi` 0.10.0 and upstream 0.10.0 are different
> code, so pin upstream **0.12.0 or later** when you switch.

Yet another Terraform provider for UniFi.

## Introduction

This is my attempt at making a working Terraform provider to manage my home UniFi network.

Full disclosure, much of this code is written by and with help from various AI coding agents (Claude code in particular).

Compared to existing UniFi providers ([paultyng](https://github.com/paultyng/terraform-provider-unifi), [filipowm](https://github.com/filipowm/terraform-provider-unifi), [ubiquiti-community](https://github.com/ubiquiti-community/terraform-provider-unifi)), Terrifi is a nearly-from-scratch implementation with a particular focus on extensive testing, including hardware-in-the-loop testing.

## Docs

- [OpenTofu Registry](https://search.opentofu.org/provider/alexklibisz/terrifi/latest)
- [Provider](./docs/index.md)
- [CLI](./docs/index.md#cli)
- [Introductory Blog Post](https://alexklibisz.com/2026/03/07/terrifi)
- [Development](./docs/development.md)
