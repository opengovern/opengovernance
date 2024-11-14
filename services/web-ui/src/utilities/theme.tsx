import { GithubComKaytuIoKaytuEnginePkgAuthApiTheme } from '../api/api'

export const parseTheme = (
    v: string
): GithubComKaytuIoKaytuEnginePkgAuthApiTheme => {
    switch (v) {
        case 'light':
            return GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeLight
        case 'dark':
            return GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeDark
        default:
            return GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeSystem
    }
}

export const currentTheme = () => {
    if (!('theme' in localStorage)) {
        return GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeLight
    }

    return parseTheme(localStorage.theme)
}

export const applyTheme = (v: GithubComKaytuIoKaytuEnginePkgAuthApiTheme) => {
    if (
        v === GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeDark ||
        (v === GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeSystem &&
            window.matchMedia('(prefers-color-scheme:dark)').matches)
    ) {
        document.documentElement.classList.add('dark')
    } else {
        document.documentElement.classList.remove('dark')
    }

    switch (v) {
        case GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeDark:
            localStorage.theme = 'dark'
            break
        case GithubComKaytuIoKaytuEnginePkgAuthApiTheme.ThemeLight:
            localStorage.theme = 'light'
            break
        default:
            localStorage.removeItem('theme')
    }
}
