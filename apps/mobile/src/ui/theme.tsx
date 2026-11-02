import { createContext, useContext } from 'react';
import type { ReactNode } from 'react';
import { useColorScheme } from 'react-native';
const light = { bg: '#F7F8F2', surface: '#FFFFFF', ink: '#19231B', muted: '#556257', accent: '#D6F45B', action: '#244D36', actionText: '#FFFFFF', border: '#788877', error: '#B42318' };
const dark = { bg: '#141C17', surface: '#1F2B23', ink: '#F3F6EF', muted: '#B8C6BB', accent: '#D6F45B', action: '#D6F45B', actionText: '#19231B', border: '#819684', error: '#FFB4AB' };
const Theme = createContext(light);
export function ThemeProvider({ children }: { children: ReactNode }) { const scheme = useColorScheme(); return <Theme.Provider value={scheme === 'dark' ? dark : light}>{children}</Theme.Provider>; }
export const useTheme = () => useContext(Theme);
