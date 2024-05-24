import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';

console.log("In .env file VITE_WEBAPI_URL=$CAPI_WEBAPI_URL:", process.env.CAPI_WEBAPI_URL)
export default defineConfig({
	server: {
		port: 8080,
		host: true,   // In Docker containers, "--host 0.0.0.0" may not work. So, tell Vite to bind to 0.0.0.0 here.
		proxy: {
			'/': {
			  target: 'https://localhost:44305',
			  changeOrigin: true,
			  secure: false,      
			  ws: true,
			  configure: (proxy, _options) => {
				proxy.on('error', (err, _req, _res) => {
				  console.log('proxy error', err);
				});
				proxy.on('proxyReq', (proxyReq, req, _res) => {
				  console.log('Sending Request to the Target:', req.method, req.url);
				});
				proxy.on('proxyRes', (proxyRes, req, _res) => {
				  console.log('Received Response from the Target:', proxyRes.statusCode, req.url);
				});
			  },
			}
		  }
	},
	plugins: [
		sveltekit()
	],
	build: {
		minify: true
	},
	esbuild: {
		minifyIdentifiers: true
	}	
});
