/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // 品牌色
        brand: {
          500: '#6366F1',
          600: '#4F46E5',
        },
        // 成功色
        success: '#10B981',
        // 警告色
        warning: '#F59E0B',
        // 错误色
        error: '#EF4444',
        // 浅色主题
        light: {
          bg: '#FAFAFA',
          card: '#FFFFFF',
          border: '#E5E7EB',
          text: '#111827',
          textSecondary: '#6B7280',
        },
        // 深色主题
        dark: {
          bg: '#0F172A',
          card: '#1E293B',
          border: '#334155',
          text: '#F1F5F9',
          textSecondary: '#94A3B8',
        },
      },
      fontFamily: {
        sans: ['-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace'],
      },
      boxShadow: {
        card: '0 1px 3px rgba(0,0,0,0.1), 0 1px 2px rgba(0,0,0,0.06)',
        cardHover: '0 10px 15px -3px rgba(0,0,0,0.1), 0 4px 6px -2px rgba(0,0,0,0.05)',
        modal: '0 25px 50px -12px rgba(0,0,0,0.25)',
      },
      borderRadius: {
        'card': '8px',
        'card-lg': '12px',
        'pill': '9999px',
      },
      animation: {
        'pulse-slow': 'pulse 2s cubic-bezier(0.4, 0, 0.6, 1) infinite',
      },
    },
  },
  plugins: [],
}
