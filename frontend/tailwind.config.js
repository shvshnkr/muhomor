/** @type {import('tailwindcss').Config} */

export default {

  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],

  darkMode: "class",

  theme: {

    extend: {

      colors: {

        bg: "var(--bg-base)",

        raised: "var(--bg-raised)",

        surface: {

          DEFAULT: "var(--surface-1)",

          2: "var(--surface-2)",

          3: "var(--surface-3)",

          glass: "var(--surface-glass)",

        },

        fg: "var(--text-primary)",

        muted: "var(--text-secondary)",

        accent: {

          DEFAULT: "var(--accent)",

          hover: "var(--accent-hover)",

        },

        success: "var(--success)",

        warn: "var(--warn)",

        error: "var(--error)",

        border: "var(--border-subtle)",

        "border-strong": "var(--border-strong)",

      },

      borderRadius: {

        sm: "6px",

        md: "10px",

        lg: "12px",

        xl: "var(--radius-xl)",

        "2xl": "var(--radius-hero)",

        card: "var(--radius-card)",

      },

      fontSize: {

        display: ["28px", { lineHeight: "1.2", fontWeight: "600" }],

        title: ["18px", { lineHeight: "1.3", fontWeight: "600" }],

        body: ["14px", { lineHeight: "1.45" }],

        caption: ["12px", { lineHeight: "1.4" }],

        stat: ["22px", { lineHeight: "1.2", fontWeight: "600" }],

      },

      boxShadow: {

        card: "var(--shadow-card)",

        "card-inset": "var(--shadow-card-inset)",

        glow: "var(--shadow-glow)",

        "power-idle": "var(--shadow-power-idle)",

        "power-accent": "var(--shadow-power-accent)",

        "power-danger": "var(--shadow-power-danger)",

      },

      minHeight: {

        connect: "44px",

      },

    },

  },

  plugins: [],

};

