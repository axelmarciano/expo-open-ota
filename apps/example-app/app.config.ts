import { ExpoConfig } from '@expo/config-types'
import { ConfigContext } from '@expo/config'

export default ({ config }: ConfigContext): ExpoConfig => {
  return {
    ...(config as ExpoConfig),
    runtimeVersion: '1.0.0',
    updates: {
      url: 'https://otatest.ngrok.io/manifest',
      codeSigningMetadata: {
        keyid: 'main',
        alg: 'rsa-v1_5-sha256',
      },
      codeSigningCertificate: './certs/certificate.pem',
      enabled: true,
      requestHeaders: {
        'expo-channel-name': process.env.RELEASE_CHANNEL,
        'expo-app-id': '269795f1-f208-4b8d-9508-73353a5495fb',
      },
    },
  }
}
