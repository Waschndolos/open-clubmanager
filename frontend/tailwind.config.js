export default {
    darkMode: "class", // Dark mode activated with the 'dark' class
    content: [
      "./index.html",
      "./src/**/*.{js,ts,jsx,tsx}",
    ],
    theme: {
      extend: {
        colors: {
          dark: {
            50: "#181926",   // Main background
            100: "#23243a",  // Sidebar and cards
            200: "#23243a",  // Cards
            300: "#31324b",  // Soft borders
            400: "#393a5a",  // Light hover
            accent: "#8ecae6", // Accent blue
            accent2: "#ffb703", // Accent yellow
          },
        },
        boxShadow: {
          'soft-dark': '0 4px 20px rgba(0, 0, 0, 0.4)',
        },
      },
    },
    plugins: [],
}