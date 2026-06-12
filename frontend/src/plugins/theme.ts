import { type ThemeDefinition } from 'vuetify'

export const lightTheme: ThemeDefinition = {
    dark: false,
    colors: {
        background: '#FAFAFA',
        surface: '#FFFFFF',
        primary: '#F06292',
        secondary: '#42A5F5',
        error: '#FF5252',
        info: '#2196F3',
        success: '#4CAF50',
        warning: '#FB8C00',
        appbar: '#F06292'
    }
}

export const darkTheme: ThemeDefinition = {
    dark: true,
    colors: {
        background: '#121212',
        surface: '#1E1E1E',
        primary: '#F491B2',
        secondary: '#90CAF9',
        error: '#FF5252',
        info: '#2196F3',
        success: '#4CAF50',
        warning: '#FB8C00',
        appbar: '#212121'
    }
}

// Selectable accent (primary) colors. Each preset carries a light/dark variant
// so the accent stays legible in both themes.
export interface AccentPreset {
    id: string
    name: string
    light: string
    dark: string
}

export const accentPresets: AccentPreset[] = [
    { id: 'pink', name: '樱粉', light: '#F06292', dark: '#F491B2' },
    { id: 'blue', name: '天蓝', light: '#1E88E5', dark: '#64B5F6' },
    { id: 'purple', name: '雅紫', light: '#7E57C2', dark: '#B39DDB' },
    { id: 'teal', name: '青碧', light: '#00897B', dark: '#4DB6AC' },
    { id: 'orange', name: '暖橙', light: '#FB8C00', dark: '#FFB74D' },
    { id: 'green', name: '草绿', light: '#43A047', dark: '#81C784' },
    { id: 'red', name: '绯红', light: '#E53935', dark: '#EF5350' },
]

export const defaultAccentId = 'pink'
export const accentStorageKey = 'lolia.accent'

export const findAccentPreset = (id: string): AccentPreset =>
    accentPresets.find((preset) => preset.id === id) ??
    accentPresets.find((preset) => preset.id === defaultAccentId) ??
    accentPresets[0]

export const readSavedAccentId = (): string => {
    try {
        const saved = localStorage.getItem(accentStorageKey)
        if (saved && accentPresets.some((preset) => preset.id === saved)) {
            return saved
        }
    } catch {
        // ignore localStorage errors
    }
    return defaultAccentId
}

export const saveAccentId = (id: string): void => {
    try {
        localStorage.setItem(accentStorageKey, id)
    } catch {
        // ignore localStorage errors
    }
}

type ThemeColorMap = { colors: Record<string, string> }

// Writes the accent's primary color onto the given light/dark theme color maps.
// In light mode the app bar tracks the accent; in dark mode the app bar stays
// neutral, so only primary changes there.
export const applyAccentColors = (
    themes: { lightTheme?: ThemeColorMap; darkTheme?: ThemeColorMap },
    id: string,
): void => {
    const preset = findAccentPreset(id)
    if (themes.lightTheme) {
        themes.lightTheme.colors.primary = preset.light
        themes.lightTheme.colors.appbar = preset.light
    }
    if (themes.darkTheme) {
        themes.darkTheme.colors.primary = preset.dark
    }
}
