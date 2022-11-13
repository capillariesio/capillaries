<script context="module">
	export function handleResponse(responseJson, setWebapiDataFunc) {
		if (!!responseJson.error.msg) {
			console.log (responseJson.error.msg);
			setWebapiDataFunc(null, responseJson.error.msg);
		} else {
            setWebapiDataFunc(responseJson.data, null);
		}
	}
</script>
<script>
	export function rootLink(ksName) {
		return "/#/";
	}
	export function ksMatrixLink(ksName) {
		return "/#/ks/" + ksName + "/matrix";
	}
	export function ksRunNodeBatchHistoryLink(ksName, runId, nodeName) {
		return "/#/ks/" + ksName + "/run/" + runId + "/node/" + nodeName + "/batch_history";
	}
	export function runStatusToIcon(runStatus) {
		switch (runStatus) {
			case 0:
				return "i/schedule-svgrepo-com.svg" 
			case 1:
				return "i/running-run-svgrepo-com.svg" 
			case 2:
				return "i/finish-line-flag-svgrepo-com.svg" 
			case 3:
				return "i/error-alt-svgrepo-com.svg" 
			default:
				return "i/question-svgrepo-com.svg" 
		}
	}

	export function runStatusToText(runStatus) {
		switch (runStatus) {
			case 0:
				return "Not started" 
			case 1:
				return "Running" 
			case 2:
				return "Complete" 
			case 3:
				return "Stopped" 
			default:
				return "Unknown" 
		}
	}     

	export function nodeStatusToIcon(nodeStatus) {
		switch (nodeStatus) {
			case 0:
				return "i/schedule-svgrepo-com.svg" 
			case 1:
				return "i/running-run-svgrepo-com.svg" 
			case 2:
				return "i/success-svgrepo-com.svg" 
			case 3:
				return "i/error-svgrepo-com.svg" 
			case 104:
				return "i/error-alt-svgrepo-com.svg" 
			default:
				return "i/question-svgrepo-com.svg" 
		}
	} 

	export function nodeStatusToText(nodeStatus) {
		switch (nodeStatus) {
			case 0:
				return "Not started" 
			case 1:
				return "Running" 
			case 2:
				return "Success" 
			case 3:
				return "Failure" 
			case 104:
				return "Run stopped" 
			default:
				return "Unknown" 
		}
	}
</script>