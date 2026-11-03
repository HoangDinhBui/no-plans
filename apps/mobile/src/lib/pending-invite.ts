import 'react-native-url-polyfill/auto';
import * as SecureStore from 'expo-secure-store';
import { IncomingInvite } from './incoming';
export const incomingInvite = new IncomingInvite({ getItem: key => SecureStore.getItemAsync(key), setItem: (key, value) => SecureStore.setItemAsync(key, value), removeItem: key => SecureStore.deleteItemAsync(key) });
