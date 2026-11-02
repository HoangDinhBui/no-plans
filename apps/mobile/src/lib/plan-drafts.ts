import AsyncStorage from '@react-native-async-storage/async-storage';
import { randomUUID } from 'expo-crypto';
import { DraftStore } from './drafts';
// One writer coordinates saves from screens during navigation transitions.
export const planDrafts = new DraftStore(AsyncStorage, randomUUID);
