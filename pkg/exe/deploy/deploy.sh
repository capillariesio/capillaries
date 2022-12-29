go run deploy.go create_instances bastion,rabbitmq,daemon01
go run deploy.go upload_files up_daemon_env_config,up_daemon_binary,up_webapi_env_config,up_webapi_binary -verbose
go run deploy.go setup_services bastion -verbose
go run deploy.go create_instances cass01,cass02,cass03 -verbose
go run deploy.go setup_services cass01,cass02,cass03 -verbose
rabbitmq,
go run deploy.go setup_services daemon01

# go run deploy.go delete_instances bastion,rabbitmq,daemon01
# go run deploy.go download_files down_capi_logs