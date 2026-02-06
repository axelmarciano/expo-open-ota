# Expo Open OTA
![Expo Open OTA Deployment](apps/docs/static/img/social_card.png)

[![Push workflow](https://github.com/axelmarciano/expo-open-ota/actions/workflows/push.yml/badge.svg)](https://github.com/axelmarciano/expo-open-ota/actions/workflows/push.yml)

**A self-hosted OTA update server for Expo apps. No database, no complex infra — just a single Docker container with S3 or a local volume.**

> **Not affiliated with [Expo](https://expo.dev/).** This is an independent open-source project.

## Why Expo Open OTA?

| | Expo Open OTA | Expo EAS Updates |
| --- | --- | --- |
| **Infrastructure** | Single container, no database | Managed service |
| **Storage** | S3 / local volume — your choice | Expo-managed |
| **Pricing** | Free forever (you pay only for your own infra) | Per MAU pricing |
| **Network control** | Full control — VPN, proxy, air-gapped | Internet required |
| **CDN** | Bring your own (CloudFront, etc.) | Expo CDN |
| **Code signing** | Built-in | Built-in |
| **Dashboard** | Included | Expo dashboard |

**Zero database.** Updates are stored directly on S3 or a local volume. No Postgres, no Redis required (Redis is optional for caching). One container, one storage backend — that's it.

## Quick Start

### Deploy to Railway (recommended)

The fastest way to get a production-ready server:

[![Deploy on Railway](https://railway.com/button.svg)](https://railway.com/template/MGW3k1?referralCode=OEHlEK)

Click the button, fill in your environment variables, and you're live.

### Run locally with Docker

```bash
docker run -p 3000:3000 \
  -v ./updates:/updates \
  -e STORAGE_MODE=local \
  -e BASE_URL=http://localhost:3000 \
  -e EXPO_ACCESS_TOKEN=your_token \
  -e EXPO_APP_ID=your_app_id \
  -e JWT_SECRET=your_secret \
  -e ADMIN_PASSWORD=your_password \
  -e USE_DASHBOARD=true \
  ghcr.io/axelmarciano/expo-open-ota:latest
```

Your server is now running at `http://localhost:3000` with the dashboard enabled.

> **Need your credentials?** See [Prerequisites](https://axelmarciano.github.io/expo-open-ota/docs/prerequisites) to get your `EXPO_ACCESS_TOKEN` and `EXPO_APP_ID`.

## Features

- **No database** — S3 or local volume, nothing else needed
- **Single container** — deploy anywhere Docker runs
- **Multi-arch** — amd64 and arm64 images available
- **Cloud storage** — AWS S3, Cloudflare R2, MinIO, or local filesystem
- **CDN ready** — CloudFront integration with signed URLs
- **Code signing** — built-in Expo code signing support
- **Dashboard** — web UI to monitor updates and branches
- **Helm chart** — one-command Kubernetes deployment

## 📖 Documentation

Full documentation: [axelmarciano.github.io/expo-open-ota](https://axelmarciano.github.io/expo-open-ota/)

## 📜 License

MIT — see [LICENSE](./LICENSE.md).

## Contact

✉️ [E-mail](mailto:expoopenota@gmail.com)
