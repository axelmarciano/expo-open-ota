import {
  Platform,
  SafeAreaView,
  ScrollView,
  Alert,
  StyleSheet,
  Button,
  ActivityIndicator,
  View,
} from 'react-native'
import * as Updates from '@latitudegames/expo-updates'
import { Picker } from '@react-native-picker/picker'
import { ThemedText } from '@/components/ThemedText'
import { ThemedView } from '@/components/ThemedView'
import Constants from 'expo-constants/src/Constants'
import { useState, useEffect } from 'react'
import { UpdatesLogViewer } from '@/components/LogViewer'

const RELEASE_CHANNELS = ['production', 'staging']

export default function HomeScreen() {
  const [loading, load] = useState<boolean>(false)
  const [logs, setLogs] = useState<Updates.UpdatesLogEntry[]>([])

  useEffect(() => {
    const fetchLogs = async () => {
      try {
        const logEntries = await Updates.readLogEntriesAsync()
        setLogs(logEntries)
      } catch (error) {
        console.error('Error fetching logs:', error)
      }
    }

    fetchLogs()
  }, [])

  const onSelectReleaseChannel = async (channel: string) => {
    if (__DEV__ || loading || Platform.OS === 'web') {
      return
    }
    Updates.setUpdateURLAndRequestHeadersOverride({
      updateUrl: Constants.expoConfig?.updates?.url as string,
      requestHeaders: {
        'expo-channel-name': channel,
        'expo-app-id': 'd8471dfc-c3e9-4e14-afd9-21dc34cc498a',
      },
    })
    await checkUpdates()
  }

  const checkUpdates = async () => {
    if (__DEV__ || loading || Platform.OS === 'web') {
      return
    }
    try {
      await Updates.clearLogEntriesAsync()
      const update = await Updates.checkForUpdateAsync()
      const logEntries = await Updates.readLogEntriesAsync()
      if (update.isAvailable) {
        load(true)
        await Updates.fetchUpdateAsync()
        setLogs(logEntries)
        load(false)
        await Updates.reloadAsync()
      } else {
        setLogs(logEntries)
        load(false)
        Alert.alert(
          'Update not available',
          'There is no new update available.',
          [
            {
              text: 'OK',
              style: 'cancel',
            },
          ],
          { cancelable: false },
        )
      }
    } catch (e) {
      load(false)
    }
  }

  if (loading) {
    return (
      <SafeAreaView style={{ flex: 1 }}>
        <View
          style={{ flex: 1, justifyContent: 'center', alignItems: 'center' }}
        >
          <ActivityIndicator size="large" color="#0000ff" />
        </View>
      </SafeAreaView>
    )
  }

  return (
    <SafeAreaView style={styles.safeAreaView}>
      <ScrollView contentContainerStyle={styles.scrollView}>
        <ThemedView style={styles.titleContainer}>
          <ThemedText type="title">Current update</ThemedText>
        </ThemedView>
        <ThemedView style={styles.informations}>
          <ThemedText>Update ID: {Updates.updateId}</ThemedText>
          <ThemedText>Runtime version: {Updates.runtimeVersion}</ThemedText>
          <ThemedText>Release channel: {Updates.channel}</ThemedText>
          <ThemedText>
            Update server url : {Constants.expoConfig?.updates?.url || ''}
          </ThemedText>
        </ThemedView>
        <ThemedView>
          <Picker
            selectedValue={Updates.channel || undefined}
            onValueChange={(val: string) => {
              if (!val) return
              return onSelectReleaseChannel(val)
            }}
          >
            {RELEASE_CHANNELS.map(channel => (
              <Picker.Item
                key={channel}
                label={channel}
                value={channel}
                testID={`release-channel-${channel}`}
              />
            ))}
          </Picker>
          <Button
            title="Check for updates"
            onPress={() => checkUpdates()}
            disabled={loading}
          />
          <UpdatesLogViewer logs={logs} />
        </ThemedView>
      </ScrollView>
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  titleContainer: {
    flexDirection: 'row',
    alignItems: 'center',
    gap: 8,
  },
  informations: {
    gap: 8,
    marginBottom: 8,
  },
  safeAreaView: {
    flex: 1,
    backgroundColor: '#fff',
  },
  scrollView: {
    flexGrow: 1,
    paddingVertical: 16,
    paddingHorizontal: 16,
  },
})
