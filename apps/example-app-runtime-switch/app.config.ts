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
      disableAntiBrickingMeasures: true,
      requestHeaders: {
        'expo-channel-name': process.env.RELEASE_CHANNEL,
        'expo-app-id': 'd8471dfc-c3e9-4e14-afd9-21dc34cc498a',
      },
    },
  }
}
