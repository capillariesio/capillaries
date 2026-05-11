import { sveltekit } from '@sveltejs/kit/vite';
import { defineConfig } from 'vite';
import { visualizer } from 'rollup-plugin-visualizer';
import { execSync } from 'child_process'

const gitHash = execSync('git describe --tags --always').toString().trim();

console.log('In .env file VITE_WEBAPI_URL=$CAPI_WEBAPI_URL:', process.env.CAPI_WEBAPI_URL);
export default defineConfig({
	define: {
        __GIT_HASH__: JSON.stringify(gitHash),
    },
	server: {
		port: 8080,
		host: true // In Docker containers, "--host 0.0.0.0" may not work. So, tell Vite to bind to 0.0.0.0 here.
	},
	plugins: [
		sveltekit(),
		visualizer({
			emitFile: true,
			filename: 'stats.html'
		})
	],
	build: {
		rollupOptions: {
			output: {
				manualChunks: (id) => {
					return 'capillaries-ui';
				}
			}
		}
	}
});
