/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,js,ts,jsx,tsx}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // SSXZ interaction color: a neutral graphite scale shared by every product surface.
        primary: {
          50: '#f7f8f8',
          100: '#eceeef',
          200: '#d8dcdf',
          300: '#b8bec4',
          400: '#89929b',
          500: '#606b76',
          600: '#48515b',
          700: '#363d45',
          800: '#252a30',
          900: '#181b1f',
          950: '#0e1012'
        },
        // Detail accent: restrained cool steel for links, focus and data emphasis.
        accent: {
          50: '#f5f7f8',
          100: '#e8ecef',
          200: '#d5dce1',
          300: '#b7c1c9',
          400: '#8f9da9',
          500: '#697987',
          600: '#556273',
          700: '#444f5d',
          800: '#343d48',
          900: '#29313a',
          950: '#171c22'
        },
        // Neutral dark surfaces. Avoid blue/slate casts in dark mode.
        dark: {
          50: '#fafafa',
          100: '#f4f4f5',
          200: '#e4e4e7',
          300: '#d4d4d8',
          400: '#a1a1aa',
          500: '#71717a',
          600: '#52525b',
          700: '#3f3f46',
          800: '#27272a',
          900: '#18181b',
          950: '#09090b'
        }
      },
      fontFamily: {
        sans: [
          'system-ui',
          '-apple-system',
          'BlinkMacSystemFont',
          'Segoe UI',
          'Roboto',
          'Helvetica Neue',
          'Arial',
          'PingFang SC',
          'Hiragino Sans GB',
          'Microsoft YaHei',
          'sans-serif'
        ],
        mono: ['ui-monospace', 'SFMono-Regular', 'Menlo', 'Monaco', 'Consolas', 'monospace']
      },
      boxShadow: {
        glass: '0 8px 32px rgba(0, 0, 0, 0.08)',
        'glass-sm': '0 4px 16px rgba(0, 0, 0, 0.06)',
        glow: '0 0 0 1px rgba(85, 98, 115, 0.18)',
        'glow-lg': '0 0 0 1px rgba(85, 98, 115, 0.24)',
        card: '0 1px 2px rgba(0, 0, 0, 0.24)',
        'card-hover': '0 12px 32px rgba(0, 0, 0, 0.24)',
        'inner-glow': 'inset 0 1px 0 rgba(255, 255, 255, 0.04)'
      },
      backgroundImage: {
        'gradient-radial': 'radial-gradient(var(--tw-gradient-stops))',
        'gradient-primary': 'linear-gradient(180deg, #30343a 0%, #181a1e 100%)',
        'gradient-dark': 'linear-gradient(135deg, #27272a 0%, #18181b 100%)',
        'gradient-glass':
          'linear-gradient(135deg, rgba(255,255,255,0.1) 0%, rgba(255,255,255,0.05) 100%)',
        'mesh-gradient': 'linear-gradient(180deg, #18191c 0%, #111214 100%)'
      },
      animation: {
        'fade-in': 'fadeIn 0.3s ease-out',
        'slide-up': 'slideUp 0.3s ease-out',
        'slide-down': 'slideDown 0.3s ease-out',
        'slide-in-right': 'slideInRight 0.3s ease-out',
        'scale-in': 'scaleIn 0.2s ease-out',
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        shimmer: 'shimmer 2s linear infinite',
        glow: 'none'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideDown: {
          '0%': { opacity: '0', transform: 'translateY(-10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        },
        slideInRight: {
          '0%': { opacity: '0', transform: 'translateX(20px)' },
          '100%': { opacity: '1', transform: 'translateX(0)' }
        },
        scaleIn: {
          '0%': { opacity: '0', transform: 'scale(0.95)' },
          '100%': { opacity: '1', transform: 'scale(1)' }
        },
        shimmer: {
          '0%': { backgroundPosition: '-200% 0' },
          '100%': { backgroundPosition: '200% 0' }
        },
        glow: {}
      },
      backdropBlur: {
        xs: '2px'
      },
      borderRadius: {
        '4xl': '2rem'
      }
    }
  },
  plugins: []
}
