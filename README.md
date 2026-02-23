<p align="center">
  <img src="apps/docs/static/img/social_card.png" alt="Expo Open OTA" />
</p>

<p align="center">
  <a href="https://github.com/axelmarciano/expo-open-ota/stargazers"><img src="https://img.shields.io/github/stars/axelmarciano/expo-open-ota?style=flat-square" alt="GitHub Stars" /></a>
  <a href="https://github.com/axelmarciano/expo-open-ota/blob/main/LICENSE.md"><img src="https://img.shields.io/github/license/axelmarciano/expo-open-ota?style=flat-square" alt="License" /></a>
  <a href="https://github.com/axelmarciano/expo-open-ota/releases"><img src="https://img.shields.io/github/v/release/axelmarciano/expo-open-ota?style=flat-square" alt="Release" /></a>
  <a href="https://hub.docker.com/r/axelmarciano/expo-open-ota"><img src="https://img.shields.io/docker/pulls/axelmarciano/expo-open-ota?style=flat-square" alt="Docker Pulls" /></a>
</p>

<h3 align="center">Self-hosted OTA updates for React Native — multi-cloud, production-ready.</h3>

<p align="center">
  An open-source Go server implementing the <a href="https://docs.expo.dev/technical-specs/expo-updates-1/">Expo Updates protocol</a>.<br/>
  Deploy on AWS, GCP, or locally. No vendor lock-in.
</p>

<p align="center">
  <a href="https://axelmarciano.github.io/expo-open-ota/">Documentation</a> · <a href="https://github.com/axelmarciano/expo-open-ota/issues">Issues</a> · <a href="mailto:expoopenota@gmail.com">Contact</a>
</p>

---

## Why Expo Open OTA?

- **Cut costs** — Expo's OTA pricing scales with MAUs. Self-hosting gives you unlimited updates at infrastructure cost only.
- **Own your infrastructure** — Store updates on your cloud, behind your VPN, with your security policies.
- **No vendor lock-in** — Works with AWS, GCP, and any S3-compatible provider. Switch anytime.

## Features

| Feature | Description |
|---------|-------------|
| **Multi-cloud storage** | AWS S3, Google Cloud Storage, S3-compatible (Cloudflare R2, MinIO, DigitalOcean Spaces), local file system |
| **Fast asset delivery** | CloudFront CDN, GCS signed URLs, or direct serving — your choice |
| **One-command publishing** | `npx eoas publish` from your CI/CD pipeline |
| **Secure key management** | AWS Secrets Manager, environment variables, or local key files |
| **Dashboard** | Built-in web UI for monitoring updates, branches, and runtime versions |
| **Prometheus metrics** | Production observability out of the box |
| **No database required** | Zero external dependencies beyond your storage provider |
| **Helm chart** | Ready for Kubernetes deployments |

## Quick Start

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/deploy/MGW3k1?referralCode=OEHlEK&utm_medium=integration&utm_source=template&utm_campaign=generic)

Or with Docker:

```bash
docker run -p 3000:3000 \
  -e STORAGE_MODE=s3 \
  -e S3_BUCKET_NAME=my-bucket \
  -e AWS_REGION=us-east-1 \
  axelmarciano/expo-open-ota
```

Then configure your Expo app to point to your server — see the [full documentation](https://axelmarciano.github.io/expo-open-ota/).

## Storage Options

| Provider | Mode | Asset Delivery |
|----------|------|----------------|
| **Amazon S3** | `STORAGE_MODE=s3` | Direct or CloudFront CDN |
| **Google Cloud Storage** | `STORAGE_MODE=gcs` | GCS signed URLs |
| **S3-compatible** (R2, MinIO, etc.) | `STORAGE_MODE=s3` + `AWS_BASE_ENDPOINT` | Direct |
| **Local file system** | `STORAGE_MODE=local` | Direct (dev only) |

## Disclaimer

Expo Open OTA is **not officially supported or affiliated with [Expo](https://expo.dev/)**. This is an independent open-source project.

## License

MIT — see [LICENSE](./LICENSE.md).

## Contact

[expoopenota@gmail.com](mailto:expoopenota@gmail.com)
