import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import replace from 'vite-plugin-filter-replace';

const suggestedWebapiUrl = !!process.env.CAPILLARIES_WEBAPI_URL
	? process.env.CAPILLARIES_WEBAPI_URL
	: 'http://localhost:6543';

export default defineConfig({
	server: {
		port: 8080
	},
	plugins: [
		sveltekit(),
		replace([
			{
				filter: /\.js$/,
				replace: {
					from: 'http://localhost:6543',
					to: suggestedWebapiUrl
				}
			}
		])
	],
	build: {
		minify: true
	},
	esbuild: {
		minifyIdentifiers: true
	}	
});
