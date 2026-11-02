import { Stack } from 'expo-router';
import { SafeAreaProvider, SafeAreaView } from 'react-native-safe-area-context';
import { StatusBar } from 'expo-status-bar';
import { SessionProvider, useSession } from '../lib/session';
import { ThemeProvider, useTheme } from '../ui/theme';
import { AuthScreen } from '../features/AuthScreen';
function Routes() {
  const { session, configured } = useSession(); const c = useTheme();
  if (!session || !configured) return <SafeAreaView style={{ flex: 1, backgroundColor: c.bg }}><AuthScreen /></SafeAreaView>;
  return <Stack key={session.user.id} screenOptions={{ headerStyle: { backgroundColor: c.bg }, headerTintColor: c.ink, contentStyle: { backgroundColor: c.bg }, title: 'NO PLANS ↗', headerBackTitle: 'Quay lại' }}><Stack.Screen name="index" options={{ title: 'NO PLANS ↗' }} /><Stack.Screen name="plans/new" options={{ title: 'Lên kèo' }} /><Stack.Screen name="plans/[id]" options={{ title: 'Cuộc hẹn của hội mình' }} /><Stack.Screen name="groups" options={{ title: 'Hội của bạn' }} /><Stack.Screen name="groups/[id]" options={{ title: 'Hội mình' }} /><Stack.Screen name="friends" options={{ title: 'Bạn bè' }} /></Stack>;
}
export default function Layout() { return <SafeAreaProvider><ThemeProvider><SessionProvider><StatusBar style="auto" /><Routes /></SessionProvider></ThemeProvider></SafeAreaProvider>; }
