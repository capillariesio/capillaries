<script context="module">
	export function webapiUrl() {
		// If no Webapi url supplied, assume it's a dev environment and use our best guess
		const webapiUrlEnvVar = import.meta.env.VITE_WEBAPI_URL;
		return (!!webapiUrlEnvVar ? webapiUrlEnvVar : 'http://localhost:6543');
	}
	export function handleResponse(responseJson, setWebapiDataFunc) {
		if (!!responseJson.error.msg) {
			console.log(responseJson.error.msg);
			setWebapiDataFunc(null, responseJson.error.msg);
		} else {
			setWebapiDataFunc(responseJson.data, null);
		}
	}
	export function nodeStatusToColor(status) {
		switch (status) {
			case 1:
				return 'Gray'; //"Started"
			case 2:
				return 'DarkGreen'; //"Success"
			case 3:
				return 'OrangeRed'; // "Failure"
			case 104:
				return 'Maroon'; // "Run stopped"
			default:
				return 'Black'; //"Unknown"
		}
	}
</script>

<script>
	export function rootLink() {
		return '/#/';
	}
	export function ksMatrixLink(ksName) {
		return '/#/ks/' + ksName + '/matrix';
	}
	export function ksRunNodeBatchHistoryLink(ksName, runId, nodeName) {
		return '/#/ks/' + ksName + '/run/' + runId + '/node/' + nodeName + '/batch_history';
	}
	export function ksRunNodeHistoryLink(ksName, runId) {
		return '/#/ks/' + ksName + '/run/' + runId + '/node_history';
	}
	export function runStatusToIconStatic(runStatus) {
		switch (runStatus) {
			case 1:
				return 'i/black-run.svg';
			case 2:
				return 'i/black-finish.svg';
			case 3:
				return 'i/black-stop.svg';
			default:
				return 'i/black-question.svg';
		}
	}
	export function runStatusToIconLink(runStatus) {
		switch (runStatus) {
			case 1:
				return 'i/blue-run.svg';
			case 2:
				return 'i/blue-finish.svg';
			case 3:
				return 'i/blue-stop.svg';
			default:
				return 'i/blue-question.svg';
		}
	}

	export function runStatusToText(runStatus) {
		switch (runStatus) {
			case 1:
				return 'Running';
			case 2:
				return 'Complete';
			case 3:
				return 'Stopped/Invalidated';
			default:
				return 'Unknown';
		}
	}

	export function nodeStatusToIconStatic(nodeStatus) {
		switch (nodeStatus) {
			case 1:
				return 'i/black-run.svg';
			case 2:
				return 'i/black-check.svg';
			case 3:
				return 'i/black-triangle.svg';
			case 104:
				return 'i/black-stop.svg';
			default:
				return 'i/black-question.svg';
		}
	}

	export function nodeStatusToIconLink(nodeStatus) {
		switch (nodeStatus) {
			case 1:
				return 'i/blue-run.svg';
			case 2:
				return 'i/blue-check.svg';
			case 3:
				return 'i/blue-triangle.svg';
			case 104:
				return 'i/blue-stop.svg';
			default:
				return 'i/blue-question.svg';
		}
	}

	export function nodeStatusToText(nodeStatus) {
		switch (nodeStatus) {
			case 1:
				return 'Started';
			case 2:
				return 'Success';
			case 3:
				return 'Failure';
			case 104:
				return 'Run stopped';
			default:
				return 'Unknown';
		}
	}
</script>
