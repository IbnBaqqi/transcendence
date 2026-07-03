import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
	plugins: [react(), tailwindcss()],
	server: {
		host: true,			// listen on 0.0.0.0 so the container is reachable
		port: 5173,
		proxy: {
			// browser calls /api/ ... -> forwarded to the backend container
			"/api": { target: "http://backend:8080", changeOrigin: true },
		},
	},
})
