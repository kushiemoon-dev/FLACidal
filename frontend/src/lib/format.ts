import { get } from 'svelte/store';
import { locale as activeLocale } from './i18n';

export function formatNumber(n: number, locale?: string): string {
    return n.toLocaleString(locale ?? get(activeLocale));
}

export function formatDate(date: Date, options: Intl.DateTimeFormatOptions, locale?: string): string {
    return date.toLocaleDateString(locale ?? get(activeLocale), options);
}

export function formatBytes(bytes: number): string {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

export function formatDuration(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
}
