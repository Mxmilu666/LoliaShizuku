import { type ThemeDefinition } from 'vuetify'
import {
    argbFromHex,
    hexFromArgb,
    Hct,
    SchemeTonalSpot,
    TonalPalette,
} from '@material/material-color-utilities'

// 语义状态色的种子；按 MD3 取色调：浅色主题 tone 40、深色主题 tone 80，保证与 surface 的对比度
const statusSeeds = {
    success: '#4CAF50',
    warning: '#FB8C00',
    info: '#2196F3',
} as const

/**
 * 按 MD3 动态配色（HCT 色彩空间 + TonalSpot 方案）由种子色生成完整的色彩角色。
 *
 * 注意 Vuetify 的 `surface-variant` / `on-surface-variant` 语义是“反色表面”（tooltip、snackbar 使用），
 * 因此映射为 MD3 的 inverseSurface / inverseOnSurface；MD3 自身的 onSurfaceVariant 以
 * `md-on-surface-variant` 提供。
 */
export const buildSchemeColors = (seedHex: string, dark: boolean): Record<string, string> => {
    const scheme = new SchemeTonalSpot(Hct.fromInt(argbFromHex(seedHex)), dark, 0)
    const hex = (argb: number) => hexFromArgb(argb)
    const statusTone = dark ? 80 : 40
    const status = (seed: string) => hexFromArgb(TonalPalette.fromInt(argbFromHex(seed)).tone(statusTone))

    return {
        primary: hex(scheme.primary),
        'on-primary': hex(scheme.onPrimary),
        'primary-container': hex(scheme.primaryContainer),
        'on-primary-container': hex(scheme.onPrimaryContainer),
        secondary: hex(scheme.secondary),
        'on-secondary': hex(scheme.onSecondary),
        'secondary-container': hex(scheme.secondaryContainer),
        'on-secondary-container': hex(scheme.onSecondaryContainer),
        tertiary: hex(scheme.tertiary),
        'on-tertiary': hex(scheme.onTertiary),
        'tertiary-container': hex(scheme.tertiaryContainer),
        'on-tertiary-container': hex(scheme.onTertiaryContainer),
        error: hex(scheme.error),
        'on-error': hex(scheme.onError),
        'error-container': hex(scheme.errorContainer),
        'on-error-container': hex(scheme.onErrorContainer),
        background: hex(scheme.surface),
        'on-background': hex(scheme.onSurface),
        surface: hex(scheme.surface),
        'on-surface': hex(scheme.onSurface),
        'surface-dim': hex(scheme.surfaceDim),
        'surface-bright': hex(scheme.surfaceBright),
        'surface-container-lowest': hex(scheme.surfaceContainerLowest),
        'surface-container-low': hex(scheme.surfaceContainerLow),
        'surface-container': hex(scheme.surfaceContainer),
        'surface-container-high': hex(scheme.surfaceContainerHigh),
        'surface-container-highest': hex(scheme.surfaceContainerHighest),
        'surface-variant': hex(scheme.inverseSurface),
        'on-surface-variant': hex(scheme.inverseOnSurface),
        'md-on-surface-variant': hex(scheme.onSurfaceVariant),
        outline: hex(scheme.outline),
        'outline-variant': hex(scheme.outlineVariant),
        'inverse-primary': hex(scheme.inversePrimary),
        success: status(statusSeeds.success),
        warning: status(statusSeeds.warning),
        info: status(statusSeeds.info),
    }
}

export const lightTheme: ThemeDefinition = {
    dark: false,
    colors: {},
}

export const darkTheme: ThemeDefinition = {
    dark: true,
    colors: {},
}

// Selectable accent colors. `seed` is the MD3 source color; the light/dark
// schemes are both generated from it. `swatch` is only used for the picker UI.
export interface AccentPreset {
    id: string
    name: string
    seed: string
}

export const accentPresets: AccentPreset[] = [
    { id: 'pink', name: '樱粉', seed: '#F06292' },
    { id: 'blue', name: '天蓝', seed: '#1E88E5' },
    { id: 'purple', name: '雅紫', seed: '#7E57C2' },
    { id: 'teal', name: '青碧', seed: '#00897B' },
    { id: 'orange', name: '暖橙', seed: '#FB8C00' },
    { id: 'green', name: '草绿', seed: '#43A047' },
    { id: 'red', name: '绯红', seed: '#E53935' },
]

export const defaultAccentId = 'pink'
export const accentStorageKey = 'lolia.accent'

// 动态取色（莫奈）：种子色从桌面壁纸 / 系统强调色 / Android 系统动态配色提取，最近一次结果缓存在本地，启动时先用缓存避免闪烁
export const DYNAMIC_ACCENT_ID = 'dynamic'
const dynamicSeedStorageKey = 'lolia.dynamicSeed'

export type DynamicSeedSource = 'wallpaper' | 'accent' | 'system'

export interface DynamicSeed {
    seed: string
    source: DynamicSeedSource
}

export const readCachedDynamicSeed = (): DynamicSeed | null => {
    try {
        const raw = localStorage.getItem(dynamicSeedStorageKey)
        if (!raw) return null
        const parsed = JSON.parse(raw) as Partial<DynamicSeed>
        if (typeof parsed.seed === 'string' && /^#[0-9a-f]{6}$/i.test(parsed.seed)) {
            const source: DynamicSeedSource =
                parsed.source === 'accent' || parsed.source === 'system' ? parsed.source : 'wallpaper'
            return { seed: parsed.seed, source }
        }
    } catch {
        // ignore localStorage / JSON errors
    }
    return null
}

export const saveCachedDynamicSeed = (value: DynamicSeed): void => {
    try {
        localStorage.setItem(dynamicSeedStorageKey, JSON.stringify(value))
    } catch {
        // ignore localStorage errors
    }
}

export const findAccentPreset = (id: string): AccentPreset =>
    accentPresets.find((preset) => preset.id === id) ??
    accentPresets.find((preset) => preset.id === defaultAccentId) ??
    accentPresets[0]

// 解析强调色 id 对应的种子色；动态取色在尚无缓存时回退到默认预设
export const resolveAccentSeed = (id: string): string => {
    if (id === DYNAMIC_ACCENT_ID) {
        return readCachedDynamicSeed()?.seed ?? findAccentPreset(defaultAccentId).seed
    }
    return findAccentPreset(id).seed
}

export const readSavedAccentId = (): string => {
    try {
        const saved = localStorage.getItem(accentStorageKey)
        if (saved && (saved === DYNAMIC_ACCENT_ID || accentPresets.some((preset) => preset.id === saved))) {
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

type ThemeColorMap = { colors?: Record<string, unknown> }
export type ThemeColorMaps = { lightTheme?: ThemeColorMap; darkTheme?: ThemeColorMap }

// Regenerates the full MD3 color scheme for both themes from a seed color.
export const applySeedColors = (themes: ThemeColorMaps, seed: string): void => {
    if (themes.lightTheme) {
        themes.lightTheme.colors = Object.assign(themes.lightTheme.colors ?? {}, buildSchemeColors(seed, false))
    }
    if (themes.darkTheme) {
        themes.darkTheme.colors = Object.assign(themes.darkTheme.colors ?? {}, buildSchemeColors(seed, true))
    }
}

// Regenerates the full MD3 color scheme for both themes from the accent id.
export const applyAccentColors = (themes: ThemeColorMaps, id: string): void => {
    applySeedColors(themes, resolveAccentSeed(id))
}
