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
        'expo-app-id': 'b97123c0-7e77-48cf-826c-90e38e175aba',
      },
    },
  }
}
