check "prometheus_node_exporter" {
	data "external" "check_prometheus_node_exporter_url" {
		program = ["bash", "${path.module}/check_url.sh", local.prometheus_node_exporter_url]
	}
	assert {
		condition     = data.external.check_prometheus_node_exporter_url.result.url_exists == "true"
		error_message = format("prometheus node exporter %s is not accessible", local.prometheus_node_exporter_url)
	}
}

check "prometheus_jmx_exporter" {
	data "external" "check_prometheus_jmx_exporter_url" {
		program = ["bash", "${path.module}/check_url.sh", local.prometheus_jmx_exporter_url]
	}
	assert {
		condition     = data.external.check_prometheus_jmx_exporter_url.result.url_exists == "true"
		error_message = format("prometheus jmx exporter %s is not accessible", local.prometheus_jmx_exporter_url)
	}
}

check "prometheus_server" {
	data "external" "check_prometheus_server_url" {
		program = ["bash", "${path.module}/check_url.sh", local.prometheus_server_url]
	}
	assert {
		condition     = data.external.check_prometheus_server_url.result.url_exists == "true"
		error_message = format("prometheus server %s is not accessible", local.prometheus_server_url)
	}
}

check "rabbitmq_erlang" {
	data "external" "check_rabbitmq_erlang_url" {
		program = ["bash", "${path.module}/check_url.sh", local.rabbitmq_erlang_url]
	}
	assert {
		condition     = data.external.check_rabbitmq_erlang_url.result.url_exists == "true"
		error_message = format("rabbitmq erlang %s is not accessible", local.rabbitmq_erlang_url)
	}
}

check "rabbitmq_server" {
	data "external" "check_rabbitmq_server_url" {
		program = ["bash", "${path.module}/check_url.sh", local.rabbitmq_server_url]
	}
	assert {
		condition     = data.external.check_rabbitmq_server_url.result.url_exists == "true"
		error_message = format("rabbitmq server %s is not accessible", local.rabbitmq_server_url)
	}
}

check "activemq_classic_server" {
	data "external" "check_activemq_classic_server_url" {
		program = ["bash", "${path.module}/check_url.sh", local.activemq_classic_server_url]
	}
	assert {
		condition     = data.external.check_activemq_classic_server_url.result.url_exists == "true"
		error_message = format("activemq_classic server %s is not accessible", local.activemq_classic_server_url)
	}
}

check "activemq_artemis_server" {
	data "external" "check_activemq_artemis_server_url" {
		program = ["bash", "${path.module}/check_url.sh", local.activemq_artemis_server_url]
	}
	assert {
		condition     = data.external.check_activemq_artemis_server_url.result.url_exists == "true"
		error_message = format("activemq_artemis server %s is not accessible", local.activemq_artemis_server_url)
	}
}
