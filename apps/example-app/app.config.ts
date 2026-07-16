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
      codeSigningCertificate: './certs/certificate-dev.pem',
      enabled: true,
      requestHeaders: {
        'expo-channel-name': process.env.RELEASE_CHANNEL,
        'expo-app-id': '04f244ae-9e9b-44ca-80d6-6790d3ed88a0',
      },
    },
  }
}
