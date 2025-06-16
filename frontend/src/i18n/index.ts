import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import de from './de.json';
import en from './en.json';
import fr from './fr.json';

i18n
    .use(initReactI18next)
    .init({
        resources: {
            de: { translation: de },
            en: { translation: en },
            fr: { translation: fr },
        },
        lng: window.userpreference?.get('1', 'language') || 'de',
        fallbackLng: 'en',
        interpolation: {
            escapeValue: false,
        },
        detection: {
          order: ['localStorage', 'navigator'],
          caches: ['localStorage'],
        },
    });

export default i18n;
